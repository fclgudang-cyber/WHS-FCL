package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "github.com/Mikdad-Force/Ghost/api/gen/go/admin/service/v1"
	settingV1 "github.com/Mikdad-Force/Ghost/api/gen/go/setting/service/v1"
)

type SettingService struct {
	adminV1.SettingServiceHTTPServer

	log *log.Helper

	settingServiceClient settingV1.SettingServiceClient
}

func NewSettingService(
	ctx *bootstrap.Context,
	settingServiceClient settingV1.SettingServiceClient,
) *SettingService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "setting/service/admin-service"))
	return &SettingService{
		log:                l,
		settingServiceClient: settingServiceClient,
	}
}

func (s *SettingService) GetWaveSetting(ctx context.Context, req *settingV1.GetWaveSettingRequest) (*settingV1.WaveSetting, error) {
	return s.settingServiceClient.GetWaveSetting(ctx, req)
}
func (s *SettingService) UpdateWaveSetting(ctx context.Context, req *settingV1.UpdateWaveSettingRequest) (*emptypb.Empty, error) {
	return s.settingServiceClient.UpdateWaveSetting(ctx, req)
}

func (s *SettingService) ListPickingBaskets(ctx context.Context, req *paginationV1.PagingRequest) (*settingV1.ListPickingBasketsResponse, error) {
	return s.settingServiceClient.ListPickingBaskets(ctx, req)
}
func (s *SettingService) CreatePickingBasket(ctx context.Context, req *settingV1.CreatePickingBasketRequest) (*emptypb.Empty, error) {
	return s.settingServiceClient.CreatePickingBasket(ctx, req)
}
func (s *SettingService) UpdatePickingBasket(ctx context.Context, req *settingV1.UpdatePickingBasketRequest) (*emptypb.Empty, error) {
	return s.settingServiceClient.UpdatePickingBasket(ctx, req)
}
func (s *SettingService) DeletePickingBasket(ctx context.Context, req *settingV1.DeletePickingBasketRequest) (*emptypb.Empty, error) {
	return s.settingServiceClient.DeletePickingBasket(ctx, req)
}

func (s *SettingService) ListSortBaskets(ctx context.Context, req *paginationV1.PagingRequest) (*settingV1.ListSortBasketsResponse, error) {
	return s.settingServiceClient.ListSortBaskets(ctx, req)
}
func (s *SettingService) GenerateSortBaskets(ctx context.Context, req *settingV1.GenerateSortBasketsRequest) (*settingV1.GenerateSortBasketsResponse, error) {
	return s.settingServiceClient.GenerateSortBaskets(ctx, req)
}
func (s *SettingService) DeleteSortBasket(ctx context.Context, req *settingV1.DeleteSortBasketRequest) (*emptypb.Empty, error) {
	return s.settingServiceClient.DeleteSortBasket(ctx, req)
}
