package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/order"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/orderline"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/predicate"

	orderV1 "github.com/Mikdad-Force/Ghost/api/gen/go/order/service/v1"
)

// OrderRepo 订单仓储：单据 CRUD + Marketplace 幂等导入 + 履约状态流转。
type OrderRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	statusConverter *mapper.EnumTypeConverter[orderV1.OrderStatus, order.Status]
	sourceConverter *mapper.EnumTypeConverter[orderV1.OrderSource, order.Source]

	mapper *mapper.CopierMapper[orderV1.Order, ent.Order]

	repository *entCrud.Repository[
		ent.OrderQuery, ent.OrderSelect,
		ent.OrderCreate, ent.OrderCreateBulk,
		ent.OrderUpdate, ent.OrderUpdateOne,
		ent.OrderDelete,
		predicate.Order,
		orderV1.Order, ent.Order,
	]
}

func NewOrderRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *OrderRepo {
	repo := &OrderRepo{
		log:       ctx.NewLoggerHelper("order/repo/core-service"),
		entClient: entClient,
	}
	repo.init()
	return repo
}

func (r *OrderRepo) init() {
	statusNames := map[int32]string{
		int32(orderV1.OrderStatus_ORDER_PENDING):    "PENDING",
		int32(orderV1.OrderStatus_ORDER_CONFIRMED):  "CONFIRMED",
		int32(orderV1.OrderStatus_ORDER_FULFILLING): "FULFILLING",
		int32(orderV1.OrderStatus_ORDER_COMPLETED):  "COMPLETED",
		int32(orderV1.OrderStatus_ORDER_CANCELLED):  "CANCELLED",
	}
	statusValues := map[string]int32{
		"PENDING":    int32(orderV1.OrderStatus_ORDER_PENDING),
		"CONFIRMED":  int32(orderV1.OrderStatus_ORDER_CONFIRMED),
		"FULFILLING": int32(orderV1.OrderStatus_ORDER_FULFILLING),
		"COMPLETED":  int32(orderV1.OrderStatus_ORDER_COMPLETED),
		"CANCELLED":  int32(orderV1.OrderStatus_ORDER_CANCELLED),
	}
	r.statusConverter = mapper.NewEnumTypeConverter[orderV1.OrderStatus, order.Status](statusNames, statusValues)

	sourceNames := map[int32]string{
		int32(orderV1.OrderSource_ORDER_SOURCE_MANUAL):    "MANUAL",
		int32(orderV1.OrderSource_ORDER_SOURCE_SHOPEE):    "SHOPEE",
		int32(orderV1.OrderSource_ORDER_SOURCE_TOKOPEDIA): "TOKOPEDIA",
		int32(orderV1.OrderSource_ORDER_SOURCE_LAZADA):    "LAZADA",
		int32(orderV1.OrderSource_ORDER_SOURCE_TIKTOK):    "TIKTOK",
		int32(orderV1.OrderSource_ORDER_SOURCE_OTHER):     "OTHER",
	}
	sourceValues := map[string]int32{
		"MANUAL":    int32(orderV1.OrderSource_ORDER_SOURCE_MANUAL),
		"SHOPEE":    int32(orderV1.OrderSource_ORDER_SOURCE_SHOPEE),
		"TOKOPEDIA": int32(orderV1.OrderSource_ORDER_SOURCE_TOKOPEDIA),
		"LAZADA":    int32(orderV1.OrderSource_ORDER_SOURCE_LAZADA),
		"TIKTOK":    int32(orderV1.OrderSource_ORDER_SOURCE_TIKTOK),
		"OTHER":     int32(orderV1.OrderSource_ORDER_SOURCE_OTHER),
	}
	r.sourceConverter = mapper.NewEnumTypeConverter[orderV1.OrderSource, order.Source](sourceNames, sourceValues)

	r.mapper = mapper.NewCopierMapper[orderV1.Order, ent.Order]()

	r.repository = entCrud.NewRepository[
		ent.OrderQuery, ent.OrderSelect,
		ent.OrderCreate, ent.OrderCreateBulk,
		ent.OrderUpdate, ent.OrderUpdateOne,
		ent.OrderDelete,
		predicate.Order,
		orderV1.Order, ent.Order,
	](r.mapper)

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.sourceConverter.NewConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func orderStatusToProto(s order.Status) *orderV1.OrderStatus {
	switch s {
	case order.StatusPending:
		return orderV1.OrderStatus_ORDER_PENDING.Enum()
	case order.StatusConfirmed:
		return orderV1.OrderStatus_ORDER_CONFIRMED.Enum()
	case order.StatusFulfilling:
		return orderV1.OrderStatus_ORDER_FULFILLING.Enum()
	case order.StatusCompleted:
		return orderV1.OrderStatus_ORDER_COMPLETED.Enum()
	case order.StatusCancelled:
		return orderV1.OrderStatus_ORDER_CANCELLED.Enum()
	default:
		return nil
	}
}

func orderSourceToProto(s order.Source) *orderV1.OrderSource {
	switch s {
	case order.SourceManual:
		return orderV1.OrderSource_ORDER_SOURCE_MANUAL.Enum()
	case order.SourceShopee:
		return orderV1.OrderSource_ORDER_SOURCE_SHOPEE.Enum()
	case order.SourceTokopedia:
		return orderV1.OrderSource_ORDER_SOURCE_TOKOPEDIA.Enum()
	case order.SourceLazada:
		return orderV1.OrderSource_ORDER_SOURCE_LAZADA.Enum()
	case order.SourceTikTok:
		return orderV1.OrderSource_ORDER_SOURCE_TIKTOK.Enum()
	case order.SourceOther:
		return orderV1.OrderSource_ORDER_SOURCE_OTHER.Enum()
	default:
		return nil
	}
}

func (r *OrderRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Order.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, orderV1.ErrorInternalServerError("query count failed")
	}
	return count, nil
}

