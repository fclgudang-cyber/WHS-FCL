package data

import (
	"context"
	"fmt"
	"time"


	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/wave"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/waveitem"

	waveV1 "github.com/Mikdad-Force/Ghost/api/gen/go/wave/service/v1"
)

// FindTenantDraftWave 取 tenant 当前最新的 DRAFT 波次（用于 SKU 累加）。
func (r *WaveRepo) FindTenantDraftWave(ctx context.Context, tid uint32) (*ent.Wave, error) {
	t, err := r.entClient.Client().Wave.Query().
		Where(
			wave.StatusEQ(wave.StatusDraft),
			wave.TenantIDEQ(tid),
		).
		Order(ent.Desc(wave.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// CreateDraftWaveTx 创建草稿波次（SKU 累加容器）。
func (r *WaveRepo) CreateDraftWaveTx(ctx context.Context, tid uint32, remark *string, createdBy *uint32) (*ent.Wave, error) {
	waveNumber := genWaveNumber()

	builder := r.entClient.Client().Wave.Create().
		SetWaveNumber(waveNumber).
		SetStatus(wave.StatusDraft).
		SetNillableTenantID(&tid).
		SetNillableRemark(remark).
		SetNillableCreatedBy(createdBy).
		SetCreatedAt(time.Now())

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("create wave failed: %s", err.Error())
		return nil, waveV1.ErrorInternalServerError("create wave failed")
	}
	return t, nil
}

func genWaveNumber() string {
	return fmt.Sprintf("WV%d", time.Now().UnixMilli())
}

// AddWaveItemUpsert 单事务：按 SKU 累加（已存在则 qty_required += qty，否则新建）。
func (r *WaveRepo) AddItemUpserts(ctx context.Context, waveID uint32, items []*waveV1.WaveItem, tenantID uint32) error {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return waveV1.ErrorInternalServerError("start transaction failed")
	}
	var txErr error
	defer func() { r.FinishTx(tx, txErr) }()

	for _, it := range items {
		if it == nil || it.GetSkuCode() == "" || it.GetQtyRequired() <= 0 {
			txErr = waveV1.ErrorBadRequest("each wave item requires sku_code and positive qty_required")
			return txErr
		}

		existing, err := tx.WaveItem.Query().
			Where(
				waveitem.WaveIDEQ(waveID),
				waveitem.SkuCodeEQ(it.GetSkuCode()),
			).
			First(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				r.log.Errorf("query wave item failed: %s", err.Error())
				txErr = waveV1.ErrorInternalServerError("query wave item failed")
				return txErr
			}
			_, txErr = tx.WaveItem.Create().
				SetNillableTenantID(&tenantID).
				SetWaveID(waveID).
				SetNillableSkuCode(it.SkuCode).
				SetNillableProductName(it.ProductName).
				SetQtyRequired(it.GetQtyRequired()).
				SetCreatedAt(time.Now()).
				Save(ctx)
			if txErr != nil {
				r.log.Errorf("insert wave item failed: %s", txErr.Error())
				txErr = waveV1.ErrorInternalServerError("insert wave item failed")
				return txErr
			}
			continue
		}

		_, txErr = tx.WaveItem.UpdateOneID(existing.ID).
			AddQtyRequired(it.GetQtyRequired()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if txErr != nil {
			r.log.Errorf("update wave item failed: %s", txErr.Error())
			txErr = waveV1.ErrorInternalServerError("update wave item failed")
			return txErr
		}
	}

	txErr = nil
	return nil
}

// GetItems 波次 SKU 行列表。
func (r *WaveRepo) GetItems(ctx context.Context, waveID uint32) ([]*ent.WaveItem, error) {
	rows, err := r.entClient.Client().WaveItem.Query().
		Where(waveitem.WaveIDEQ(waveID)).
		Order(ent.Asc(waveitem.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query wave items failed: %s", err.Error())
		return nil, waveV1.ErrorInternalServerError("query wave items failed")
	}
	return rows, nil
}

// UpdateItem 更新单行（cek/allocate 结果）。
func (r *WaveRepo) UpdateItem(ctx context.Context, tx *ent.Tx, id uint32, allocatedQty int64, availability, allocatedFrom string, allocatedLocationID *string) error {
	builder := tx.WaveItem.UpdateOneID(id).
		SetQtyAllocated(allocatedQty).
		SetNillableAvailability(strPtr(availability)).
		SetNillableAllocatedFrom(strPtr(allocatedFrom)).
		SetNillableAllocatedLocationID(allocatedLocationID).
		SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update wave item failed: %s", err.Error())
		return waveV1.ErrorInternalServerError("update wave item failed")
	}
	return nil
}

func strPtr(s string) *string { return &s }

// SumQtyAllocatedForSku 跨波次按 (tenant, sku) 合计已分配量（排除指定 wave）。
func (r *WaveRepo) SumQtyAllocatedForSku(ctx context.Context, tenantID uint32, sku string, excludeWaveID uint32) (int64, error) {
	rows, err := r.entClient.Client().WaveItem.Query().
		Where(
			waveitem.TenantIDEQ(tenantID),
			waveitem.SkuCodeEQ(sku),
		).
		Where(waveitem.WaveIDNEQ(excludeWaveID)).
		Select(waveitem.FieldQtyAllocated).
		All(ctx)
	if err != nil {
		r.log.Errorf("sum wave items failed: %s", err.Error())
		return 0, waveV1.ErrorInternalServerError("query wave items failed")
	}
	var sum int64
	for _, row := range rows {
		sum += derefInt64(row.QtyAllocated)
	}
	return sum, nil
}

// SumQtyAllocatedForSkuTx 事务版。
func (r *WaveRepo) SumQtyAllocatedForSkuTx(ctx context.Context, tx *ent.Tx, tenantID uint32, sku string, excludeWaveID uint32) (int64, error) {
	rows, err := tx.WaveItem.Query().
		Where(
			waveitem.TenantIDEQ(tenantID),
			waveitem.SkuCodeEQ(sku),
			waveitem.WaveIDNEQ(excludeWaveID),
		).
		Select(waveitem.FieldQtyAllocated).
		All(ctx)
	if err != nil {
		r.log.Errorf("sum wave items failed: %s", err.Error())
		return 0, waveV1.ErrorInternalServerError("query wave items failed")
	}
	var sum int64
	for _, row := range rows {
		sum += derefInt64(row.QtyAllocated)
	}
	return sum, nil
}

// CountActiveWaves 统计 tenant 波次数量（用于决定 ACTIVE wave）。
func (r *WaveRepo) CountActiveWaves(ctx context.Context, tid uint32) (int, error) {
	n, err := r.entClient.Client().Wave.Query().
		Where(wave.TenantIDEQ(tid), wave.StatusIn(wave.StatusDraft, wave.StatusChecked, wave.StatusReadyToRelease, wave.StatusReleased)).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count waves failed: %s", err.Error())
		return 0, waveV1.ErrorInternalServerError("count waves failed")
	}
	return n, nil
}

