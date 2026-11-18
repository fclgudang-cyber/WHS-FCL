package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingtask"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/go-utils/trans"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data"

	"github.com/tx7do/go-crud/viewer"

	pickingTaskV1 "github.com/Mikdad-Force/Ghost/api/gen/go/picking_task/service/v1"
)

// PickingTaskService 拣货任务服务：波次 lepaskan → 生成任务。
type PickingTaskService struct {
	pickingTaskV1.UnimplementedPickingTaskServiceServer

	log *log.Helper

	pickingTaskRepo *data.PickingTaskRepo
	pickingCartRepo *data.PickingCartRepo
	waveRepo        *data.WaveRepo

	stockPickingRepo *data.StockPickingRepo
}

func NewPickingTaskService(
	ctx *bootstrap.Context,
	pickingTaskRepo *data.PickingTaskRepo,
	pickingCartRepo *data.PickingCartRepo,
	stockPickingRepo *data.StockPickingRepo,
	waveRepo *data.WaveRepo,
) *PickingTaskService {
	return &PickingTaskService{
		log:             ctx.NewLoggerHelper("picking-task/service/core-service"),
		pickingTaskRepo: pickingTaskRepo,
		pickingCartRepo: pickingCartRepo,
		stockPickingRepo: stockPickingRepo,
		waveRepo:        waveRepo,
	}
}

func (s *PickingTaskService) List(ctx context.Context, req *paginationV1.PagingRequest) (*pickingTaskV1.ListPickingTaskResponse, error) {
	return s.pickingTaskRepo.List(ctx, req)
}

func (s *PickingTaskService) Get(ctx context.Context, req *pickingTaskV1.GetPickingTaskRequest) (*pickingTaskV1.PickingTask, error) {
	if req == nil || req.GetId() == 0 {
		return nil, pickingTaskV1.ErrorBadRequest("invalid parameter")
	}
	return s.pickingTaskRepo.Get(ctx, req.GetId())
}

func (s *PickingTaskService) ListCarts(ctx context.Context, _ *pickingTaskV1.ListCartsRequest) (*pickingTaskV1.ListPickingCartResponse, error) {
	_ = tenantIDFromContext(ctx)

	page, pageSize := uint32(1), uint32(20)
	req := &paginationV1.PagingRequest{Page: &page, PageSize: &pageSize}
	rows, err := s.pickingCartRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	items := make([]*pickingTaskV1.PickingCart, 0, len(rows.Items))
	for _, row := range rows.Items {
		statusStr := "FREE"
		if row.Status != nil {
			statusStr = string(*row.Status)
		}
		items = append(items, &pickingTaskV1.PickingCart{
			Id:         trans.Ptr(row.ID),
			CartNumber: row.CartNumber,
			Status:     trans.Ptr(statusStr),
		})
	}

	return &pickingTaskV1.ListPickingCartResponse{
		Total: uint64(len(items)),
		Items: items,
	}, nil
}

func (s *PickingTaskService) CreateCart(ctx context.Context, req *pickingTaskV1.CreateCartRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, pickingTaskV1.ErrorBadRequest("invalid parameter")
	}
	number := req.GetCartNumber()
	if number == "" {
		return nil, pickingTaskV1.ErrorBadRequest("cart_number is required")
	}

	tid := tenantIDFromContext(ctx)
	userID, _ := viewerUserIDFromContext(ctx)

	if err := s.pickingCartRepo.Create(ctx, tid, number, &userID); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *PickingTaskService) ReleaseCart(ctx context.Context, req *pickingTaskV1.CancelPickingCartRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, pickingTaskV1.ErrorBadRequest("invalid parameter")
	}
	callerUserID, _ := viewerUserIDFromContext(ctx)
	_ = callerUserID
	if err := s.pickingTaskRepo.SetCartStatusFree(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *PickingTaskService) Create(ctx context.Context, req *pickingTaskV1.CreatePickingTaskRequest) (*emptypb.Empty, error) {
	return nil, pickingTaskV1.ErrorBadRequest("create via wave release only")
}

