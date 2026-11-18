package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/order"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/orderline"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingcart"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingtask"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingbasket"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortbasket"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortsession"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortslot"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortslotitem"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/waveorder"
)

// SortSessionRepo 分拣 sesi 仓储。
type SortSessionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewSortSessionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *SortSessionRepo {
	return &SortSessionRepo{
		log:       ctx.NewLoggerHelper("sort-session/repo/core-service"),
		entClient: entClient,
	}
}

// === Keranjang picking (scan) ===

func (r *SortSessionRepo) FindBasketByCode(ctx context.Context, tenantID uint32, code string) (*ent.PickingBasket, error) {
	return r.entClient.Client().PickingBasket.Query().
		Where(pickingbasket.TenantIDEQ(tenantID), pickingbasket.Code(code)).
		Only(ctx)
}

func (r *SortSessionRepo) FindPickingCartByNumber(ctx context.Context, tenantID uint32, cartNumber string) (*ent.PickingCart, error) {
	return r.entClient.Client().PickingCart.Query().
		Where(pickingcart.TenantIDEQ(tenantID), pickingcart.CartNumber(cartNumber)).
		Only(ctx)
}

func (r *SortSessionRepo) LatestDoneTaskByCart(ctx context.Context, tenantID uint32, cartID uint32) (*ent.PickingTask, error) {
	return r.entClient.Client().PickingTask.Query().
		Where(pickingtask.TenantIDEQ(tenantID), pickingtask.CartIDEQ(cartID), pickingtask.StatusEQ(pickingtask.StatusDone)).
		Order(ent.Desc(pickingtask.FieldID)).
		First(ctx)
}

func (r *SortSessionRepo) LatestDoneTaskByBasket(ctx context.Context, tenantID uint32, basketID uint32) (*ent.PickingTask, error) {
	// 新 master 尚无直接绑定时返回 nil（alur scan lama yang dipakai）
	_ = tenantID
	_ = basketID
	return nil, nil
}

// === SortBasket ===

func (r *SortSessionRepo) FindSortBasketByCode(ctx context.Context, tenantID uint32, code string) (*ent.SortBasket, error) {
	return r.entClient.Client().SortBasket.Query().
		Where(sortbasket.TenantIDEQ(tenantID), sortbasket.Code(code)).
		Only(ctx)
}

