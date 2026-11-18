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
)

// PickingCartRepo Keranjang 仓储：增删查 + 空闲回收。
type PickingCartRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}


func NewPickingCartRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PickingCartRepo {
	return &PickingCartRepo{
		log:       ctx.NewLoggerHelper("picking-cart/repo/core-service"),
		entClient: entClient,
	}
}

// Bootstrap helper
func (r *PickingCartRepo) BeginTx(ctx context.Context) (*ent.Tx, error) {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, err
	}
	return tx, nil
}

func (r *PickingCartRepo) FinishTx(tx *ent.Tx, err error) {
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

// List 简单列表（分页）。
func (r *PickingCartRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*PickingCartList, error) {
	if req == nil {
		req = &paginationV1.PagingRequest{Page: uint32Ptr(1), PageSize: uint32Ptr(20)}
	}
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	rows, err := r.entClient.Client().PickingCart.Query().
		Order(ent.Desc(pickingcart.FieldCreatedAt)).
		Offset((page-1)*pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return &PickingCartList{Items: rows}, nil
}

// PickingCartList 临时列表类型
type PickingCartList struct {
	Items []*ent.PickingCart
}

func (r *PickingCartRepo) Get(ctx context.Context, id uint32) (*ent.PickingCart, error) {
	return r.entClient.Client().PickingCart.Query().Where(pickingcart.IDEQ(id)).First(ctx)
}

func (r *PickingCartRepo) Create(ctx context.Context, tenantID uint32, cartNumber string, createdBy *uint32) error {
	builder := r.entClient.Client().PickingCart.Create().
		SetNillableTenantID(&tenantID).
		SetNillableCartNumber(&cartNumber).
		SetStatus(pickingcart.StatusFree).
		SetCreatedAt(time.Now())
	if createdBy != nil {
		builder = builder.SetCreatedBy(*createdBy)
	}
	_, err := builder.Save(ctx)
	return err
}

func (r *PickingCartRepo) CreateTx(ctx context.Context, tx *ent.Tx, cartNumber string, tenantID uint32, createdBy *uint32) (*ent.PickingCart, error) {
	builder := tx.PickingCart.Create().
		SetNillableTenantID(&tenantID).
		SetNillableCartNumber(&cartNumber).
		SetStatus(pickingcart.StatusFree).
		SetCreatedAt(time.Now())
	if createdBy != nil {
		builder = builder.SetCreatedBy(*createdBy)
	}
	cart, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("create picking cart failed: %s", err.Error())
		return nil, err
	}
	return cart, nil
}

func (r *PickingCartRepo) AssignCartToTaskTx(ctx context.Context, tx *ent.Tx, taskID uint32) error {
	// MVP: 不分配具体 cart，由仓库在 scan-cart 时现场分配
	return nil
}

func uint32Ptr(v uint32) *uint32 { return &v }