func (r *OrderRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (ret *orderV1.ListOrderResponse, err error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Order.Query()

	ret2, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret2 == nil {
		return &orderV1.ListOrderResponse{Total: 0, Items: nil}, nil
	}

	r.backfillStatusSource(ctx, ret2.Items)
	if err = r.attachLines(ctx, ret2.Items); err != nil {
		return nil, err
	}

	return &orderV1.ListOrderResponse{
		Total: ret2.Total,
		Items: ret2.Items,
	}, nil
}

func (r *OrderRepo) Get(ctx context.Context, req *orderV1.GetOrderRequest) (dto *orderV1.Order, err error) {
	if req == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Order.Query()

	var whereCond []func(s *sql.Selector)
	if id := req.GetId(); id != 0 {
		whereCond = append(whereCond, order.IDEQ(id))
	}

	dto, err = r.repository.Get(ctx, builder, nil, whereCond...)
	if err != nil {
		return nil, err
	}

	r.backfillStatusSource(ctx, []*orderV1.Order{dto})
	if err = r.attachLines(ctx, []*orderV1.Order{dto}); err != nil {
		return nil, err
	}

	return dto, nil
}

func (r *OrderRepo) GetByNumber(ctx context.Context, number string) (dto *orderV1.Order, err error) {
	if number == "" {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Order.Query().
		Where(order.OrderNumberEQ(number))

	dto, err = r.repository.Get(ctx, builder, nil)
	if err != nil {
		return nil, err
	}

	r.backfillStatusSource(ctx, []*orderV1.Order{dto})
	if err = r.attachLines(ctx, []*orderV1.Order{dto}); err != nil {
		return nil, err
	}

	return dto, nil
}

// backfillStatusSource 回填 ent 的 Nillable 枚举字段（status/source）。
func (r *OrderRepo) backfillStatusSource(ctx context.Context, items []*orderV1.Order) {
	if len(items) == 0 {
		return
	}

	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.GetId())
	}

	rows, err := r.entClient.Client().Order.Query().
		Where(order.IDIn(ids...)).
		Select(order.FieldID, order.FieldStatus, order.FieldSource, order.FieldOrderedAt, order.FieldPickingID, order.FieldTotalAmountCents, order.FieldCurrency).
		All(ctx)
	if err != nil {
		r.log.Errorf("backfill order status failed: %s", err.Error())
		return
	}

	byID := make(map[uint32]*ent.Order, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	for _, it := range items {
		row, ok := byID[it.GetId()]
		if !ok {
			continue
		}
		if row.Status != nil {
			it.Status = orderStatusToProto(*row.Status)
		}
		if row.Source != nil {
			it.Source = orderSourceToProto(*row.Source)
		}
		if row.OrderedAt != nil && it.OrderedAt == nil {
			it.OrderedAt = timestamppb.New(*row.OrderedAt)
		}
		if row.TotalAmountCents != nil && it.TotalAmountCents == nil {
			it.TotalAmountCents = row.TotalAmountCents
		}
		if row.Currency != nil && it.Currency == nil {
			it.Currency = row.Currency
		}
		if row.PickingID != nil {
			it.PickingId = row.PickingID
		}
	}
}

// attachLines 批量装配明细（按 order_id 一次查询，按单分组回填）。
func (r *OrderRepo) attachLines(ctx context.Context, items []*orderV1.Order) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.GetId())
	}

	lines, err := r.entClient.Client().OrderLine.Query().
		Where(orderline.OrderIDIn(ids...)).
		Order(ent.Asc(orderline.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query order lines failed: %s", err.Error())
		return orderV1.ErrorInternalServerError("query order lines failed")
	}

	byOrder := make(map[uint32][]*orderV1.OrderLine, len(items))
	for _, l := range lines {
		dto := r.lineToDTO(l)
		tid := derefUint32(l.OrderID)
		byOrder[tid] = append(byOrder[tid], dto)
	}
	for _, it := range items {
		it.Lines = append(it.Lines, byOrder[it.GetId()]...)
	}

	return nil
}

