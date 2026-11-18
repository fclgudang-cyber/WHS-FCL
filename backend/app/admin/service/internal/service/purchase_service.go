package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "github.com/Mikdad-Force/Ghost/api/gen/go/admin/service/v1"
	purchaseV1 "github.com/Mikdad-Force/Ghost/api/gen/go/purchase/service/v1"
)

type PurchaseService struct {
	adminV1.PurchaseServiceHTTPServer

	log *log.Helper

	purchaseServiceClient purchaseV1.PurchaseServiceClient
}

func NewPurchaseService(
	ctx *bootstrap.Context,
	purchaseServiceClient purchaseV1.PurchaseServiceClient,
) *PurchaseService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "purchase/service/admin-service"))
	return &PurchaseService{
		log:              l,
		purchaseServiceClient: purchaseServiceClient,
	}
}

func (s *PurchaseService) List(ctx context.Context, req *paginationV1.PagingRequest) (*purchaseV1.ListPurchaseResponse, error) {
	return s.purchaseServiceClient.List(ctx, req)
}

func (s *PurchaseService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*purchaseV1.CountPurchaseResponse, error) {
	return s.purchaseServiceClient.Count(ctx, req)
}

func (s *PurchaseService) Get(ctx context.Context, req *purchaseV1.GetPurchaseRequest) (*purchaseV1.Purchase, error) {
	return s.purchaseServiceClient.Get(ctx, req)
}

func (s *PurchaseService) Create(ctx context.Context, req *purchaseV1.CreatePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Create(ctx, req)
}

func (s *PurchaseService) Update(ctx context.Context, req *purchaseV1.UpdatePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Update(ctx, req)
}

func (s *PurchaseService) Delete(ctx context.Context, req *purchaseV1.DeletePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Delete(ctx, req)
}

func (s *PurchaseService) Announce(ctx context.Context, req *purchaseV1.AnnouncePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Announce(ctx, req)
}

func (s *PurchaseService) Cancel(ctx context.Context, req *purchaseV1.CancelPurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Cancel(ctx, req)
}

func (s *PurchaseService) Receive(ctx context.Context, req *purchaseV1.ReceivePurchaseRequest) (*purchaseV1.ReceivePurchaseResponse, error) {
	if req == nil || req.GetPurchaseId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Receive(ctx, req)
}

func (s *PurchaseService) Putaway(ctx context.Context, req *purchaseV1.PutawayPurchaseRequest) (*purchaseV1.PutawayPurchaseResponse, error) {
	if req == nil || req.GetPurchaseId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Putaway(ctx, req)
}

func (s *PurchaseService) Close(ctx context.Context, req *purchaseV1.ClosePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseServiceClient.Close(ctx, req)
}
