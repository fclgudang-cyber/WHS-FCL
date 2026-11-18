package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "github.com/Mikdad-Force/Ghost/api/gen/go/admin/service/v1"
	sortingV1 "github.com/Mikdad-Force/Ghost/api/gen/go/sorting/service/v1"
)

type SortingService struct {
	adminV1.SortingServiceHTTPServer

	log *log.Helper

	sortingServiceClient sortingV1.SortingServiceClient
}

func NewSortingService(
	ctx *bootstrap.Context,
	sortingServiceClient sortingV1.SortingServiceClient,
) *SortingService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "sorting/service/admin-service"))
	return &SortingService{
		log:                  l,
		sortingServiceClient: sortingServiceClient,
	}
}

func (s *SortingService) ListSessions(ctx context.Context, req *paginationV1.PagingRequest) (*sortingV1.ListSortSessionsResponse, error) {
	return s.sortingServiceClient.ListSessions(ctx, req)
}
func (s *SortingService) GetSession(ctx context.Context, req *sortingV1.GetSortSessionRequest) (*sortingV1.SortSession, error) {
	return s.sortingServiceClient.GetSession(ctx, req)
}
func (s *SortingService) StartSession(ctx context.Context, req *sortingV1.StartSortSessionRequest) (*sortingV1.StartSortSessionResponse, error) {
	return s.sortingServiceClient.StartSession(ctx, req)
}
func (s *SortingService) LockBasket(ctx context.Context, req *sortingV1.LockBasketRequest) (*sortingV1.LockBasketResponse, error) {
	return s.sortingServiceClient.LockBasket(ctx, req)
}
func (s *SortingService) UnlockBasket(ctx context.Context, req *sortingV1.UnlockBasketRequest) (*emptypb.Empty, error) {
	return s.sortingServiceClient.UnlockBasket(ctx, req)
}
func (s *SortingService) ScanSku(ctx context.Context, req *sortingV1.ScanSkuSortingRequest) (*sortingV1.ScanSkuSortingResponse, error) {
	return s.sortingServiceClient.ScanSku(ctx, req)
}
func (s *SortingService) ScanSlotItem(ctx context.Context, req *sortingV1.ScanSortSlotItemRequest) (*sortingV1.ScanSortSlotItemResponse, error) {
	return s.sortingServiceClient.ScanSlotItem(ctx, req)
}
func (s *SortingService) CancelSession(ctx context.Context, req *sortingV1.CancelSortSessionRequest) (*emptypb.Empty, error) {
	return s.sortingServiceClient.CancelSession(ctx, req)
}