func (s *PickingTaskService) Cancel(ctx context.Context, req *pickingTaskV1.CancelPickingTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, pickingTaskV1.ErrorBadRequest("invalid parameter")
	}
	_, _ = viewerUserIDFromContext(ctx)
	if err := s.pickingTaskRepo.SetStatus(ctx, req.GetId(), nil, "CANCELLED"); err != nil {
		return nil, err
	}
	task, err := s.pickingTaskRepo.Get(ctx, req.GetId())
	if err == nil && task.GetCartId() != 0 {
		_ = s.pickingTaskRepo.SetCartStatusFree(ctx, task.GetCartId())
	}
	return &emptypb.Empty{}, nil
}

func (s *PickingTaskService) ScanCart(ctx context.Context, req *pickingTaskV1.ScanCartRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetTaskId() == 0 || req.GetCartNumber() == "" {
		return nil, pickingTaskV1.ErrorBadRequest("task_id and cartNumber required")
	}

	tid := tenantIDFromContext(ctx)

	task, err := s.pickingTaskRepo.Get(ctx, req.GetTaskId())
	if err != nil {
		return nil, err
	}
	status := task.GetStatus()
	if status != pickingTaskV1.PickingTaskStatus_PICKING_TASK_WAITING && status != pickingTaskV1.PickingTaskStatus_PICKING_TASK_RUNNING {
		return nil, pickingTaskV1.ErrorConflict("task is not in waiting/running state")
	}

	cart, err := s.pickingTaskRepo.FindCartByNumber(ctx, tid, req.GetCartNumber())
	if err != nil {
		return nil, pickingTaskV1.ErrorNotFound("keranjang not found")
	}
	if cart.Status == nil || *cart.Status != "FREE" {
		return nil, pickingTaskV1.ErrorConflict("keranjang is in use or disabled")
	}
	// 检查 cart 是否被其他进行中的任务占用
	inUse, err := s.pickingTaskRepo.IsCartInUseByOther(ctx, tid, cart.ID, req.GetTaskId())
	if err != nil {
		return nil, err
	}
	if inUse {
		return nil, pickingTaskV1.ErrorConflict("keranjang already used by another active task")
	}

	if err := s.pickingTaskRepo.SetCart(ctx, req.GetTaskId(), cart.ID); err != nil {
		return nil, err
	}
	if err := s.pickingTaskRepo.SetCartStatusInUse(ctx, cart.ID); err != nil {
		return nil, err
	}

	// 把任务状态推到 RUNNING（首次扫码）
	if status == pickingTaskV1.PickingTaskStatus_PICKING_TASK_WAITING {
		if err := s.pickingTaskRepo.SetStatus(ctx, req.GetTaskId(), nil, "RUNNING"); err != nil {
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *PickingTaskService) ScanLocation(ctx context.Context, req *pickingTaskV1.ScanLocationRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetTaskId() == 0 || req.GetLocationCode() == "" {
		return nil, pickingTaskV1.ErrorBadRequest("task_id and locationCode required")
	}

	items, err := s.pickingTaskRepo.GetItems(ctx, req.GetTaskId())
	if err != nil {
		return nil, err
	}
	found := false
	for _, it := range items {
		if it.AllocatedLocationCode != nil && *it.AllocatedLocationCode == req.GetLocationCode() {
			found = true
			break
		}
	}
	if !found {
		return nil, pickingTaskV1.ErrorNotFound("locationCode not found in this task")
	}
	// 校验任务状态：未 RUNNING 先扫 cart
	task, err := s.pickingTaskRepo.Get(ctx, req.GetTaskId())
	if err != nil {
		return nil, err
	}
	if task.GetStatus() != pickingTaskV1.PickingTaskStatus_PICKING_TASK_RUNNING {
		return nil, pickingTaskV1.ErrorConflict("scan keranjang first to start the task")
	}
	callerUserID, _ := viewerUserIDFromContext(ctx)
	_ = s.pickingTaskRepo.LogAction(ctx, req.GetTaskId(), 0, nil, "SCAN_LOCATION", req.GetLocationCode(), "", 0, &callerUserID, 0)
	return &emptypb.Empty{}, nil
}

func (s *PickingTaskService) ScanSku(ctx context.Context, req *pickingTaskV1.ScanSkuRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetTaskId() == 0 || req.GetSkuCode() == "" {
		return nil, pickingTaskV1.ErrorBadRequest("task_id and skuCode required")
	}
	items, err := s.pickingTaskRepo.GetItems(ctx, req.GetTaskId())
	if err != nil {
		return nil, err
	}

	var target *ent.PickingTaskItem
	for _, it := range items {
		if it.SkuCode != nil && *it.SkuCode == req.GetSkuCode() {
			// 选第一个未完成的
			if it.Status == nil {
				target = it
				break
			}
			switch *it.Status {
			case "DONE", "PICKING":
				continue
			}
			target = it
			break
		}
	}
	if target == nil {
		return nil, pickingTaskV1.ErrorNotFound("skuCode not found in this task")
	}

	callerUserID, _ := viewerUserIDFromContext(ctx)
	if err := s.pickingTaskRepo.SetItemPicking(ctx, target.ID); err != nil {
		_ = err
	}
	_ = s.pickingTaskRepo.LogAction(ctx, req.GetTaskId(), target.ID, target, "SCAN_SKU", req.GetSkuCode(), req.GetSkuCode(), 0, &callerUserID, 0)
	return &emptypb.Empty{}, nil
}

func (s *PickingTaskService) ScanQty(ctx context.Context, req *pickingTaskV1.ScanQtyRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetTaskId() == 0 || req.GetItemId() == 0 {
		return nil, pickingTaskV1.ErrorBadRequest("task_id and itemId required")
	}
	qty := req.GetQty()
	if qty <= 0 {
		return nil, pickingTaskV1.ErrorBadRequest("qty must be > 0")
	}
	items, err := s.pickingTaskRepo.GetItems(ctx, req.GetTaskId())
	if err != nil {
		return nil, err
	}
	var target *ent.PickingTaskItem
	for _, it := range items {
		if it.ID == req.GetItemId() {
			target = it
			break
		}
	}
	if target == nil {
		return nil, pickingTaskV1.ErrorNotFound("item not found")
	}
	required := derefInt64(target.QtyRequired)
	picked := derefInt64(target.QtyPicked)
	if picked+qty > required {
		return nil, pickingTaskV1.ErrorBadRequest("qty exceeds required")
	}

	callerUserID, _ := viewerUserIDFromContext(ctx)
	shortReason := req.GetShortReason()
	if picked+qty < required {
		if shortReason == "" {
			return nil, pickingTaskV1.ErrorBadRequest("shortReason required when qty < remaining")
		}
		if err := s.pickingTaskRepo.SetItemShort(ctx, target.ID, picked+qty, shortReason); err != nil {
			return nil, err
		}
	} else {
		if err := s.pickingTaskRepo.SetItemDone(ctx, target.ID, picked+qty); err != nil {
			return nil, err
		}
	}
	_ = s.pickingTaskRepo.LogAction(ctx, req.GetTaskId(), target.ID, target, "SCAN_QTY", shortReason, deref(target.SkuCode), qty, &callerUserID, 0)

	// 全明细 DONE → 任务 DONE，keranjang → FREE
	rest, err := s.pickingTaskRepo.GetItems(ctx, req.GetTaskId())
	if err == nil {
		allDone := true
		for _, it := range rest {
			if it.Status == nil {
				allDone = false
				break
			}
			switch *it.Status {
			case "DONE", "SHORT":
			default:
				allDone = false
			}
		}
		if allDone {
			_ = s.pickingTaskRepo.SetStatus(ctx, req.GetTaskId(), []pickingtask.Status{pickingtask.StatusWaiting, pickingtask.StatusRunning}, pickingtask.StatusDone)
			task, err := s.pickingTaskRepo.Get(ctx, req.GetTaskId())
			if err == nil && task.GetCartId() != 0 {
				_ = s.pickingTaskRepo.SetCartStatusFree(ctx, task.GetCartId())
			}
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *PickingTaskService) CompleteItem(ctx context.Context, req *pickingTaskV1.CompleteItemRequest) (*emptypb.Empty, error) {
	return nil, pickingTaskV1.ErrorBadRequest("not implemented")
}

// tenantIDFromContext service 层提取租户 ID（与 data 层同语义）
func tenantIDFromContext(ctx context.Context) uint32 {
	if vc, exist := viewer.FromContext(ctx); exist && vc != nil {
		return uint32(vc.TenantID())
	}
	return 0
}
