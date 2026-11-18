package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	settingV1 "github.com/Mikdad-Force/Ghost/api/gen/go/setting/service/v1"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingbasket"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortbasket"
		"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/wavesetting"
)

// SettingRepo 设置仓储：波次设置 + 两类 Keranjang。
type SettingRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewSettingRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *SettingRepo {
	return &SettingRepo{
		log:       ctx.NewLoggerHelper("setting/repo/core-service"),
		entClient: entClient,
	}
}

// === WaveSetting ===

func (r *SettingRepo) GetWaveSetting(ctx context.Context, tenantID uint32, waveType string) (*ent.WaveSetting, error) {
	return r.entClient.Client().WaveSetting.Query().
		Where(wavesetting.TenantID(tenantID), wavesetting.WaveTypeEQ(wavesetting.WaveType(waveType))).
		Only(ctx)
}

func (r *SettingRepo) UpsertWaveSetting(ctx context.Context, tenantID uint32, req *settingV1.UpdateWaveSettingRequest) error {
	waveType := wavesetting.WaveType(req.GetWaveType())
	create := r.entClient.Client().WaveSetting.Create().
		SetTenantID(tenantID).
		SetWaveType(waveType)
	update := r.entClient.Client().WaveSetting.Update().
		Where(wavesetting.TenantID(tenantID), wavesetting.WaveTypeEQ(wavesetting.WaveType(waveType)))

	if req.MaxOrdersPerCart != nil {
		create.SetMaxOrdersPerCart(int(req.GetMaxOrdersPerCart()))
		update.SetMaxOrdersPerCart(int(req.GetMaxOrdersPerCart()))
	}
	if req.MaxSkuPerWave != nil {
		create.SetMaxSkuPerWave(int(req.GetMaxSkuPerWave()))
		update.SetMaxSkuPerWave(int(req.GetMaxSkuPerWave()))
	}
	if req.GroupingRule != nil {
		gr := wavesetting.GroupingRule(req.GetGroupingRule())
		create.SetGroupingRule(gr)
		update.SetGroupingRule(gr)
	}
	if req.AutoSplit != nil {
		create.SetAutoSplit(req.GetAutoSplit())
		update.SetAutoSplit(req.GetAutoSplit())
	}
	if req.DefaultCartCapacity != nil {
		create.SetDefaultCartCapacity(int(req.GetDefaultCartCapacity()))
		update.SetDefaultCartCapacity(int(req.GetDefaultCartCapacity()))
	}

	exists := false
	_, err := r.entClient.Client().WaveSetting.Query().
		Where(wavesetting.TenantID(tenantID), wavesetting.WaveTypeEQ(wavesetting.WaveType(waveType))).
		Only(ctx)
	if err == nil {
		exists = true
	} else if !ent.IsNotFound(err) {
		return err
	}
	if exists {
		_, err = update.Save(ctx)
		return err
	}
	_, err = create.Save(ctx)
	return err
}

// === PickingBasket ===

func (r *SettingRepo) ListPickingBaskets(ctx context.Context, tenantID uint32) ([]*ent.PickingBasket, int, error) {
	query := r.entClient.Client().PickingBasket.Query().
		Where(
			pickingbasket.TenantID(tenantID),
			
		).
		Order(ent.Asc(pickingbasket.FieldCode))
	items, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, len(items), nil
}

func (r *SettingRepo) CreatePickingBasket(ctx context.Context, tenantID uint32, code, basketType string, capacity int) (*ent.PickingBasket, error) {
	return r.entClient.Client().PickingBasket.Create().
		SetTenantID(tenantID).
		SetCode(code).
		SetBasketType(pickingbasket.BasketType(basketType)).
		SetCapacity(capacity).
		SetStatus(pickingbasket.StatusFree).
		Save(ctx)
}

func (r *SettingRepo) UpdatePickingBasket(ctx context.Context, tenantID uint32, req *settingV1.UpdatePickingBasketRequest) error {
	update := r.entClient.Client().PickingBasket.Update().
		Where(pickingbasket.ID(req.GetId()), pickingbasket.TenantID(tenantID))
	if req.Capacity != nil {
		update.SetCapacity(int(req.GetCapacity()))
	}
	if req.Status != nil {
		update.SetStatus(pickingbasket.Status(req.GetStatus()))
	}
	if req.BasketType != nil {
		update.SetBasketType(pickingbasket.BasketType(req.GetBasketType()))
	}
	_, err := update.Save(ctx)
	return err
}

func (r *SettingRepo) DeletePickingBasket(ctx context.Context, tenantID uint32, id uint32) error {
	_, err := r.entClient.Client().PickingBasket.Delete().
		Where(pickingbasket.ID(id), pickingbasket.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (r *SettingRepo) GetPickingBasket(ctx context.Context, tenantID uint32, id uint32) (*ent.PickingBasket, error) {
	return r.entClient.Client().PickingBasket.Query().
		Where(pickingbasket.ID(id), pickingbasket.TenantID(tenantID)).
		Only(ctx)
}

// === SortBasket ===

func (r *SettingRepo) ListSortBaskets(ctx context.Context, tenantID uint32) ([]*ent.SortBasket, int, error) {
	query := r.entClient.Client().SortBasket.Query().
		Where(
			sortbasket.TenantID(tenantID),
			
		).
		Order(ent.Asc(sortbasket.FieldCode))
	items, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, len(items), nil
}

func (r *SettingRepo) ExistingSortCodes(ctx context.Context, tenantID uint32, codes []string) (map[string]bool, error) {
	rows, err := r.entClient.Client().SortBasket.Query().
		Where(sortbasket.TenantID(tenantID), sortbasket.CodeIn(codes...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(rows))
	for _, row := range rows {
		m[row.Code] = true
	}
	return m, nil
}

func (r *SettingRepo) CreateSortBasket(ctx context.Context, tenantID uint32, prefix string, number int, code string, boxCount int) (*ent.SortBasket, error) {
	return r.entClient.Client().SortBasket.Create().
		SetTenantID(tenantID).
		SetPrefix(prefix).
		SetNumber(number).
		SetCode(code).
		SetBoxCount(boxCount).
		SetStatus(sortbasket.StatusFree).
		Save(ctx)
}

func (r *SettingRepo) DeleteSortBasket(ctx context.Context, tenantID uint32, id uint32) error {
	_, err := r.entClient.Client().SortBasket.Delete().
		Where(sortbasket.ID(id), sortbasket.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (r *SettingRepo) GetSortBasket(ctx context.Context, tenantID uint32, id uint32) (*ent.SortBasket, error) {
	return r.entClient.Client().SortBasket.Query().
		Where(sortbasket.ID(id), sortbasket.TenantID(tenantID)).
		Only(ctx)
}

// ExistingPickingBasketCode 检查 code 是否已被占用（返回 true = 存在）。
func (r *SettingRepo) ExistingPickingBasketCode(ctx context.Context, tenantID uint32, code string) (bool, error) {
	n, err := r.entClient.Client().PickingBasket.Query().
		Where(pickingbasket.TenantID(tenantID), pickingbasket.Code(code)).
		Count(ctx)
	return n > 0, err
}