// GetWaveItemsForCheck 事务内批量取 SKU 行。
func (r *WaveRepo) GetWaveItemsForCheck(ctx context.Context, waveID uint32) ([]*ent.WaveItem, error) {
	return r.GetItems(ctx, waveID)
}

// MarkWaveReadyTx 条件更新：DRAFT/CHECKED → READY_TO_RELEASE。
func (r *WaveRepo) MarkWaveReadyTx(ctx context.Context, tx *ent.Tx, id uint32) error {
	n, err := tx.Wave.Update().
		Where(
			wave.IDEQ(id),
			wave.StatusIn(wave.StatusDraft, wave.StatusChecked),
		).
		SetStatus(wave.StatusReadyToRelease).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return waveV1.ErrorInternalServerError("update wave failed")
	}
	if n == 0 {
		return waveV1.ErrorConflict("wave already released or status changed")
	}
	return nil
}

// MarkWaveCheckedTx 条件更新：DRAFT → CHECKED。
func (r *WaveRepo) MarkWaveCheckedTx(ctx context.Context, tx *ent.Tx, id uint32) error {
	n, err := tx.Wave.Update().
		Where(
			wave.IDEQ(id),
			wave.StatusEQ(wave.StatusDraft),
		).
		SetStatus(wave.StatusChecked).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return waveV1.ErrorInternalServerError("update wave failed")
	}
	if n == 0 {
		return waveV1.ErrorConflict("wave status changed concurrently or invalid")
	}
	return nil
}