func (r *OrderRepo) lineToDTO(t *ent.OrderLine) *orderV1.OrderLine {
	dto := &orderV1.OrderLine{
		Id:              &t.ID,
		OrderId:         t.OrderID,
		SkuCode:         t.SkuCode,
		ProductName:     t.ProductName,
		Quantity:        t.Quantity,
		UnitPriceCents:  t.UnitPriceCents,
		Remark:          t.Remark,
	}
	return dto
}

func (r *OrderRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Order.Query().
		Where(order.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, orderV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// FindBySourceRef 查外部平台订单（幂等键：tenant + source + source_ref）。
func (r *OrderRepo) FindBySourceRef(ctx context.Context, tenantID uint32, source order.Source, sourceRef string) (*ent.Order, error) {
	row, err := r.entClient.Client().Order.Query().
		Where(
			order.TenantIDEQ(tenantID),
			order.SourceEQ(source),
			order.SourceRefEQ(sourceRef),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query order by source_ref failed: %s", err.Error())
		return nil, orderV1.ErrorInternalServerError("query order failed")
	}
	return row, nil
}

// OrderNumberExists 单号是否已存在（租户范围）。
func (r *OrderRepo) OrderNumberExists(ctx context.Context, tenantID uint32, number string) (bool, error) {
	exist, err := r.entClient.Client().Order.Query().
		Where(
			order.TenantIDEQ(tenantID),
			order.OrderNumberEQ(number),
		).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query order number failed: %s", err.Error())
		return false, orderV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// genOrderNumber 生成唯一订单号：ORD + 毫秒时间戳 + 随机后缀（防并发碰撞）。
func genOrderNumber() string {
	return fmt.Sprintf("ORD%d%04d", time.Now().UnixMilli(), time.Now().Nanosecond()%10000)
}

// Create 创建订单（手动补录）+ 嵌套明细（单事务）。
// 单号服务端生成：ORD + 毫秒时间戳 + 纳秒后缀。
func (r *OrderRepo) Create(ctx context.Context, req *orderV1.CreateOrderRequest) (_ *orderV1.Order, err error) {
	if req == nil || req.Data == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	lines := req.Data.GetLines()
	if len(lines) == 0 {
		return nil, orderV1.ErrorBadRequest("order lines are required")
	}

	tid, _ := maybeTenantFromViewer(ctx)
	tenantID := tid
	if req.Data.TenantId != nil {
		tenantID = req.Data.GetTenantId()
	}

	orderNumber := genOrderNumber()

	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, orderV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() { r.FinishTx(tx, err) }()

	builder := tx.Order.Create().
		SetNillableTenantID(&tenantID).
		SetOrderNumber(orderNumber).
		SetNillableWarehouseCode(req.Data.WarehouseCode).
		SetNillableSourceRef(req.Data.SourceRef).
		SetNillableSourceShop(req.Data.SourceShop).
		SetNillableCustomerName(req.Data.CustomerName).
		SetNillableCustomerPhone(req.Data.CustomerPhone).
		SetNillableShippingAddress(req.Data.ShippingAddress).
		SetNillableCurrency(req.Data.Currency).
		SetStatus(order.StatusPending).
		SetNillableRemark(req.Data.Remark).
		SetCreatedAt(time.Now())
	if src := r.sourceConverter.ToEntity(req.Data.Source); src != nil {
		builder.SetSource(*src)
	} else {
		builder.SetSource(order.SourceManual)
	}
	if req.Data.TotalAmountCents != nil {
		builder.SetTotalAmountCents(req.Data.GetTotalAmountCents())
	}
	if req.Data.OrderedAt != nil {
		builder.SetOrderedAt(req.Data.OrderedAt.AsTime())
	}
	if req.Data.CreatedBy != nil {
		builder.SetCreatedBy(req.Data.GetCreatedBy())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert order failed: %s", err.Error())
		return nil, orderV1.ErrorInternalServerError("insert order failed")
	}

	if err = r.createLines(ctx, tx, t.ID, tenantID, lines, req.Data.CreatedBy); err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(t)
	dto.Status = orderV1.OrderStatus_ORDER_PENDING.Enum()
	if err = r.attachLinesTx(ctx, tx, []*orderV1.Order{dto}); err != nil {
		return nil, err
	}

	return dto, nil
}

// createLines 批量创建明细。
func (r *OrderRepo) createLines(ctx context.Context, tx *ent.Tx, orderID uint32, tenantID uint32, lines []*orderV1.OrderLine, createdBy *uint32) error {
	for _, l := range lines {
		if l == nil || l.GetSkuCode() == "" || l.GetQuantity() <= 0 {
			return orderV1.ErrorBadRequest("each order line requires sku_code and positive quantity")
		}

		lb := tx.OrderLine.Create().
			SetNillableTenantID(&tenantID).
			SetOrderID(orderID).
			SetNillableSkuCode(l.SkuCode).
			SetNillableProductName(l.ProductName).
			SetQuantity(l.GetQuantity()).
			SetNillableRemark(l.Remark).
			SetCreatedAt(time.Now())
		if l.UnitPriceCents != nil {
			lb.SetUnitPriceCents(l.GetUnitPriceCents())
		}
		if createdBy != nil {
			lb.SetCreatedBy(*createdBy)
		}
		if _, err := lb.Save(ctx); err != nil {
			r.log.Errorf("insert order line failed: %s", err.Error())
			return orderV1.ErrorInternalServerError("insert order line failed")
		}
	}

	return nil
}

// attachLinesTx 事务内按 order_id 批量装配明细。
func (r *OrderRepo) attachLinesTx(ctx context.Context, tx *ent.Tx, items []*orderV1.Order) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.GetId())
	}

	lines, err := tx.OrderLine.Query().
		Where(orderline.OrderIDIn(ids...)).
		Order(ent.Asc(orderline.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query order lines failed: %s", err.Error())
		return orderV1.ErrorInternalServerError("query order lines failed")
	}

	byOrder := make(map[uint32][]*orderV1.OrderLine, len(items))
	for _, l := range lines {
		dto := r.lineToDTO(l)
		byOrder[derefUint32(l.OrderID)] = append(byOrder[derefUint32(l.OrderID)], dto)
	}
	for _, it := range items {
		it.Lines = append(it.Lines, byOrder[it.GetId()]...)
	}

	return nil
}

// Update 更新订单头 + 整体替换明细（仅 PENDING；状态机校验在服务层）。
func (r *OrderRepo) Update(ctx context.Context, req *orderV1.UpdateOrderRequest) (result *orderV1.Order, err error) {
	if req == nil || req.Data == nil {
		return nil, orderV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Order.UpdateOneID(req.GetId())
	builder.Where(order.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(order.TenantIDEQ(tid))
	}

	if req.Data.OrderedAt != nil {
		builder.SetOrderedAt(req.Data.OrderedAt.AsTime())
	}
	if req.Data.TotalAmountCents != nil {
		builder.SetTotalAmountCents(req.Data.GetTotalAmountCents())
	}
	if req.Data.WarehouseCode != nil {
		builder.SetWarehouseCode(req.Data.GetWarehouseCode())
	}
	if req.Data.CustomerName != nil {
		builder.SetCustomerName(req.Data.GetCustomerName())
	}
	if req.Data.CustomerPhone != nil {
		builder.SetCustomerPhone(req.Data.GetCustomerPhone())
	}
	if req.Data.ShippingAddress != nil {
		builder.SetShippingAddress(req.Data.GetShippingAddress())
	}
	if req.Data.Remark != nil {
		builder.SetRemark(req.Data.GetRemark())
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}
	builder.SetUpdatedAt(time.Now())

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("update order failed: %s", err.Error())
		return nil, orderV1.ErrorInternalServerError("update order failed")
	}

	result = r.mapper.ToDTO(t)
	result.Source = orderSourceToProto(*t.Source)
	result.Status = orderStatusToProto(*t.Status)

	// 明细整体替换（仅 PENDING；调用方保证）。单事务先删后建。
	if req.Data.Lines != nil {
		tx, terr := r.entClient.Client().Tx(ctx)
		if terr != nil {
			return nil, orderV1.ErrorInternalServerError("start transaction failed")
		}
		defer func() { r.FinishTx(tx, err) }()

		if _, err = tx.OrderLine.Delete().
			Where(orderline.OrderIDEQ(req.GetId())).
			Exec(ctx); err != nil {
			r.log.Errorf("delete order lines failed: %s", err.Error())
			return nil, orderV1.ErrorInternalServerError("delete order lines failed")
		}

		if err = r.createLines(ctx, tx, req.GetId(), result.GetTenantId(), req.Data.Lines, result.CreatedBy); err != nil {
			return nil, err
		}
	}

	r.backfillStatusSource(ctx, []*orderV1.Order{result})
	if err = r.attachLines(ctx, []*orderV1.Order{result}); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *OrderRepo) Delete(ctx context.Context, req *orderV1.DeleteOrderRequest) error {
	if req == nil {
		return orderV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)

	lineDel := r.entClient.Client().OrderLine.Delete().
		Where(orderline.OrderIDEQ(req.GetId()))
	if hasTenant {
		lineDel.Where(orderline.TenantIDEQ(tid))
	}
	if _, err := lineDel.Exec(ctx); err != nil {
		r.log.Errorf("delete order lines failed: %s", err.Error())
		return orderV1.ErrorInternalServerError("delete failed")
	}

	delBuilder := r.entClient.Client().Order.Delete()
	delBuilder.Where(order.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(order.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return orderV1.ErrorNotFound("order not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return orderV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

// SetStatus 单票原子状态流转（带租户范围校验）。
func (r *OrderRepo) SetStatus(ctx context.Context, id uint32, from order.Status, to order.Status, pickingID *uint32) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	builder := r.entClient.Client().Order.Update().
		Where(
			order.IDEQ(id),
			order.StatusEQ(from),
		)
	if hasTenant {
		builder.Where(order.TenantIDEQ(tid))
	}
	builder.SetStatus(to).
		SetNillablePickingID(pickingID).
		SetUpdatedAt(time.Now())

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("update order status failed: %s", err.Error())
		return orderV1.ErrorInternalServerError("update order failed")
	}
	if n == 0 {
		return orderV1.ErrorConflict("order status transition not allowed in current status")
	}

	return nil
}

// BeginTx 开启事务。
func (r *OrderRepo) BeginTx(ctx context.Context) (*ent.Tx, error) {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, orderV1.ErrorInternalServerError("start transaction failed")
	}
	return tx, nil
}

// FinishTx 提交/回滚事务（err 非 nil 回滚）。
func (r *OrderRepo) FinishTx(tx *ent.Tx, err error) {
	if tx == nil {
		return
	}
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
		}
		return
	}
	if commitErr := tx.Commit(); commitErr != nil {
		r.log.Errorf("transaction commit failed: %s", commitErr.Error())
	}
}
