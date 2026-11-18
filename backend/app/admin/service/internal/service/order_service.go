package service

import (
	"context"
	"os"


	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	orderV1 "github.com/Mikdad-Force/Ghost/api/gen/go/order/service/v1"

	adminV1 "github.com/Mikdad-Force/Ghost/api/gen/go/admin/service/v1"
	"github.com/Mikdad-Force/Ghost/pkg/netutil"
)

// OrderService 订单管理 BFF facade（管理后台）。
// Webhook 用于 Marketplace 平台回调：Header X-Marketplace-Token 与环境变量
// ORDER_WEBHOOK_TOKEN 比对（缺省为开发默认值，生产必须覆盖）。
type OrderService struct {
	adminV1.OrderServiceHTTPServer

	log *log.Helper

	orderServiceClient orderV1.OrderServiceClient
}

func NewOrderService(
	ctx *bootstrap.Context,
	orderServiceClient orderV1.OrderServiceClient,
) *OrderService {
	return &OrderService{
		log:                ctx.NewLoggerHelper("order/service/admin-service"),
		orderServiceClient: orderServiceClient,
	}
}

func (s *OrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*orderV1.ListOrderResponse, error) {
	return s.orderServiceClient.List(ctx, req)
}

func (s *OrderService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*orderV1.CountOrderResponse, error) {
	return s.orderServiceClient.Count(ctx, req)
}

func (s *OrderService) Get(ctx context.Context, req *orderV1.GetOrderRequest) (*orderV1.Order, error) {
	return s.orderServiceClient.Get(ctx, req)
}

func (s *OrderService) Create(ctx context.Context, req *orderV1.CreateOrderRequest) (*emptypb.Empty, error) {
	return s.orderServiceClient.Create(ctx, req)
}

func (s *OrderService) Update(ctx context.Context, req *orderV1.UpdateOrderRequest) (*emptypb.Empty, error) {
	return s.orderServiceClient.Update(ctx, req)
}

func (s *OrderService) Delete(ctx context.Context, req *orderV1.DeleteOrderRequest) (*emptypb.Empty, error) {
	return s.orderServiceClient.Delete(ctx, req)
}

func (s *OrderService) Confirm(ctx context.Context, req *orderV1.ConfirmOrderRequest) (*emptypb.Empty, error) {
	return s.orderServiceClient.Confirm(ctx, req)
}

func (s *OrderService) Cancel(ctx context.Context, req *orderV1.CancelOrderRequest) (*emptypb.Empty, error) {
	return s.orderServiceClient.Cancel(ctx, req)
}

func (s *OrderService) Fulfill(ctx context.Context, req *orderV1.FulfillOrderRequest) (*orderV1.FulfillOrderResponse, error) {
	return s.orderServiceClient.Fulfill(ctx, req)
}

func (s *OrderService) Complete(ctx context.Context, req *orderV1.CompleteOrderRequest) (*emptypb.Empty, error) {
	return s.orderServiceClient.Complete(ctx, req)
}

func (s *OrderService) ImportMarketplace(ctx context.Context, req *orderV1.ImportMarketplaceRequest) (*orderV1.ImportMarketplaceResponse, error) {
	return s.orderServiceClient.ImportMarketplace(ctx, req)
}

// Webhook Marketplace 回调入口：校验 X-Marketplace-Token 后转发 core 导入。
// source（来源渠道）由路由参数 {source} 携带（SHOPEE/TOKOPEDIA/LAZADA/TIKTOK/OTHER）。
func (s *OrderService) Webhook(ctx context.Context, req *orderV1.ImportMarketplaceRequest) (*orderV1.ImportMarketplaceResponse, error) {
	token := netutil.HeaderFromContext(ctx).Get("X-Marketplace-Token")
	expected := os.Getenv("ORDER_WEBHOOK_TOKEN")
	if expected == "" {
		expected = "dev-webhook-token-change-me"
	}
	if token == "" || token != expected {
		return nil, orderV1.ErrorUnauthorized("invalid or missing X-Marketplace-Token")
	}
	return s.orderServiceClient.ImportMarketplace(ctx, req)
}
