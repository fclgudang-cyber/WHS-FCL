package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	settingV1 "github.com/Mikdad-Force/Ghost/api/gen/go/setting/service/v1"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
)

// SettingService 波次设置 + 两类 Keranjang master。
type SettingService struct {
	settingV1.UnimplementedSettingServiceServer

	log *log.Helper

	settingRepo *data.SettingRepo
}

func NewSettingService(ctx *bootstrap.Context, settingRepo *data.SettingRepo) *SettingService {
	return &SettingService{
		log:        ctx.NewLoggerHelper("setting/service/core-service"),
		settingRepo: settingRepo,
	}
}

func settingWaveTypeOrDefault(v string) string {
	if v == "MULTI" {
		return "MULTI"
	}
	return "SINGLE"
}

// === WaveSetting ===

func (s *SettingService) GetWaveSetting(ctx context.Context, req *settingV1.GetWaveSettingRequest) (*settingV1.WaveSetting, error) {
	if req == nil {
		return nil, settingV1.ErrorBadRequest("invalid parameter")
	}
	tid := tenantIDFromContext(ctx)
	waveType := settingWaveTypeOrDefault(req.GetWaveType())

	row, err := s.settingRepo.GetWaveSetting(ctx, tid, waveType)
	if err != nil {
		if ent.IsNotFound(err) {
			// defaults without persisting
			defaultMax := int32(20)
			if waveType == "MULTI" {
				defaultMax = 10
			}
			gr := "PER_WAREHOUSE"
			return &settingV1.WaveSetting{
				WaveType:            &waveType,
				TenantId:            &tid,
				MaxOrdersPerCart:    &defaultMax,
				MaxSkuPerWave:       int32ToProtoPtr(0),
				GroupingRule:        &gr,
				AutoSplit:           boolToProtoPtr(true),
				DefaultCartCapacity: &defaultMax,
			}, nil
		}
		return nil, settingV1.ErrorInternalServerError("query wave setting failed")
	}

	return waveSettingToDTO(row), nil
}

func (s *SettingService) UpdateWaveSetting(ctx context.Context, req *settingV1.UpdateWaveSettingRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetWaveType() == "" {
		return nil, settingV1.ErrorBadRequest("wave_type is required")
	}
	tid := tenantIDFromContext(ctx)
	req.WaveType = settingWaveTypeOrDefault(req.GetWaveType())
	if err := s.settingRepo.UpsertWaveSetting(ctx, tid, req); err != nil {
		return nil, settingV1.ErrorInternalServerError("update wave setting failed")
	}
	return &emptypb.Empty{}, nil
}

// === PickingBasket ===

func (s *SettingService) ListPickingBaskets(ctx context.Context, req *paginationV1.PagingRequest) (*settingV1.ListPickingBasketsResponse, error) {
	tid := tenantIDFromContext(ctx)
	items, total, err := s.settingRepo.ListPickingBaskets(ctx, tid)
	if err != nil {
		return nil, settingV1.ErrorInternalServerError("list picking baskets failed")
	}
	out := &settingV1.ListPickingBasketsResponse{Total: uint64(total)}
	for _, row := range items {
		out.Items = append(out.Items, pickingBasketToDTO(row))
	}
	return out, nil
}

func (s *SettingService) CreatePickingBasket(ctx context.Context, req *settingV1.CreatePickingBasketRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetCode() == "" {
		return nil, settingV1.ErrorBadRequest("code is required")
	}
	if req.GetBasketType() == "" {
		return nil, settingV1.ErrorBadRequest("basket_type is required")
	}
	if req.GetBasketType() != "SINGLE" && req.GetBasketType() != "MULTI" {
		return nil, settingV1.ErrorBadRequest("basket_type must be SINGLE or MULTI")
	}
	tid := tenantIDFromContext(ctx)
	capacity := int(req.GetCapacity())
	if capacity <= 0 {
		capacity = 20
	}
	exists, err := s.settingRepo.ExistingPickingBasketCode(ctx, tid, req.GetCode())
	if err != nil {
		return nil, settingV1.ErrorInternalServerError("query picking basket failed")
	}
	if exists {
		return nil, settingV1.ErrorConflict("code keranjang sudah ada: %s", req.GetCode())
	}
	if _, err := s.settingRepo.CreatePickingBasket(ctx, tid, req.GetCode(), req.GetBasketType(), capacity); err != nil {
		return nil, settingV1.ErrorInternalServerError("create picking basket failed")
	}
	return &emptypb.Empty{}, nil
}

