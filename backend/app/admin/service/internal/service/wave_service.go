package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "github.com/Mikdad-Force/Ghost/api/gen/go/admin/service/v1"
	waveV1 "github.com/Mikdad-Force/Ghost/api/gen/go/wave/service/v1"
)

type WaveService struct {
	adminV1.WaveServiceHTTPServer

	log *log.Helper

	waveServiceClient waveV1.WaveServiceClient
}

func NewWaveService(
	ctx *bootstrap.Context,
	waveServiceClient waveV1.WaveServiceClient,
) *WaveService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "wave/service/admin-service"))
	return &WaveService{
		log:               l,
		waveServiceClient: waveServiceClient,
	}
}

func (s *WaveService) List(ctx context.Context, req *paginationV1.PagingRequest) (*waveV1.ListWaveResponse, error) {
	return s.waveServiceClient.List(ctx, req)
}

func (s *WaveService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*waveV1.CountWaveResponse, error) {
	return s.waveServiceClient.Count(ctx, req)
}

func (s *WaveService) Get(ctx context.Context, req *waveV1.GetWaveRequest) (*waveV1.Wave, error) {
	return s.waveServiceClient.Get(ctx, req)
}

func (s *WaveService) Create(ctx context.Context, req *waveV1.CreateWaveRequest) (*emptypb.Empty, error) {
	return s.waveServiceClient.Create(ctx, req)
}

func (s *WaveService) AddSkus(ctx context.Context, req *waveV1.AddSkusRequest) (*waveV1.AddSkusResponse, error) {
	return s.waveServiceClient.AddSkus(ctx, req)
}

func (s *WaveService) Check(ctx context.Context, req *waveV1.CheckWaveRequest) (*waveV1.CheckWaveResponse, error) {
	return s.waveServiceClient.Check(ctx, req)
}

func (s *WaveService) Release(ctx context.Context, req *waveV1.ReleaseWaveRequest) (*waveV1.ReleaseWaveResponse, error) {
	return s.waveServiceClient.Release(ctx, req)
}

func (s *WaveService) Cancel(ctx context.Context, req *waveV1.CancelWaveRequest) (*emptypb.Empty, error) {
	return s.waveServiceClient.Cancel(ctx, req)
}
