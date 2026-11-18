package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	appV1 "github.com/Mikdad-Force/Ghost/api/gen/go/app/service/v1"
	purchaseV1 "github.com/Mikdad-Force/Ghost/api/gen/go/purchase/service/v1"
)

// PurchaseService 到货通知单（PDA 手持端转发）：扫码查单 + 收货 + 上架。
type PurchaseService struct {
	appV1.PurchaseServiceHTTPServer

	log *log.Helper

	purchaseServiceClient purchaseV1.PurchaseServiceClient
}

func NewPurchaseService(
	ctx *bootstrap.Context,
	purchaseServiceClient purchaseV1.PurchaseServiceClient,
) *PurchaseService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "purchase/service/app-service"))
	return &PurchaseService{
		log:              l,
		purchaseServiceClient: purchaseServiceClient,
	}
}

func (s *PurchaseService) List(ctx context.Context, req *paginationV1.PagingRequest) (*purchaseV1.ListPurchaseResponse, error) {
	return s.purchaseServiceClient.List(ctx, req)
}

func (s *PurchaseService) Get(ctx context.Context, req *purchaseV1.GetPurchaseRequest) (*purchaseV1.Purchase, error) {
	return s.purchaseServiceClient.Get(ctx, req)
}

func (s *PurchaseService) GetByNumber(ctx context.Context, req *purchaseV1.GetPurchaseByNumberRequest) (*purchaseV1.Purchase, error) {
	if req == nil || req.GetNumber() == "" {
		return nil, appV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.GetByNumber(ctx, req)
}

func (s *PurchaseService) Receive(ctx context.Context, req *purchaseV1.ReceivePurchaseRequest) (*purchaseV1.ReceivePurchaseResponse, error) {
	if req == nil || req.GetPurchaseId() == 0 {
		return nil, appV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Receive(ctx, req)
}

func (s *PurchaseService) Putaway(ctx context.Context, req *purchaseV1.PutawayPurchaseRequest) (*purchaseV1.PutawayPurchaseResponse, error) {
	if req == nil || req.GetPurchaseId() == 0 {
		return nil, appV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Putaway(ctx, req)
}