func (s *SettingService) UpdatePickingBasket(ctx context.Context, req *settingV1.UpdatePickingBasketRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, settingV1.ErrorBadRequest("id is required")
	}
	tid := tenantIDFromContext(ctx)
	if err := s.settingRepo.UpdatePickingBasket(ctx, tid, req); err != nil {
		return nil, settingV1.ErrorInternalServerError("update picking basket failed")
	}
	return &emptypb.Empty{}, nil
}

func (s *SettingService) DeletePickingBasket(ctx context.Context, req *settingV1.DeletePickingBasketRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, settingV1.ErrorBadRequest("id is required")
	}
	tid := tenantIDFromContext(ctx)

	row, err := s.settingRepo.GetPickingBasket(ctx, tid, req.GetId())
	if err != nil {
		return nil, settingV1.ErrorNotFound("picking basket not found")
	}
	if row.Status != nil && string(*row.Status) != pickingBasketStatusFree() {
		return nil, settingV1.ErrorBadRequest("keranjang sedang dipakai, tidak bisa dihapus")
	}
	if err := s.settingRepo.DeletePickingBasket(ctx, tid, req.GetId()); err != nil {
		return nil, settingV1.ErrorInternalServerError("delete picking basket failed")
	}
	return &emptypb.Empty{}, nil
}

// === SortBasket ===

func (s *SettingService) ListSortBaskets(ctx context.Context, req *paginationV1.PagingRequest) (*settingV1.ListSortBasketsResponse, error) {
	tid := tenantIDFromContext(ctx)
	items, total, err := s.settingRepo.ListSortBaskets(ctx, tid)
	if err != nil {
		return nil, settingV1.ErrorInternalServerError("list sort baskets failed")
	}
	out := &settingV1.ListSortBasketsResponse{Total: uint64(total)}
	for _, row := range items {
		out.Items = append(out.Items, sortBasketToDTO(row))
	}
	return out, nil
}

func (s *SettingService) GenerateSortBaskets(ctx context.Context, req *settingV1.GenerateSortBasketsRequest) (*settingV1.GenerateSortBasketsResponse, error) {
	if req == nil || req.GetPrefix() == "" || req.GetCount() <= 0 {
		return nil, settingV1.ErrorBadRequest("prefix and count (>0) are required")
	}
	prefix := req.GetPrefix()
	count := int(req.GetCount())
	startFrom := int(req.GetStartFrom())
	if startFrom <= 0 {
		startFrom = 1
	}
	boxCount := int(req.GetBoxCount())
	if boxCount <= 0 {
		boxCount = 20
	}

	tid := tenantIDFromContext(ctx)

	// tentukan digit otomatis: <100 → 2 digit, else 3+ digit
	formatNumber := func(n int) string {
		if n < 100 {
			return fmt.Sprintf("%02d", n)
		}
		return fmt.Sprintf("%d", n)
	}

	codes := make([]string, 0, count)
	numbers := make([]int, 0, count)
	for i := 0; i < count; i++ {
		n := startFrom + i
		numbers = append(numbers, n)
		codes = append(codes, prefix+formatNumber(n))
	}

	existing, err := s.settingRepo.ExistingSortCodes(ctx, tid, codes)
	if err != nil {
		return nil, settingV1.ErrorInternalServerError("query existing sort baskets failed")
	}

	created := int32(0)
	skipped := int32(0)
	createdCodes := make([]string, 0, count)
	for i, code := range codes {
		if existing[code] {
			skipped++
			continue
		}
		n := numbers[i]
		_, err := s.settingRepo.CreateSortBasket(ctx, tid, prefix, n, code, boxCount)
		if err != nil {
			return nil, settingV1.ErrorInternalServerError("create sort basket failed")
		}
		created++
		createdCodes = append(createdCodes, code)
	}

	return &settingV1.GenerateSortBasketsResponse{
		Created: created,
		Skipped: skipped,
		Codes:   createdCodes,
	}, nil
}

func (s *SettingService) DeleteSortBasket(ctx context.Context, req *settingV1.DeleteSortBasketRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, settingV1.ErrorBadRequest("id is required")
	}
	tid := tenantIDFromContext(ctx)

	row, err := s.settingRepo.GetSortBasket(ctx, tid, req.GetId())
	if err != nil {
		return nil, settingV1.ErrorNotFound("sort basket not found")
	}
	if row.Status != nil && string(*row.Status) != sortBasketStatusFree() {
		return nil, settingV1.ErrorBadRequest("keranjang sortir sedang dipakai, tidak bisa dihapus")
	}
	if err := s.settingRepo.DeleteSortBasket(ctx, tid, req.GetId()); err != nil {
		return nil, settingV1.ErrorInternalServerError("delete sort basket failed")
	}
	return &emptypb.Empty{}, nil
}
