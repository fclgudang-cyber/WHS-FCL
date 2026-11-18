package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "github.com/Mikdad-Force/Ghost/api/gen/go/admin/service/v1"
	pickingTaskV1 "github.com/Mikdad-Force/Ghost/api/gen/go/picking_task/service/v1"
)

// PickingTaskService 拣货任务 BFF facade（管理后台）。
type PickingTaskService struct {
	adminV1.PickingTaskServiceHTTPServer

	log *log.Helper

	pickingTaskServiceClient pickingTaskV1.PickingTaskServiceClient
}

func NewPickingTaskService(
	ctx *bootstrap.Context,
	pickingTaskServiceClient pickingTaskV1.PickingTaskServiceClient,
) *PickingTaskService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "picking-task/service/admin-service"))
	return &PickingTaskService{
		log:                     l,
		pickingTaskServiceClient: pickingTaskServiceClient,
	}
}

func (s *PickingTaskService) List(ctx context.Context, req *paginationV1.PagingRequest) (*pickingTaskV1.ListPickingTaskResponse, error) {
	return s.pickingTaskServiceClient.List(ctx, req)
}

func (s *PickingTaskService) Get(ctx context.Context, req *pickingTaskV1.GetPickingTaskRequest) (*pickingTaskV1.PickingTask, error) {
	return s.pickingTaskServiceClient.Get(ctx, req)
}

func (s *PickingTaskService) Create(ctx context.Context, req *pickingTaskV1.CreatePickingTaskRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.Create(ctx, req)
}

func (s *PickingTaskService) ScanCart(ctx context.Context, req *pickingTaskV1.ScanCartRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.ScanCart(ctx, req)
}

func (s *PickingTaskService) ScanLocation(ctx context.Context, req *pickingTaskV1.ScanLocationRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.ScanLocation(ctx, req)
}

func (s *PickingTaskService) ScanSku(ctx context.Context, req *pickingTaskV1.ScanSkuRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.ScanSku(ctx, req)
}

func (s *PickingTaskService) ScanQty(ctx context.Context, req *pickingTaskV1.ScanQtyRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.ScanQty(ctx, req)
}

func (s *PickingTaskService) CompleteItem(ctx context.Context, req *pickingTaskV1.CompleteItemRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.CompleteItem(ctx, req)
}

func (s *PickingTaskService) Cancel(ctx context.Context, req *pickingTaskV1.CancelPickingTaskRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.Cancel(ctx, req)
}

func (s *PickingTaskService) ListCarts(ctx context.Context, req *pickingTaskV1.ListCartsRequest) (*pickingTaskV1.ListPickingCartResponse, error) {
	return s.pickingTaskServiceClient.ListCarts(ctx, req)
}

func (s *PickingTaskService) CreateCart(ctx context.Context, req *pickingTaskV1.CreateCartRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.CreateCart(ctx, req)
}

func (s *PickingTaskService) ReleaseCart(ctx context.Context, req *pickingTaskV1.CancelPickingCartRequest) (*emptypb.Empty, error) {
	return s.pickingTaskServiceClient.ReleaseCart(ctx, req)
}