func (r *SortSessionRepo) SetSortBasketStatus(ctx context.Context, id uint32, status sortbasket.Status) error {
	_, err := r.entClient.Client().SortBasket.UpdateOneID(id).
		SetStatus(status).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *SortSessionRepo) SetPickingBasketStatus(ctx context.Context, id uint32, status pickingbasket.Status) error {
	_, err := r.entClient.Client().PickingBasket.UpdateOneID(id).
		SetStatus(status).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

// === Orders ===

type SortOrderLine struct {
	OrderID     uint32
	OrderNumber string
	CustomerName string
	SkuCode     string
	ProductName string
	Qty         int64
}

func (r *SortSessionRepo) ListMultiOrdersByWave(ctx context.Context, tenantID uint32, waveID uint32) ([]SortOrderLine, error) {
	links, err := r.entClient.Client().WaveOrder.Query().
		Where(waveorder.WaveIDEQ(waveID), waveorder.TenantIDEQ(tenantID)).
		Order(ent.Asc(waveorder.FieldOrderID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}
	orderIDs := make([]uint32, 0, len(links))
	for _, l := range links {
		orderIDs = append(orderIDs, l.OrderID)
	}

	orders, err := r.entClient.Client().Order.Query().
		Where(order.IDIn(orderIDs...), order.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]SortOrderLine, 0, len(orders)*2)
	for _, o := range orders {
		lines, err := r.entClient.Client().OrderLine.Query().
			Where(orderline.OrderIDEQ(o.ID)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		// 仅订单 MULTI（>1 jenis SKU）diikutkan
		if len(lines) <= 1 {
			continue
		}
		cn := ""
		if o.CustomerName != nil {
			cn = *o.CustomerName
		}
		for _, l := range lines {
			pn := ""
			if l.ProductName != nil {
				pn = *l.ProductName
			}
			sku := ""
			if l.SkuCode != nil {
				sku = *l.SkuCode
			}
			qty := int64(0)
			if l.Quantity != nil {
				qty = *l.Quantity
			}
			out = append(out, SortOrderLine{
			OrderID:     o.ID,
				OrderNumber: derefStrX(o.OrderNumber),
				CustomerName: cn,
				SkuCode:     sku,
				ProductName: pn,
				Qty:         qty,
			})
		}
	}
	return out, nil
}

// === SortSession ===

func (r *SortSessionRepo) CreateSession(ctx context.Context, sessionEnt *ent.SortSession) (*ent.SortSession, error) {
	builder := r.entClient.Client().SortSession.Create().
		SetTenantID(*sessionEnt.TenantID).
		SetStatus(sortsession.StatusRunning)
	if sessionEnt.PickingBasketID != nil {
		builder = builder.SetPickingBasketID(*sessionEnt.PickingBasketID)
	}
	if sessionEnt.PickingTaskID != nil {
		builder = builder.SetPickingTaskID(*sessionEnt.PickingTaskID)
	}
	if sessionEnt.PickingCartNumber != nil {
		builder = builder.SetPickingCartNumber(*sessionEnt.PickingCartNumber)
	}
	if sessionEnt.SortBasketID != nil {
		builder = builder.SetSortBasketID(*sessionEnt.SortBasketID)
	}
	if sessionEnt.WaveID != nil {
		builder = builder.SetWaveID(*sessionEnt.WaveID)
	}
	if sessionEnt.UserID != nil {
		builder = builder.SetUserID(*sessionEnt.UserID)
	}
	if sessionEnt.Device != nil {
		builder = builder.SetDevice(*sessionEnt.Device)
	}
	return builder.Save(ctx)
}

func (r *SortSessionRepo) GetSession(ctx context.Context, id uint32) (*ent.SortSession, error) {
	return r.entClient.Client().SortSession.Query().Where(sortsession.IDEQ(id)).Only(ctx)
}

func (r *SortSessionRepo) SetSessionFinished(ctx context.Context, id uint32, status sortsession.Status) error {
	_, err := r.entClient.Client().SortSession.UpdateOneID(id).
		SetStatus(status).
		SetUpdatedAt(time.Now()).
		SetNillableFinishedAt(transTimePtr(time.Now())).
		Save(ctx)
	return err
}

func (r *SortSessionRepo) ListSessions(ctx context.Context, tenantID uint32) ([]*ent.SortSession, error) {
	return r.entClient.Client().SortSession.Query().
		Where(sortsession.TenantIDEQ(tenantID), sortsession.IDNEQ(0)).
		Order(ent.Desc(sortsession.FieldID)).
		Limit(100).
		All(ctx)
}

// === Slots ===

func (r *SortSessionRepo) CreateSlot(ctx context.Context, sessionID uint32, sortBasketID uint32, slotNumber int, orderID uint32) (*ent.SortSlot, error) {
	return r.entClient.Client().SortSlot.Create().
		SetSessionID(sessionID).
		SetSortBasketID(sortBasketID).
		SetSlotNumber(slotNumber).
		SetOrderID(orderID).
		SetStatus(sortslot.StatusEmpty).
		SetQtyScanned(0).
		Save(ctx)
}

func (r *SortSessionRepo) CreateSlotItem(ctx context.Context, slotID uint32, skuCode, productName string, qtyRequired int64) error {
	_, err := r.entClient.Client().SortSlotItem.Create().
		SetSlotID(slotID).
		SetSkuCode(skuCode).
		SetNillableProductName(strPtr(productName)).
		SetQtyRequired(qtyRequired).
		SetQtyScanned(0).
		SetStatus(sortslotitem.StatusPending).
		Save(ctx)
	return err
}

func (r *SortSessionRepo) ListSlots(ctx context.Context, sessionID uint32) ([]*ent.SortSlot, error) {
	return r.entClient.Client().SortSlot.Query().
		Where(sortslot.SessionIDEQ(sessionID)).
		Order(ent.Asc(sortslot.FieldSlotNumber)).
		All(ctx)
}

func (r *SortSessionRepo) GetSlot(ctx context.Context, id uint32) (*ent.SortSlot, error) {
	return r.entClient.Client().SortSlot.Query().Where(sortslot.IDEQ(id)).Only(ctx)
}

func (r *SortSessionRepo) ListSlotItems(ctx context.Context, slotID uint32) ([]*ent.SortSlotItem, error) {
	return r.entClient.Client().SortSlotItem.Query().
		Where(sortslotitem.SlotID(slotID)).
		All(ctx)
}

func (r *SortSessionRepo) UpdateSlotQty(ctx context.Context, slotID uint32, qtyScanned int64, status sortslot.Status) error {
	_, err := r.entClient.Client().SortSlot.UpdateOneID(slotID).
		SetQtyScanned(int(qtyScanned)).
		SetStatus(status).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *SortSessionRepo) UpdateItemQty(ctx context.Context, itemID uint32, qtyScanned int64, status sortslotitem.Status) error {
	_, err := r.entClient.Client().SortSlotItem.UpdateOneID(itemID).
		SetQtyScanned(qtyScanned).
		SetStatus(status).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *SortSessionRepo) OrderByID(ctx context.Context, tenantID uint32, orderID uint32) (*ent.Order, error) {
	return r.entClient.Client().Order.Query().
		Where(order.ID(orderID), order.TenantIDEQ(tenantID)).
		Only(ctx)
}

func (r *SortSessionRepo) SetOrderStatusSorted(ctx context.Context, orderID uint32) error {
	_, err := r.entClient.Client().Order.UpdateOneID(orderID).
		SetStatus(order.StatusSorted).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *SortSessionRepo) GetSortBasketByID(ctx context.Context, id uint32) (*ent.SortBasket, error) {
	return r.entClient.Client().SortBasket.Query().Where(sortbasket.IDEQ(id)).Only(ctx)
}

// transTimePtr helper for Nillable time.
func transTimePtr(t time.Time) *time.Time {
	return &t
}

// derefStrX deref string pointer (local helper; stock_quant_repo has derefStr in different package scope).
func derefStrX(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// === Lock/Unlock ===

func (r *SortSessionRepo) SetPickingCartStatus(ctx context.Context, id uint32, status string) error {
	_, err := r.entClient.Client().PickingCart.UpdateOneID(id).
		SetStatus(pickingcart.Status(status)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *SortSessionRepo) CountRunningSessionsByPickingCart(ctx context.Context, tenantID uint32, cartNumber string) (int, error) {
	return r.entClient.Client().SortSession.Query().
		Where(
			sortsession.TenantIDEQ(tenantID),
			sortsession.StatusEQ(sortsession.StatusRunning),
			sortsession.PickingCartNumber(cartNumber),
		).
		Count(ctx)
}

func (r *SortSessionRepo) CountRunningSessionsBySortBasket(ctx context.Context, tenantID uint32, sortBasketID uint32) (int, error) {
	return r.entClient.Client().SortSession.Query().
		Where(
			sortsession.TenantIDEQ(tenantID),
			sortsession.StatusEQ(sortsession.StatusRunning),
			sortsession.SortBasketID(sortBasketID),
		).
		Count(ctx)
}

func (r *SortSessionRepo) CountRunningSessionsByPickingBasket(ctx context.Context, tenantID uint32, basketID uint32) (int, error) {
	return r.entClient.Client().SortSession.Query().
		Where(
			sortsession.TenantIDEQ(tenantID),
			sortsession.StatusEQ(sortsession.StatusRunning),
			sortsession.PickingBasketID(basketID),
		).
		Count(ctx)
}
