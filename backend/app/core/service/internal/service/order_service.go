package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data"

	"github.com/tx7do/go-crud/viewer"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/order"

	orderV1 "github.com/Mikdad-Force/Ghost/api/gen/go/order/service/v1"
	inventoryV1 "github.com/Mikdad-Force/Ghost/api/gen/go/inventory/service/v1"
)

// OrderService 订单管理服务：Marketplace / 手动补录 → 确认 → 履约（生成出库拣货单）→ 完结。
//
// 状态机：PENDING → CONFIRMED → FULFILLING → COMPLETED
// PENDING / CONFIRMED → CANCELLED（终态）。
// Fulfill：CONFIRMED → FULFILLING，生成 OUTGOING 拣货单（partner_code=客户名）。
type OrderService struct {
	orderV1.UnimplementedOrderServiceServer

	log *log.Helper

	orderRepo        *data.OrderRepo
	stockPickingRepo *data.StockPickingRepo
	stockPickingService *StockPickingService
}

func NewOrderService(
	ctx *bootstrap.Context,
	orderRepo *data.OrderRepo,
	stockPickingRepo *data.StockPickingRepo,
	stockPickingService *StockPickingService,
) *OrderService {
	return &OrderService{
		log:                 ctx.NewLoggerHelper("order/service/core-service"),
		orderRepo:           orderRepo,
		stockPickingRepo:    stockPickingRepo,
		stockPickingService: stockPickingService,
	}
}

func (s *OrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*orderV1.ListOrderResponse, error) {
	return s.orderRepo.List(ctx, req)
}

func (s *OrderService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*orderV1.CountOrderResponse, error) {
	count, err := s.orderRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &orderV1.CountOrderResponse{Count: uint64(count)}, nil
}

func (s *OrderService) Get(ctx context.Context, req *orderV1.GetOrderRequest) (*orderV1.Order, error) {
	return s.orderRepo.Get(ctx, req)
}

func (s *OrderService) GetByNumber(ctx context.Context, req *orderV1.GetOrderByNumberRequest) (*orderV1.Order, error) {
	if req == nil || req.GetNumber() == "" {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}
	return s.orderRepo.GetByNumber(ctx, req.GetNumber())
}

func (s *OrderService) Create(ctx context.Context, req *orderV1.CreateOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}
	if len(req.Data.GetLines()) == 0 {
		return nil, orderV1.ErrorBadRequest("order lines are required")
	}

	// 客户名/联系人至少填一个（手动单或外部单）
	if req.Data.GetCustomerName() == "" && req.Data.GetSourceRef() == "" {
		return nil, orderV1.ErrorBadRequest("customer_name or source_ref is required")
	}

	// 客户端不传单号（服务端生成），忽略 status（创建恒为 PENDING）
	req.Data.OrderNumber = nil
	req.Data.Status = nil

	if _, err := s.orderRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrderService) Update(ctx context.Context, req *orderV1.UpdateOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	cur, err := s.orderRepo.Get(ctx, &orderV1.GetOrderRequest{QueryBy: &orderV1.GetOrderRequest_Id{Id: req.GetId()}})
	if err != nil {
		return nil, err
	}
	if cur.GetStatus() != orderV1.OrderStatus_ORDER_PENDING {
		return nil, orderV1.ErrorConflict("only pending orders can be updated")
	}

	if _, err = s.orderRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrderService) Delete(ctx context.Context, req *orderV1.DeleteOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	cur, err := s.orderRepo.Get(ctx, &orderV1.GetOrderRequest{QueryBy: &orderV1.GetOrderRequest_Id{Id: req.GetId()}})
	if err != nil {
		return nil, err
	}
	st := cur.GetStatus()
	if st != orderV1.OrderStatus_ORDER_PENDING && st != orderV1.OrderStatus_ORDER_CANCELLED {
		return nil, orderV1.ErrorConflict("only pending or cancelled orders can be deleted")
	}

	if err = s.orderRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrderService) Confirm(ctx context.Context, req *orderV1.ConfirmOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.orderRepo.SetStatus(ctx, req.GetId(), order.StatusPending, order.StatusConfirmed, nil); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrderService) Cancel(ctx context.Context, req *orderV1.CancelOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	// 可取消状态：PENDING / CONFIRMED（FULFILLING 之后不再允许取消，避免已发货订单被误取消）
	cur, err := s.orderRepo.Get(ctx, &orderV1.GetOrderRequest{QueryBy: &orderV1.GetOrderRequest_Id{Id: req.GetId()}})
	if err != nil {
		return nil, err
	}
	st := cur.GetStatus()
	if st != orderV1.OrderStatus_ORDER_PENDING && st != orderV1.OrderStatus_ORDER_CONFIRMED {
		return nil, orderV1.ErrorConflict("only pending or confirmed orders can be cancelled")
	}

	from := order.StatusPending
	if st == orderV1.OrderStatus_ORDER_CONFIRMED {
		from = order.StatusConfirmed
	}
	if err = s.orderRepo.SetStatus(ctx, req.GetId(), from, order.StatusCancelled, nil); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Fulfill 生成出库拣货单（OUTGOING，M2 直发链）：source = 出库仓库接收位置，
// dest = 租户客户虚拟位置。partner_code = 客户名称（无客户 → 订单号）。
func (s *OrderService) Fulfill(ctx context.Context, req *orderV1.FulfillOrderRequest) (*orderV1.FulfillOrderResponse, error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	ord, err := s.orderRepo.Get(ctx, &orderV1.GetOrderRequest{QueryBy: &orderV1.GetOrderRequest_Id{Id: req.GetId()}})
	if err != nil {
		return nil, err
	}
	if ord.GetStatus() != orderV1.OrderStatus_ORDER_CONFIRMED {
		return nil, orderV1.ErrorConflict("only confirmed orders can be fulfilled")
	}

	lines := ord.GetLines()
	if len(lines) == 0 {
		return nil, orderV1.ErrorBadRequest("order has no lines")
	}

	moves := make([]*inventoryV1.StockMove, 0, len(lines))
	for _, l := range lines {
		moves = append(moves, &inventoryV1.StockMove{
			ProductCode:      l.SkuCode,
			PlannedQuantity:  l.Quantity,
		})
	}

	warehouse := ord.GetWarehouseCode()
	if warehouse == "" {
		warehouse = "MAIN"
	}

	partner := ord.GetCustomerName()
	if partner == "" {
		partner = ord.GetOrderNumber()
	}

	picking, err := s.stockPickingService.createOutgoingPicking(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			FromWarehouseCode: &warehouse,
			PartnerCode:       &partner,
			Moves:             moves,
			Remark:            ord.Remark,
		},
	})
	if err != nil {
		return nil, err
	}

	if err = s.orderRepo.SetStatus(ctx, ord.GetId(), order.StatusConfirmed, order.StatusFulfilling, picking.Id); err != nil {
		return nil, err
	}

	return &orderV1.FulfillOrderResponse{PickingId: picking.GetId()}, nil
}

