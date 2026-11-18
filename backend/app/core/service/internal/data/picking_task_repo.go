package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingcart"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingtask"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingtaskitem"

	"google.golang.org/protobuf/types/known/timestamppb"

	pickingTaskV1 "github.com/Mikdad-Force/Ghost/api/gen/go/picking_task/service/v1"
)

// PickingTaskRepo 拣货任务仓储。
type PickingTaskRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewPickingTaskRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PickingTaskRepo {
	repo := &PickingTaskRepo{
		log:       ctx.NewLoggerHelper("picking-task/repo/core-service"),
		entClient: entClient,
	}
	return repo
}

// Bootstrap helper
func (r *PickingTaskRepo) BeginTx(ctx context.Context) (*ent.Tx, error) {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, pickingTaskV1.ErrorInternalServerError("start transaction failed")
	}
	return tx, nil
}
func (r *PickingTaskRepo) FinishTx(tx *ent.Tx, err error) {
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

// CreateTx 创建任务（WAITING）。
func (r *PickingTaskRepo) CreateTx(ctx context.Context, tx *ent.Tx, waveID uint32, createdBy *uint32) (*ent.PickingTask, error) {
	taskNumber := "TASK" + time.Now().Format("20060102150405")[2:] + time.Now().Format("_1504")
	taskNumber = time.Now().Format("20060102150405")[2:]
	taskNumber = "T" + taskNumber

	builder := tx.PickingTask.Create().
		SetTaskNumber(taskNumber).
		SetWaveID(waveID).
		SetStatus(pickingtask.StatusWaiting).
		SetNillableCreatedBy(createdBy).
		SetCreatedAt(time.Now())

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("create picking task failed: %s", err.Error())
		return nil, pickingTaskV1.ErrorInternalServerError("create picking task failed")
	}
	return t, nil
}

// allocCodeFromWaveItem 从波次行推导 lokasi 扫码编码（LOK{id}）。
func allocCodeFromWaveItem(it *ent.WaveItem) *string {
	if it == nil || it.AllocatedLocationID == nil || *it.AllocatedLocationID == "" {
		return nil
	}
	code := "LOK" + *it.AllocatedLocationID
	return &code
}

// CreateItemsFromWaveItemsTx 波次 SKU 行 → 任务明细。
func (r *PickingTaskRepo) CreateItemsFromWaveItemsTx(ctx context.Context, tx *ent.Tx, taskID uint32, items []*ent.WaveItem, createdBy *uint32) error {
	for _, it := range items {
		if it == nil {
			continue
		}

		_, err := tx.PickingTaskItem.Create().
			SetNillableTenantID(it.TenantID).
			SetTaskID(taskID).
			SetNillableSkuCode(it.SkuCode).
			SetNillableProductName(it.ProductName).
			SetNillableQtyRequired(it.QtyRequired).
			SetNillableAllocatedLocationID(it.AllocatedLocationID).
			SetNillableAllocatedLocationCode(allocCodeFromWaveItem(it)).
			SetNillableCreatedBy(createdBy).
			SetCreatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			r.log.Errorf("create picking task item failed: %s", err.Error())
			return pickingTaskV1.ErrorInternalServerError("create picking task item failed")
		}
	}
	return nil
}

