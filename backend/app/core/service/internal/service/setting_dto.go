package service

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingbasket"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortbasket"

	settingV1 "github.com/Mikdad-Force/Ghost/api/gen/go/setting/service/v1"
)

func intPtrToProto(v int) *int32 {
	n := int32(v)
	return &n
}

func int32ToProtoPtr(v int32) *int32 {
	return &v
}

func boolToProtoPtr(v bool) *bool {
	return &v
}

func uint32PtrToProto(v uint32) *uint32 {
	return &v
}

func uint32PtrToUint32(v *uint32) uint32 {
	if v == nil {
		return 0
	}
	return *v
}

func pickingBasketStatusFree() string { return string(pickingbasket.StatusFree) }
func sortBasketStatusFree() string    { return string(sortbasket.StatusFree) }

func waveSettingToDTO(row *ent.WaveSetting) *settingV1.WaveSetting {
	waveType := string(row.WaveType)

	maxOrders := int32(row.MaxOrdersPerCart)
	maxSku := int32(0)
	if row.MaxSkuPerWave != nil {
		maxSku = int32(*row.MaxSkuPerWave)
	}
	gr := "PER_WAREHOUSE"
	if row.GroupingRule != nil {
		gr = string(*row.GroupingRule)
	}
	autoSplit := row.AutoSplit
	defCap := int32(row.DefaultCartCapacity)

	out := &settingV1.WaveSetting{
		WaveType:            &waveType,
		MaxOrdersPerCart:    &maxOrders,
		MaxSkuPerWave:       &maxSku,
		GroupingRule:        &gr,
		AutoSplit:           &autoSplit,
		DefaultCartCapacity: &defCap,
	}

	if row.ID > 0 {
		out.Id = uint32PtrToProto(row.ID)
	}
	if row.TenantID != nil {
		out.TenantId = uint32PtrToProto(*row.TenantID)
	}
	if row.CreatedAt != nil {
		out.CreatedAt = timestamppb.New(*row.CreatedAt)
	}
	if row.UpdatedAt != nil {
		out.UpdatedAt = timestamppb.New(*row.UpdatedAt)
	}
	return out
}

func pickingBasketToDTO(row *ent.PickingBasket) *settingV1.PickingBasket {
	code := ""
	if row.Code != nil {
		code = *row.Code
	}
	basketType := "SINGLE"
	if row.BasketType != nil {
		basketType = string(*row.BasketType)
	}
	cap := row.Capacity
	status := string(pickingbasket.StatusFree)
	if row.Status != nil {
		status = string(*row.Status)
	}
	out := &settingV1.PickingBasket{
		Code:       &code,
		BasketType: &basketType,
		Capacity:   intPtrToProto(cap),
		Status:     &status,
		Id:         uint32PtrToProto(row.ID),
	}
	if row.TenantID != nil {
		out.TenantId = uint32PtrToProto(*row.TenantID)
	}
	if row.CurrentWaveID != nil {
		out.CurrentWaveId = uint32PtrToProto(*row.CurrentWaveID)
	}
	if row.CurrentTaskID != nil {
		out.CurrentTaskId = uint32PtrToProto(*row.CurrentTaskID)
	}
	if row.CreatedAt != nil {
		out.CreatedAt = timestamppb.New(*row.CreatedAt)
	}
	if row.UpdatedAt != nil {
		out.UpdatedAt = timestamppb.New(*row.UpdatedAt)
	}
	return out
}

func sortBasketToDTO(row *ent.SortBasket) *settingV1.SortBasket {
	n := int32(row.Number)
	status := string(sortbasket.StatusFree)
	if row.Status != nil {
		status = string(*row.Status)
	}
	out := &settingV1.SortBasket{
		Id:       uint32PtrToProto(row.ID),
		Prefix:   &row.Prefix,
		Number:   &n,
		Code:     &row.Code,
		BoxCount: intPtrToProto(row.BoxCount),
		Status:   &status,
	}
	if row.TenantID != nil {
		out.TenantId = uint32PtrToProto(*row.TenantID)
	}
	if row.CreatedAt != nil {
		out.CreatedAt = timestamppb.New(*row.CreatedAt)
	}
	if row.UpdatedAt != nil {
		out.UpdatedAt = timestamppb.New(*row.UpdatedAt)
	}
	return out
}

// silence unused warnings for DTO helpers not used yet
var _ = context.Background
var _ = uint32PtrToUint32