func (s *OrderService) Complete(ctx context.Context, req *orderV1.CompleteOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.orderRepo.SetStatus(ctx, req.GetId(), order.StatusFulfilling, order.StatusCompleted, nil); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ImportMarketplace 按 (tenant, source, source_ref) 幂等 upsert 外部订单。
// 同一平台订单重复同步：已存在 → skipped；不存在 → 创建为 PENDING。
func (s *OrderService) ImportMarketplace(ctx context.Context, req *orderV1.ImportMarketplaceRequest) (*orderV1.ImportMarketplaceResponse, error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}
	source := req.GetSource()
	if source == orderV1.OrderSource_ORDER_SOURCE_MANUAL {
		return nil, orderV1.ErrorBadRequest("source must not be MANUAL for marketplace import")
	}

	entSource := sourceToEnt(source)
	if entSource == nil {
		return nil, orderV1.ErrorBadRequest("unsupported marketplace source")
	}

	resp := &orderV1.ImportMarketplaceResponse{}
	for _, o := range req.GetOrders() {
		if o == nil {
			resp.Failed++
			continue
		}

		srcRef := o.GetSourceRef()
		if srcRef == "" {
			resp.Failed++
			continue
		}

		tenantID := uint32(0)
		if vc, exist := viewer.FromContext(ctx); exist && vc != nil {
			tenantID = uint32(vc.TenantID())
		}
		if o.TenantId != nil {
			tenantID = o.GetTenantId()
		}

		existing, err := s.orderRepo.FindBySourceRef(ctx, tenantID, *entSource, srcRef)
		if err != nil {
			resp.Failed++
			continue
		}
		if existing != nil {
			resp.Skipped++
			continue
		}

		// 订单号/状态由服务端生成：忽略客户端传入的 order_number/status
		data := &orderV1.Order{
			Source:           source.Enum(),
			SourceRef:        o.SourceRef,
			SourceShop:       o.SourceShop,
			CustomerName:     o.CustomerName,
			CustomerPhone:    o.CustomerPhone,
			ShippingAddress:  o.ShippingAddress,
			WarehouseCode:    o.WarehouseCode,
			Lines:            o.Lines,
			TotalAmountCents: o.TotalAmountCents,
			Currency:         o.Currency,
			OrderedAt:        o.OrderedAt,
			Remark:           o.Remark,
			TenantId:         &tenantID,
		}

		if _, err = s.orderRepo.Create(ctx, &orderV1.CreateOrderRequest{Data: data}); err != nil {
			resp.Failed++
			continue
		}

		resp.Created++
	}

	return resp, nil
}

func sourceToEnt(s orderV1.OrderSource) *order.Source {
	switch s {
	case orderV1.OrderSource_ORDER_SOURCE_MANUAL:
		v := order.SourceManual
		return &v
	case orderV1.OrderSource_ORDER_SOURCE_SHOPEE:
		v := order.SourceShopee
		return &v
	case orderV1.OrderSource_ORDER_SOURCE_TOKOPEDIA:
		v := order.SourceTokopedia
		return &v
	case orderV1.OrderSource_ORDER_SOURCE_LAZADA:
		v := order.SourceLazada
		return &v
	case orderV1.OrderSource_ORDER_SOURCE_TIKTOK:
		v := order.SourceTikTok
		return &v
	case orderV1.OrderSource_ORDER_SOURCE_OTHER:
		v := order.SourceOther
		return &v
	default:
		return nil
	}
}