// List 任务列表（分页）。
func (r *PickingTaskRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*pickingTaskV1.ListPickingTaskResponse, error) {
	if req == nil {
		return nil, pickingTaskV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PickingTask.Query()

	// 兜底默认排序
	builder.Order(ent.Desc(pickingtask.FieldCreatedAt))

	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	total, err := builder.Clone().Count(ctx)
	if err != nil {
		return nil, pickingTaskV1.ErrorInternalServerError("query count failed")
	}

	rows, err := builder.
		Order(ent.Desc(pickingtask.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		r.log.Errorf("query picking tasks failed: %s", err.Error())
		return nil, pickingTaskV1.ErrorInternalServerError("query picking tasks failed")
	}

	items := make([]*pickingTaskV1.PickingTask, 0, len(rows))
	for _, row := range rows {
		dto := toTaskDTO(row)
		// attach 明细
		taskItems, err := r.GetItems(ctx, row.ID)
		if err == nil {
			dto.Items = make([]*pickingTaskV1.PickingTaskItem, 0, len(taskItems))
			for _, it := range taskItems {
				dto.Items = append(dto.Items, toTaskItemDTO(it))
			}
		}
		items = append(items, dto)
	}

	return &pickingTaskV1.ListPickingTaskResponse{
		Total: uint64(total),
		Items: items,
	}, nil
}

func (r *PickingTaskRepo) Get(ctx context.Context, id uint32) (*pickingTaskV1.PickingTask, error) {
	row, err := r.entClient.Client().PickingTask.Query().
		Where(pickingtask.IDEQ(id)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, pickingTaskV1.ErrorNotFound("picking task not found")
		}
		return nil, pickingTaskV1.ErrorInternalServerError("query picking task failed")
	}
	dto := toTaskDTO(row)
	taskItems, err := r.GetItems(ctx, id)
	if err != nil {
		return nil, err
	}
	dto.Items = make([]*pickingTaskV1.PickingTaskItem, 0, len(taskItems))
	for _, it := range taskItems {
		dto.Items = append(dto.Items, toTaskItemDTO(it))
	}
	return dto, nil
}

func (r *PickingTaskRepo) GetItems(ctx context.Context, taskID uint32) ([]*ent.PickingTaskItem, error) {
	rows, err := r.entClient.Client().PickingTaskItem.Query().
		Where(pickingtaskitem.TaskIDEQ(taskID)).
		Order(ent.Asc(pickingtaskitem.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query task items failed: %s", err.Error())
		return nil, pickingTaskV1.ErrorInternalServerError("query task items failed")
	}
	return rows, nil
}

// SetStatus 条件更新状态。
func (r *PickingTaskRepo) SetStatus(ctx context.Context, id uint32, from []pickingtask.Status, to pickingtask.Status) error {
	builder := r.entClient.Client().PickingTask.UpdateOneID(id)
	if len(from) != 0 {
		builder.Where(pickingtask.StatusIn(from...))
	}
	builder.SetStatus(to).SetUpdatedAt(time.Now())
	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update task status failed: %s", err.Error())
		return pickingTaskV1.ErrorInternalServerError("update task failed")
	}
	return nil
}

func (r *PickingTaskRepo) SetCart(ctx context.Context, taskID uint32, cartID uint32) error {
	_, err := r.entClient.Client().PickingTask.UpdateOneID(taskID).
		SetCartID(cartID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("set cart failed: %s", err.Error())
		return pickingTaskV1.ErrorInternalServerError("update task failed")
	}
	return nil
}

func (r *PickingTaskRepo) FindCartByNumber(ctx context.Context, tid uint32, number string) (*ent.PickingCart, error) {
	row, err := r.entClient.Client().PickingCart.Query().
		Where(pickingcart.TenantIDEQ(tid), pickingcart.CartNumberEQ(number)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *PickingTaskRepo) SetCartStatusInUse(ctx context.Context, cartID uint32) error {
	_, err := r.entClient.Client().PickingCart.UpdateOneID(cartID).
		SetStatus(pickingcart.StatusInUse).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *PickingTaskRepo) SetCartStatusFree(ctx context.Context, cartID uint32) error {
	_, err := r.entClient.Client().PickingCart.UpdateOneID(cartID).
		SetStatus(pickingcart.StatusFree).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *PickingTaskRepo) FindCartByID(ctx context.Context, cartID uint32) (*ent.PickingCart, error) {
	row, err := r.entClient.Client().PickingCart.Query().
		Where(pickingcart.IDEQ(cartID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *PickingTaskRepo) IsCartInUseByOther(ctx context.Context, tid uint32, cartID uint32, excludeTaskID uint32) (bool, error) {
	exists, err := r.entClient.Client().PickingTask.Query().
		Where(
			pickingtask.TenantIDEQ(tid),
			pickingtask.CartIDEQ(cartID),
			pickingtask.StatusIn(pickingtask.StatusWaiting, pickingtask.StatusRunning),
			pickingtask.IDEQ(excludeTaskID),
		).
		Exist(ctx)
	return exists, err
}

func (r *PickingTaskRepo) IsCartInUseTx(ctx context.Context, tx *ent.Tx, tid uint32, cartID uint32, excludeTaskID uint32) (bool, error) {
	exists, err := tx.PickingTask.Query().
		Where(
			pickingtask.TenantIDEQ(tid),
			pickingtask.CartIDEQ(cartID),
			pickingtask.StatusIn(pickingtask.StatusWaiting, pickingtask.StatusRunning),
			pickingtask.IDEQ(excludeTaskID),
		).
		Exist(ctx)
	return exists, err
}

func (r *PickingTaskRepo) SetItemPicking(ctx context.Context, itemID uint32) error {
	_, err := r.entClient.Client().PickingTaskItem.UpdateOneID(itemID).
		SetStatus(pickingtaskitem.StatusPicking).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *PickingTaskRepo) SetItemDone(ctx context.Context, itemID uint32, qtyPicked int64) error {
	_, err := r.entClient.Client().PickingTaskItem.UpdateOneID(itemID).
		SetQtyPicked(qtyPicked).
		SetStatus(pickingtaskitem.StatusDone).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *PickingTaskRepo) SetItemShort(ctx context.Context, itemID uint32, qtyPicked int64, reason string) error {
	_, err := r.entClient.Client().PickingTaskItem.UpdateOneID(itemID).
		SetQtyPicked(qtyPicked).
		SetShortReason(reason).
		SetStatus(pickingtaskitem.StatusShort).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *PickingTaskRepo) SetItemPending(ctx context.Context, itemID uint32) error {
	_, err := r.entClient.Client().PickingTaskItem.UpdateOneID(itemID).
		SetStatus(pickingtaskitem.StatusPending).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *PickingTaskRepo) LogAction(ctx context.Context, taskID uint32, itemID uint32, taskItem *ent.PickingTaskItem, action, detail, skuCode string, qty int64, createdBy *uint32, tenantID uint32) error {
	builder := r.entClient.Client().PickingTaskLog.Create().
		SetNillableTenantID(&tenantID).
		SetTaskID(taskID).
		SetNillableItemID(&itemID).
		SetNillableAction(strPtr(action)).
		SetNillableDetail(strPtr(detail)).
		SetNillableSkuCode(strPtr(skuCode)).
		SetNillableCreatedBy(createdBy).
		SetCreatedAt(time.Now())
	if qty > 0 {
		builder.SetNillableQty(&qty)
	}
	_, err := builder.Save(ctx)
	return err
}

func toTaskDTO(t *ent.PickingTask) *pickingTaskV1.PickingTask {
	dto := &pickingTaskV1.PickingTask{
		Id:         &t.ID,
		TaskNumber: t.TaskNumber,
		WaveId:     t.WaveID,
		CartId:     t.CartID,
	}
	if t.Status != nil {
		switch *t.Status {
		case pickingtask.StatusWaiting:
			dto.Status = pickingTaskV1.PickingTaskStatus_PICKING_TASK_WAITING.Enum()
		case pickingtask.StatusRunning:
			dto.Status = pickingTaskV1.PickingTaskStatus_PICKING_TASK_RUNNING.Enum()
		case pickingtask.StatusDone:
			dto.Status = pickingTaskV1.PickingTaskStatus_PICKING_TASK_DONE.Enum()
		case pickingtask.StatusCancelled:
			dto.Status = pickingTaskV1.PickingTaskStatus_PICKING_TASK_CANCELLED.Enum()
		}
	}
	if t.CreatedAt != nil {
		dto.CreatedAt = timestamppb.New(*t.CreatedAt)
	}
	if t.UpdatedAt != nil {
		dto.UpdatedAt = timestamppb.New(*t.UpdatedAt)
	}
	return dto
}

func toTaskItemDTO(t *ent.PickingTaskItem) *pickingTaskV1.PickingTaskItem {
	dto := &pickingTaskV1.PickingTaskItem{
		Id:                 &t.ID,
		TaskId:             t.TaskID,
		SkuCode:            t.SkuCode,
		ProductName:        t.ProductName,
		QtyRequired:        t.QtyRequired,
		QtyPicked:          t.QtyPicked,
		AllocatedLocationId: t.AllocatedLocationID,
		AllocatedLocationCode: t.AllocatedLocationCode,
		ShortReason:        t.ShortReason,
	}
	if t.Status != nil {
		switch *t.Status {
		case pickingtaskitem.StatusPending:
			dto.Status = pickingTaskV1.PickingTaskItemStatus_PICKING_TASK_ITEM_PENDING.Enum()
		case pickingtaskitem.StatusPicking:
			dto.Status = pickingTaskV1.PickingTaskItemStatus_PICKING_TASK_ITEM_PICKING.Enum()
		case pickingtaskitem.StatusDone:
			dto.Status = pickingTaskV1.PickingTaskItemStatus_PICKING_TASK_ITEM_DONE.Enum()
		case pickingtaskitem.StatusShort:
			dto.Status = pickingTaskV1.PickingTaskItemStatus_PICKING_TASK_ITEM_SHORT.Enum()
		}
	}
	return dto
}
