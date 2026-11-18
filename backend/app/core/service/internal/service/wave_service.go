package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/wave"

	inventoryV1 "github.com/Mikdad-Force/Ghost/api/gen/go/inventory/service/v1"
	waveV1 "github.com/Mikdad-Force/Ghost/api/gen/go/wave/service/v1"
)

// WaveService 波次服务：订单分析 SKU → Wave → cek stok → Lepaskan → 拣货任务。
type WaveService struct {
	waveV1.UnimplementedWaveServiceServer

	log *log.Helper

	waveRepo         *data.WaveRepo
	stockQuantRepo  *data.StockQuantRepo
	pickingTaskRepo *data.PickingTaskRepo
	pickingCartRepo *data.PickingCartRepo
}

func NewWaveService(
	ctx *bootstrap.Context,
	waveRepo *data.WaveRepo,
	stockQuantRepo *data.StockQuantRepo,
	pickingTaskRepo *data.PickingTaskRepo,
	pickingCartRepo *data.PickingCartRepo,
) *WaveService {
	return &WaveService{
		log:             ctx.NewLoggerHelper("wave/service/core-service"),
		waveRepo:        waveRepo,
		stockQuantRepo:  stockQuantRepo,
		pickingTaskRepo: pickingTaskRepo,
		pickingCartRepo: pickingCartRepo,
	}
}

func (s *WaveService) List(ctx context.Context, req *paginationV1.PagingRequest) (*waveV1.ListWaveResponse, error) {
	return s.waveRepo.List(ctx, req)
}

func (s *WaveService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*waveV1.CountWaveResponse, error) {
	count, err := s.waveRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &waveV1.CountWaveResponse{Count: uint64(count)}, nil
}

// Get 波次详情（含 SKU 行）。
func (s *WaveService) Get(ctx context.Context, req *waveV1.GetWaveRequest) (*waveV1.Wave, error) {
	id := req.GetId()
	if id == 0 {
		return nil, waveV1.ErrorBadRequest("invalid parameter")
	}
	dto, err := s.waveRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.waveRepo.GetItems(ctx, id)
	if err != nil {
		return nil, err
	}
	dto.Items = make([]*waveV1.WaveItem, 0, len(items))
	for _, it := range items {
		dto.Items = append(dto.Items, s.itemToDTO(it))
	}
	return dto, nil
}