// MarkWaveReleasedTx DRAFT/CHECKED/READY_TO_RELEASE → RELEASED
func (r *WaveRepo) MarkWaveReleasedTx(ctx context.Context, tx *ent.Tx, id uint32) error {
	n, err := tx.Wave.Update().
		Where(
			wave.IDEQ(id),
			wave.StatusIn(wave.StatusDraft, wave.StatusChecked, wave.StatusReadyToRelease),
		).
		SetStatus(wave.StatusReleased).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return waveV1.ErrorInternalServerError("update wave failed")
	}
	if n == 0 {
		return waveV1.ErrorConflict("wave cannot be released in current status")
	}
	return nil
}

// MarkWaveCompletedTx RELEASED → COMPLETED
func (r *WaveRepo) MarkWaveCompletedTx(ctx context.Context, tx *ent.Tx, id uint32) error {
	n, err := tx.Wave.Update().
		Where(
			wave.IDEQ(id),
			wave.StatusEQ(wave.StatusReleased),
		).
		SetStatus(wave.StatusCompleted).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return waveV1.ErrorInternalServerError("update wave failed")
	}
	if n == 0 {
		return waveV1.ErrorConflict("wave cannot be completed in current status")
	}
	return nil
}

// SumQtyAllocatedBySku 统计指定 sku 跨 active 波次合计已分配量（排除当前波次）。
func (r *WaveRepo) SumQtyAllocatedBySku(ctx context.Context, tenantID uint32, sku string, excludeWaveID uint32) (int64, error) {
	return r.SumQtyAllocatedForSku(ctx, tenantID, sku, excludeWaveID)
}

// GetWaveForCheck 查询波次+SKU 行用于 cek stok（同事务）。
func (r *WaveRepo) GetWaveForCheck(ctx context.Context, id uint32) (*ent.Wave, []*ent.WaveItem, error) {
	t, err := r.entClient.Client().Wave.Query().
		Where(wave.IDEQ(id)).
		First(ctx)
	if err != nil {
		return nil, nil, waveV1.ErrorInternalServerError("query wave failed")
	}
	items, err := r.GetItems(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return t, items, nil
}

// SumQtyAllocatedBySkuForReleaseTx 用于 cek: 对每个 sku 算可分配总量。
func (r *WaveRepo) SumQtyAllocatedBySkuTx(ctx context.Context, tx *ent.Tx, tenantID uint32, sku string, excludeWaveID uint32) (int64, error) {
	return r.SumQtyAllocatedForSkuTx(ctx, tx, tenantID, sku, excludeWaveID)
}

// ResetAllocatedTx 清空波次 SKU 行的已分配量（用于取消波次释放 reservasi）。
func (r *WaveRepo) ResetAllocatedTx(ctx context.Context, tx *ent.Tx, waveID uint32) error {
	_, err := tx.WaveItem.Update().
		Where(waveitem.WaveIDEQ(waveID)).
		SetQtyAllocated(0).
		SetAvailability("").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

// GetEntity 取波次 entity（用于服务层判断状态机/分派），绕开 proto DTO 映射。
func (r *WaveRepo) GetEntity(ctx context.Context, id uint32) (*ent.Wave, error) {
	return r.entClient.Client().Wave.Get(ctx, id)
}