// Create 兼容旧版：从 picking_ids 创建波次（保留旧流程）。
func (s *WaveService) Create(ctx context.Context, req *waveV1.CreateWaveRequest) (*emptypb.Empty, error) {
	if req == nil || len(req.GetPickingIds()) == 0 {
		return nil, waveV1.ErrorBadRequest("picking_ids are required")
	}
	callerUserID, _ := viewerUserIDFromContext(ctx)
	if _, err := s.waveRepo.CreateTx(ctx, req.GetPickingIds(), req.Remark, &callerUserID); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// AddSkus 把订单里的 SKU 加进波次（自动取/建 tenant 当前草稿波次）。
func (s *WaveService) AddSkus(ctx context.Context, req *waveV1.AddSkusRequest) (*waveV1.AddSkusResponse, error) {
	if req == nil || len(req.GetItems()) == 0 {
		return nil, waveV1.ErrorBadRequest("items are required")
	}

	tid := tenantIDFromContext(ctx)

	// 找或建 tenant 草稿波次
	var target *ent.Wave
	if id := req.GetWaveId(); id != 0 {
		targetEntity, err := s.waveRepo.GetEntity(ctx, id)
		if err != nil {
			return nil, waveV1.ErrorNotFound("wave not found")
		}
		target = targetEntity
	} else {
		target, _ = s.waveRepo.FindTenantDraftWave(ctx, tid)
	}

	created := false
	if target == nil {
		tidVal := tid
		userID, _ := viewerUserIDFromContext(ctx)
		t, err := s.waveRepo.CreateDraftWaveTx(ctx, tidVal, nil, &userID)
		if err != nil {
			return nil, err
		}
		target = t
		created = true
	}

	items := make([]*waveV1.WaveItem, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		if it == nil || it.GetSkuCode() == "" || it.GetQty() <= 0 {
			return nil, waveV1.ErrorBadRequest("each item requires skuCode and positive qty")
		}
		items = append(items, &waveV1.WaveItem{
			SkuCode:     trans.Ptr(it.GetSkuCode()),
			ProductName: trans.Ptr(it.GetProductName()),
			QtyRequired: trans.Ptr(it.GetQty()),
		})
	}

	if err := s.waveRepo.AddItemUpserts(ctx, target.ID, items, tid); err != nil {
		return nil, err
	}
	for _, it := range req.GetItems() {
		if it.GetOrderId() == 0 {
			continue
		}
		if err := s.waveRepo.AddWaveOrder(ctx, tid, target.ID, it.GetOrderId()); err != nil {
			s.log.Errorf("record wave_order failed: %s", err.Error())
		}
	}

	return &waveV1.AddSkusResponse{
		WaveId:  target.ID,
		Created: created,
	}, nil
}

// CheckWave cek stok 每个 SKU（跨 lokasi 合计 + 扣除其他波次已预留量），
// 更新 SKU 行 + 波次状态；所有 SKU 可用 → READY_TO_RELEASE。
func (s *WaveService) Check(ctx context.Context, req *waveV1.CheckWaveRequest) (*waveV1.CheckWaveResponse, error) {
	if req == nil || req.GetId() == 0 {
		return nil, waveV1.ErrorBadRequest("invalid parameter")
	}

	tid := tenantIDFromContext(ctx)

	t, items, err := s.waveRepo.GetWaveForCheck(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, waveV1.ErrorBadRequest("wave has no items")
	}

	// 1) 每个 SKU：查库的存量合计，扣除其他波次已预留量
	type alloc struct {
		id         uint32
		sku        string
		productName string
		required   int64
		allocated  int64
		// 单 lokasi 分配（最大的 lokasi 优先）
		locationID   *uint32
		locationName string
	}

	preps := make([]alloc, 0, len(items))
	for _, it := range items {
		sku := deref(it.SkuCode)
		required := derefInt64(it.QtyRequired)

		quants, err := s.stockQuantRepo.ListByProductCodeWithTenant(ctx, tid, sku)
		if err != nil {
			s.log.Errorf("query quants failed: %s", err.Error())
			quants = nil
		}

		bestAvail := int64(0)
		var bestLoc *ent.StockQuant
		totalAvailable := int64(0)
		for _, q := range quants {
			avail := derefInt64(q.Quantity) - derefInt64(q.ReservedQuantity)
			if avail < 0 {
				avail = 0
			}
			totalAvailable += avail
			if avail > bestAvail {
				bestAvail = avail
				bestLoc = q
			}
		}

		allocated := int64(0)
		availability := "UNAVAILABLE"
		if bestAvail >= required {
			allocated = required
			availability = "AVAILABLE"
		} else if totalAvailable >= required {
			// Sisa tersplit has plenty combined but no single location fits — still available (ambil multi lokasi)
			allocated = required
			availability = "AVAILABLE"
		} else if totalAvailable > 0 {
			allocated = totalAvailable
			availability = "PARTIAL"
		}

		preps = append(preps, alloc{
			id:           it.ID,
			sku:          sku,
			productName:  deref(it.ProductName),
			required:     required,
			allocated:    allocated,
			locationID:   bestLocLocationID(bestLoc),
			locationName: bestLocLocationName(bestLoc),
		})
		_ = availability
	}

	_ = bestLocLocationName
	// 2) 写回每个 SKU 行（含 status）+ reserve 库存
	readyToRelease := true
	results := make([]*waveV1.WaveItem, 0, len(preps))

	tx, err := s.waveRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	var txErr error
	defer func() { s.waveRepo.FinishTx(tx, txErr) }()

	for _, p := range preps {
		availability := "UNAVAILABLE"
		if p.allocated >= p.required {
			availability = "AVAILABLE"
		} else if p.allocated > 0 {
			availability = "PARTIAL"
			readyToRelease = false
		} else {
			availability = "UNAVAILABLE"
			readyToRelease = false
		}

		if err = s.waveRepo.UpdateItem(ctx, tx, p.id, p.allocated, availability, p.locationName, locationIDStrPtr(p.locationID)); err != nil {
			txErr = err
			return nil, txErr
		}

		// reserve stock for this wave
		if p.allocated > 0 && p.locationID != nil {
			_ = s.stockQuantRepo.ReserveForLocationTx(ctx, tx, *p.locationID, p.sku, p.allocated)
		}

		results = append(results, &waveV1.WaveItem{
			Id:                  trans.Ptr(p.id),
			SkuCode:             trans.Ptr(p.sku),
			ProductName:         trans.Ptr(p.productName),
			QtyRequired:         trans.Ptr(p.required),
			QtyAllocated:        trans.Ptr(p.allocated),
			Availability:        trans.Ptr(availability),
			AllocatedFrom:       trans.Ptr(p.locationName),
			AllocatedLocationId: locationIDStrPtr(p.locationID),
		})
	}

	newStatus := "CHECKED"
	if readyToRelease {
		newStatus = "READY_TO_RELEASE"
		if err = s.waveRepo.MarkWaveReadyTx(ctx, tx, t.ID); err != nil {
			txErr = err
			return nil, txErr
		}
	} else {
		if err = s.waveRepo.MarkWaveCheckedTx(ctx, tx, t.ID); err != nil {
			txErr = err
			return nil, txErr
		}
	}

	txErr = nil

	savedStatus := protoStatusForCheck(newStatus)

	return &waveV1.CheckWaveResponse{
		Id:             t.ID,
		Status:         savedStatus,
		ReadyToRelease: readyToRelease,
		Items:          results,
		Message:        checkMessage(readyToRelease),
	}, nil
}

func bestLocLocationID(q *ent.StockQuant) *uint32 {
	if q == nil || q.LocationID == nil {
		return nil
	}
	return q.LocationID
}

func bestLocLocationName(q *ent.StockQuant) string {
	if q == nil {
		return ""
	}
	return fmt.Sprintf("LOK%d", derefUint32(q.LocationID))
}

func locationIDStrPtr(id *uint32) *string {
	if id == nil {
		return nil
	}
	return trans.Ptr(fmt.Sprintf("%d", *id))
}

// Release lepaskan 波次 → 生成拣货任务（READY_TO_RELEASE → RELEASED）。
func (s *WaveService) Release(ctx context.Context, req *waveV1.ReleaseWaveRequest) (*waveV1.ReleaseWaveResponse, error) {
	if req == nil || req.GetId() == 0 {
		return nil, waveV1.ErrorBadRequest("invalid parameter")
	}

	// 必须是 READY_TO_RELEASE
	t, err := s.waveRepo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if t.GetStatus() != waveV1.WaveStatus_WAVE_READY_TO_RELEASE {
		return nil, waveV1.ErrorConflict("wave is not ready to release (need all SKU available)")
	}

	items, err := s.waveRepo.GetItems(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	callerUserID, _ := viewerUserIDFromContext(ctx)

	tx, err := s.waveRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	var txErr error
	defer func() { s.waveRepo.FinishTx(tx, txErr) }()

	if err = s.waveRepo.MarkWaveReleasedTx(ctx, tx, req.GetId()); err != nil {
		txErr = err
		return nil, txErr
	}

	task, err := s.pickingTaskRepo.CreateTx(ctx, tx, req.GetId(), &callerUserID)
	if err != nil {
		txErr = err
		return nil, txErr
	}

	if err = s.pickingTaskRepo.CreateItemsFromWaveItemsTx(ctx, tx, task.ID, items, &callerUserID); err != nil {
		txErr = err
		return nil, txErr
	}

	if err = s.pickingCartRepo.AssignCartToTaskTx(ctx, tx, task.ID); err != nil {
		txErr = err
		return nil, txErr
	}

	txErr = nil

	return &waveV1.ReleaseWaveResponse{
		Id:             req.GetId(),
		Status:         waveV1.WaveStatus_WAVE_RELEASED,
		PickingTaskId:  task.ID,
	}, nil
}

// Cancel 取消波次（释放 reservasi 的 SKU 行也清零）。
func (s *WaveService) Cancel(ctx context.Context, req *waveV1.CancelWaveRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, waveV1.ErrorBadRequest("invalid parameter")
	}

	tx, err := s.waveRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	var txErr error
	defer func() { s.waveRepo.FinishTx(tx, txErr) }()

	if err = s.waveRepo.TransitionStatusTx(ctx, tx, req.GetId(),
		wave.StatusDraft, wave.StatusCancelled); err != nil {
		// 尝试其他合法迁移态
		if cerr := s.waveRepo.TransitionStatusTx(ctx, tx, req.GetId(),
			wave.StatusChecked, wave.StatusCancelled); cerr == nil {
			// ok
		} else if cerr2 := s.waveRepo.TransitionStatusTx(ctx, tx, req.GetId(),
			wave.StatusReadyToRelease, wave.StatusCancelled); cerr2 == nil {
			// ok
		} else {
			txErr = err
			return nil, txErr
		}
	}

	// 释放 SKU 行的 allocated_qty
	if err = s.waveRepo.ResetAllocatedTx(ctx, tx, req.GetId()); err != nil {
		txErr = err
		return nil, txErr
	}

	txErr = nil
	return &emptypb.Empty{}, nil
}

func (s *WaveService) itemToDTO(it *ent.WaveItem) *waveV1.WaveItem {
	dto := &waveV1.WaveItem{
		Id:                   &it.ID,
		WaveId:               it.WaveID,
		SkuCode:              it.SkuCode,
		ProductName:          it.ProductName,
		QtyRequired:          it.QtyRequired,
		QtyAllocated:         it.QtyAllocated,
		Availability:         it.Availability,
		AllocatedFrom:        it.AllocatedFrom,
		AllocatedLocationId:  it.AllocatedLocationID,
	}
	return dto
}

func protoStatusForCheck(s string) waveV1.WaveStatus {
	switch s {
	case "READY_TO_RELEASE":
		return waveV1.WaveStatus_WAVE_READY_TO_RELEASE
	default:
		return waveV1.WaveStatus_WAVE_CHECKED
	}
}

func checkMessage(allReady bool) string {
	if allReady {
		return "Semua SKU tersedia — Siap Dilepas"
	}
	return "Ada SKU yang kurang/tidak tersedia — tidak bisa Dilepas"
}

var _ = time.Now
var _ = inventoryV1.ErrorBadRequest
