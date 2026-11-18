package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	sortingV1 "github.com/Mikdad-Force/Ghost/api/gen/go/sorting/service/v1"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingbasket"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/pickingcart"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortbasket"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortsession"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortslotitem"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/sortslot"
)

// SortingService 分拣服务（仅 MULTI）。
type SortingService struct {
	sortingV1.UnimplementedSortingServiceServer

	log *log.Helper

	sortSessionRepo *data.SortSessionRepo
}

func NewSortingService(
	ctx *bootstrap.Context,
	sortSessionRepo *data.SortSessionRepo,
) *SortingService {
	return &SortingService{
		log:             ctx.NewLoggerHelper("sorting/service/core-service"),
		sortSessionRepo: sortSessionRepo,
	}
}

// === List / Get ===

func (s *SortingService) ListSessions(ctx context.Context, req *paginationV1.PagingRequest) (*sortingV1.ListSortSessionsResponse, error) {
	tid := tenantIDFromContext(ctx)
	rows, err := s.sortSessionRepo.ListSessions(ctx, tid)
	if err != nil {
		return nil, sortingV1.ErrorInternalServerError("list sessions failed")
	}
	out := &sortingV1.ListSortSessionsResponse{Total: uint64(len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, s.toSessionDTO(ctx, r, false))
	}
	return out, nil
}

func (s *SortingService) GetSession(ctx context.Context, req *sortingV1.GetSortSessionRequest) (*sortingV1.SortSession, error) {
	if req == nil || req.GetId() == 0 {
		return nil, sortingV1.ErrorBadRequest("invalid parameter")
	}
	row, err := s.sortSessionRepo.GetSession(ctx, req.GetId())
	if err != nil {
		return nil, sortingV1.ErrorNotFound("session not found")
	}
	return s.toSessionDTO(ctx, row, true), nil
}

// === StartSession ===

func (s *SortingService) StartSession(ctx context.Context, req *sortingV1.StartSortSessionRequest) (*sortingV1.StartSortSessionResponse, error) {
	if req == nil || req.GetPickingBasketCode() == "" || req.GetSortBasketCode() == "" {
		return nil, sortingV1.ErrorBadRequest("pickingBasketCode and sortBasketCode are required")
	}
	tid := tenantIDFromContext(ctx)
	userID, _ := viewerUserIDFromContext(ctx)

	// 1) keranjang sortir: harus FREE
	sb, err := s.sortSessionRepo.FindSortBasketByCode(ctx, tid, req.GetSortBasketCode())
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, sortingV1.ErrorNotFound("keranjang sortir tidak ditemukan: %s", req.GetSortBasketCode())
		}
		return nil, sortingV1.ErrorInternalServerError("query sort basket failed")
	}
	if sb.Status != nil && *sb.Status != sortbasket.StatusFree && *sb.Status != sortbasket.StatusInUse {
		return nil, sortingV1.ErrorConflict("keranjang sortir sudah dipakai")
	}

	// 2) keranjang picking: scan kode (bisa kode master baru atau cart_number alur lama)
	var basketID *uint32
	var taskID *uint32
	var waveID *uint32

	basket, berr := s.sortSessionRepo.FindBasketByCode(ctx, tid, req.GetPickingBasketCode())
	if berr == nil && basket != nil {
		basketID = &basket.ID
		if basket.Status != nil && *basket.Status != pickingbasket.StatusFree && *basket.Status != pickingbasket.StatusWaitingSort && *basket.Status != pickingbasket.StatusInUse {
			return nil, sortingV1.ErrorConflict("keranjang picking sedang dipakai tugas lain")
		}
		if basket.CurrentWaveID != nil {
			waveID = basket.CurrentWaveID
		}
	} else {
		// alur scan lama: keranjang = cart_number; cari tugas picking DONE terbaru
		cart, cerr := s.sortSessionRepo.FindPickingCartByNumber(ctx, tid, req.GetPickingBasketCode())
		if cerr != nil {
			if ent.IsNotFound(cerr) {
				return nil, sortingV1.ErrorNotFound("keranjang picking tidak ditemukan: %s", req.GetPickingBasketCode())
			}
			return nil, sortingV1.ErrorInternalServerError("query picking cart failed")
		}
		task, terr := s.sortSessionRepo.LatestDoneTaskByCart(ctx, tid, cart.ID)
		if terr != nil {
			if ent.IsNotFound(terr) {
				return nil, sortingV1.ErrorConflict("keranjang belum selesai picking (task belum DONE)")
			}
			return nil, sortingV1.ErrorInternalServerError("query picking task failed")
		}
		tidv := task.ID
		taskID = &tidv
		if task.WaveID != nil {
			waveID = task.WaveID
		}
	}

	// 3) orders MULTI dari wave ini
	if waveID == nil {
		return nil, sortingV1.ErrorConflict("keranjang belum terikat ke wave picking — selesai picking dulu")
	}
	lines, err := s.sortSessionRepo.ListMultiOrdersByWave(ctx, tid, *waveID)
	if err != nil {
		return nil, sortingV1.ErrorInternalServerError("query wave orders failed")
	}
	// group lines per order
	type orderGroup struct {
		orderID     uint32
		orderNumber string
		customer    string
		items       []data.SortOrderLine
	}
	groups := map[uint32]*orderGroup{}
	orderedIDs := make([]uint32, 0)
	for _, l := range lines {
		g, ok := groups[l.OrderID]
		if !ok {
			g = &orderGroup{orderID: l.OrderID, orderNumber: l.OrderNumber, customer: l.CustomerName}
			groups[l.OrderID] = g
			orderedIDs = append(orderedIDs, l.OrderID)
		}
		g.items = append(g.items, l)
	}
	if len(orderedIDs) == 0 {
		return nil, sortingV1.ErrorConflict("tidak ada order MULTI di keranjang ini — sortir hanya untuk MULTI")
	}
	if len(orderedIDs) > sb.BoxCount {
		return nil, sortingV1.ErrorConflict("jumlah order MULTI (%d) melebihi kotak keranjang sortir (%d)", len(orderedIDs), sb.BoxCount)
	}

	// 4) buat sesi + slots + slot items
	device := req.GetDevice()
	session, err := s.sortSessionRepo.CreateSession(ctx, &ent.SortSession{
		TenantID:          &tid,
		PickingBasketID:   basketID,
		PickingTaskID:     taskID,
		PickingCartNumber: trans.Ptr(req.GetPickingBasketCode()),
		SortBasketID:      &sb.ID,
		WaveID:            waveID,
		UserID:            userIDPtr(&userID),
		Device:            trans.Ptr(device),
	})
	if err != nil {
		return nil, sortingV1.ErrorInternalServerError("create session failed")
	}

	for i, orderID := range orderedIDs {
		g := groups[orderID]
		slot, err := s.sortSessionRepo.CreateSlot(ctx, session.ID, sb.ID, i+1, orderID)
		if err != nil {
			return nil, sortingV1.ErrorInternalServerError("create slot failed")
		}
		for _, l := range g.items {
			if err := s.sortSessionRepo.CreateSlotItem(ctx, slot.ID, l.SkuCode, l.ProductName, l.Qty); err != nil {
				return nil, sortingV1.ErrorInternalServerError("create slot item failed")
			}
		}
	}

	// 5) tandai keranjang sortir IN_USE; keranjang picking jadi WAITING_SORT (master baru)
	if err := s.sortSessionRepo.SetSortBasketStatus(ctx, sb.ID, sortbasket.StatusInUse); err != nil {
		s.log.Errorf("set sort basket IN_USE failed: %s", err.Error())
	}
	if basketID != nil {
		if err := s.sortSessionRepo.SetPickingBasketStatus(ctx, *basketID, pickingbasket.StatusWaitingSort); err != nil {
			s.log.Errorf("set picking basket WAITING_SORT failed: %s", err.Error())
		}
	}

	dto := s.toSessionDTO(ctx, session, true)
	return &sortingV1.StartSortSessionResponse{
		SessionId: session.ID,
		Session:   dto,
	}, nil
}

// === ScanSku ===

// === LockBasket / UnlockBasket ===

func (s *SortingService) LockBasket(ctx context.Context, req *sortingV1.LockBasketRequest) (*sortingV1.LockBasketResponse, error) {
	if req == nil || req.GetCode() == "" {
		return nil, sortingV1.ErrorBadRequest("code is required")
	}
	tid := tenantIDFromContext(ctx)
	code := req.GetCode()

	switch req.GetBasketType() {
	case sortingV1.BasketType_BASKET_TYPE_PICKING:
		// 1) master baru dulu
		if b, err := s.sortSessionRepo.FindBasketByCode(ctx, tid, code); err == nil && b != nil {
			st := ""
			if b.Status != nil {
				st = string(*b.Status)
			}
			if st != string(pickingbasket.StatusFree) && st != string(pickingbasket.StatusWaitingSort) && st != string(pickingbasket.StatusInUse) {
				return nil, sortingV1.ErrorConflict("keranjang picking sedang dipakai atau tidak bisa dipakai")
			}
			if st != string(pickingbasket.StatusInUse) {
				if err := s.sortSessionRepo.SetPickingBasketStatus(ctx, b.ID, pickingbasket.StatusInUse); err != nil {
					return nil, sortingV1.ErrorInternalServerError("lock picking basket failed")
				}
			}
			basketCode := ""
			if b.Code != nil {
				basketCode = *b.Code
			}
			return &sortingV1.LockBasketResponse{
				Id:         b.ID,
				Code:       basketCode,
				BasketType: "PICKING",
				Status:     string(pickingbasket.StatusInUse),
				Capacity:   trans.Ptr(int32(b.Capacity)),
				Message:    "keranjang terkunci",
			}, nil
		}
		// 2) alur lama (cart)
		cart, err := s.sortSessionRepo.FindPickingCartByNumber(ctx, tid, code)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, sortingV1.ErrorNotFound("keranjang picking tidak ditemukan: %s", code)
			}
			return nil, sortingV1.ErrorInternalServerError("query picking cart failed")
		}
		if cart.Status != nil && *cart.Status != pickingcart.StatusFree {
			return nil, sortingV1.ErrorConflict("keranjang picking sedang dipakai")
		}
		if err := s.sortSessionRepo.SetPickingCartStatus(ctx, cart.ID, "IN_USE"); err != nil {
			return nil, sortingV1.ErrorInternalServerError("lock picking cart failed")
		}
		cartCode := ""
		if cart.CartNumber != nil {
			cartCode = *cart.CartNumber
		}
		return &sortingV1.LockBasketResponse{
			Id:         cart.ID,
			Code:       cartCode,
			BasketType: "PICKING",
			Status:     "IN_USE",
			Message:    "keranjang terkunci",
		}, nil

	case sortingV1.BasketType_BASKET_TYPE_SORTING:
		sb, err := s.sortSessionRepo.FindSortBasketByCode(ctx, tid, code)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, sortingV1.ErrorNotFound("keranjang sortir tidak ditemukan: %s", code)
			}
			return nil, sortingV1.ErrorInternalServerError("query sort basket failed")
		}
		st := ""
		if sb.Status != nil {
			st = string(*sb.Status)
		}
		if st != string(sortbasket.StatusFree) {
			return nil, sortingV1.ErrorConflict("keranjang sortir sudah dipakai")
		}
		if err := s.sortSessionRepo.SetSortBasketStatus(ctx, sb.ID, sortbasket.StatusInUse); err != nil {
			return nil, sortingV1.ErrorInternalServerError("lock sort basket failed")
		}
		cap := int32(sb.BoxCount)
		return &sortingV1.LockBasketResponse{
			Id:         sb.ID,
			Code:       sb.Code,
			BasketType: "SORTING",
			Status:     string(sortbasket.StatusInUse),
			Capacity:   &cap,
			Message:    "keranjang terkunci",
		}, nil

	default:
		return nil, sortingV1.ErrorBadRequest("unknown basket type")
	}
}

func (s *SortingService) UnlockBasket(ctx context.Context, req *sortingV1.UnlockBasketRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetCode() == "" {
		return nil, sortingV1.ErrorBadRequest("code is required")
	}
	tid := tenantIDFromContext(ctx)
	code := req.GetCode()

	switch req.GetBasketType() {
	case sortingV1.BasketType_BASKET_TYPE_PICKING:
		if b, err := s.sortSessionRepo.FindBasketByCode(ctx, tid, code); err == nil && b != nil {
			count, err := s.sortSessionRepo.CountRunningSessionsByPickingBasket(ctx, tid, b.ID)
			if err == nil && count > 0 {
				return nil, sortingV1.ErrorConflict("keranjang sedang dipakai sesi sortir berjalan")
			}
			if err := s.sortSessionRepo.SetPickingBasketStatus(ctx, b.ID, pickingbasket.StatusFree); err != nil {
				return nil, sortingV1.ErrorInternalServerError("unlock picking basket failed")
			}
			return &emptypb.Empty{}, nil
		}
		cart, err := s.sortSessionRepo.FindPickingCartByNumber(ctx, tid, code)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, sortingV1.ErrorNotFound("keranjang picking tidak ditemukan: %s", code)
			}
			return nil, sortingV1.ErrorInternalServerError("query picking cart failed")
		}
		count, err := s.sortSessionRepo.CountRunningSessionsByPickingCart(ctx, tid, code)
		if err == nil && count > 0 {
			return nil, sortingV1.ErrorConflict("keranjang sedang dipakai sesi sortir berjalan")
		}
		if err := s.sortSessionRepo.SetPickingCartStatus(ctx, cart.ID, "FREE"); err != nil {
			return nil, sortingV1.ErrorInternalServerError("unlock picking cart failed")
		}
		return &emptypb.Empty{}, nil

	case sortingV1.BasketType_BASKET_TYPE_SORTING:
		sb, err := s.sortSessionRepo.FindSortBasketByCode(ctx, tid, code)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, sortingV1.ErrorNotFound("keranjang sortir tidak ditemukan: %s", code)
			}
			return nil, sortingV1.ErrorInternalServerError("query sort basket failed")
		}
		if sb.Status != nil && *sb.Status != sortbasket.StatusInUse {
			return nil, sortingV1.ErrorConflict("keranjang sortir tidak sedang terkunci (mungkin sudah lengkap/dipakai)")
		}
		count, err := s.sortSessionRepo.CountRunningSessionsBySortBasket(ctx, tid, sb.ID)
		if err == nil && count > 0 {
			return nil, sortingV1.ErrorConflict("keranjang sedang dipakai sesi sortir berjalan")
		}
		if err := s.sortSessionRepo.SetSortBasketStatus(ctx, sb.ID, sortbasket.StatusFree); err != nil {
			return nil, sortingV1.ErrorInternalServerError("unlock sort basket failed")
		}
		return &emptypb.Empty{}, nil

	default:
		return nil, sortingV1.ErrorBadRequest("unknown basket type")
	}
}

func (s *SortingService) ScanSku(ctx context.Context, req *sortingV1.ScanSkuSortingRequest) (*sortingV1.ScanSkuSortingResponse, error) {
	if req == nil || req.GetSessionId() == 0 || req.GetSkuCode() == "" {
		return nil, sortingV1.ErrorBadRequest("sessionId and skuCode are required")
	}
	tid := tenantIDFromContext(ctx)

	session, slots, err := s.loadSessionSlots(ctx, tid, req.GetSessionId())
	if err != nil {
		return nil, err
	}
	_ = session

	// kandidat: item PENDING belum penuh, prioritas slot_number terkecil
	type cand struct {
		slot          *ent.SortSlot
		item          *ent.SortSlotItem
		remainingScan int64
	}
	candidates := make([]cand, 0)
	for _, slot := range slots {
		if slot.Status != nil && *slot.Status == sortslot.StatusComplete {
			continue
		}
		items, err := s.sortSessionRepo.ListSlotItems(ctx, slot.ID)
		if err != nil {
			return nil, sortingV1.ErrorInternalServerError("list slot items failed")
		}
		for _, item := range items {
			if item.SkuCode == req.GetSkuCode() && derefInt64(item.QtyScanned) < item.QtyRequired && (item.Status == nil || *item.Status == sortslotitem.StatusPending) {
				candidates = append(candidates, cand{slot: slot, item: item, remainingScan: item.QtyRequired - derefInt64(item.QtyScanned)})
			}
		}
	}
	if len(candidates) == 0 {
		return nil, sortingV1.ErrorNotFound("SKU %s tidak ada di order mana pun di keranjang ini", req.GetSkuCode())
	}
	c := candidates[0]
	orderNumber, customerName := "", ""
	if o, err := s.sortSessionRepo.OrderByID(ctx, tid, derefUint32(c.slot.OrderID)); err == nil && o != nil {
		orderNumber = derefStr(o.OrderNumber)
		if o.CustomerName != nil {
			customerName = *o.CustomerName
		}
	}
	return &sortingV1.ScanSkuSortingResponse{
		SlotId:      c.slot.ID,
		SlotNumber:  int32(c.slot.SlotNumber),
		OrderId:     derefUint32(c.slot.OrderID),
		OrderNumber: orderNumber,
		QtyRequired: c.item.QtyRequired,
		QtyScanned:  derefInt64(c.item.QtyScanned),
		ProductName: derefStr(c.item.ProductName),
		Message:     fmt.Sprintf("Masukkan ke Kotak %d (%s)", c.slot.SlotNumber, customerName),
	}, nil
}

// === ScanSlotItem ===

func (s *SortingService) ScanSlotItem(ctx context.Context, req *sortingV1.ScanSortSlotItemRequest) (*sortingV1.ScanSortSlotItemResponse, error) {
	if req == nil || req.GetSessionId() == 0 || req.GetSlotId() == 0 || req.GetSkuCode() == "" || req.GetQty() <= 0 {
		return nil, sortingV1.ErrorBadRequest("sessionId, slotId, skuCode and qty (>0) are required")
	}
	tid := tenantIDFromContext(ctx)

	session, slots, err := s.loadSessionSlots(ctx, tid, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	var targetSlot *ent.SortSlot
	for _, slot := range slots {
		if slot.ID == req.GetSlotId() {
			targetSlot = slot
			break
		}
	}
	if targetSlot == nil {
		return nil, sortingV1.ErrorBadRequest("slot tidak termasuk sesi ini")
	}

	items, err := s.sortSessionRepo.ListSlotItems(ctx, targetSlot.ID)
	if err != nil {
		return nil, sortingV1.ErrorInternalServerError("list slot items failed")
	}
	var targetItem *ent.SortSlotItem
	for _, item := range items {
		if item.SkuCode == req.GetSkuCode() {
			targetItem = item
			break
		}
	}
	if targetItem == nil {
		return nil, sortingV1.ErrorNotFound("SKU %s tidak dibutuhkan oleh kotak ini", req.GetSkuCode())
	}

	add := req.GetQty()
	required := targetItem.QtyRequired
	newQty := derefInt64(targetItem.QtyScanned) + add
	if newQty > required {
		newQty = required // clamp — sisa diabaikan
	}

	itemStatus := sortslotitem.StatusPending
	if newQty >= required {
		itemStatus = sortslotitem.StatusDone
	}
	if err := s.sortSessionRepo.UpdateItemQty(ctx, targetItem.ID, newQty, itemStatus); err != nil {
		return nil, sortingV1.ErrorInternalServerError("update item qty failed")
	}

	slotQty := derefInt(targetSlot.QtyScanned) + add
	requiredTotal := int64(0)
	doneCount := 0
	for _, item := range items {
		var q int64
		if item.ID == targetItem.ID {
			q = newQty
		} else {
			q = derefInt64(item.QtyScanned)
		}
		slotQty += 0 // already added
		_ = q
		if item.ID == targetItem.ID {
			if newQty >= required {
				doneCount++
			}
		} else if item.Status != nil && *item.Status == sortslotitem.StatusDone {
			doneCount++
		}
		requiredTotal += item.QtyRequired
	}
	slotComplete := doneCount == len(items)
	slotStatus := sortslot.StatusFilled
	if slotComplete {
		slotStatus = sortslot.StatusComplete
	}
	if err := s.sortSessionRepo.UpdateSlotQty(ctx, targetSlot.ID, slotQty, slotStatus); err != nil {
		return nil, sortingV1.ErrorInternalServerError("update slot qty failed")
	}

	// semua slot complete?
	sessionComplete := true
	remaining := 0
	for _, slot := range slots {
		if slot.ID == targetSlot.ID {
			if !slotComplete {
				sessionComplete = false
			}
			continue
		}
		if slot.Status == nil || *slot.Status != sortslot.StatusComplete {
			sessionComplete = false
			remaining++
		}
	}

	if sessionComplete {
		if err := s.sortSessionRepo.SetSessionFinished(ctx, session.ID, sortsession.StatusCompleted); err != nil {
			s.log.Errorf("mark session completed failed: %s", err.Error())
		}
		if session.SortBasketID != nil {
			if err := s.sortSessionRepo.SetSortBasketStatus(ctx, *session.SortBasketID, sortbasket.StatusFull); err != nil {
				s.log.Errorf("set sort basket FULL failed: %s", err.Error())
			}
		}
		// bebaskan keranjang picking (baru) → FREE
		if session.PickingBasketID != nil {
			if err := s.sortSessionRepo.SetPickingBasketStatus(ctx, *session.PickingBasketID, pickingbasket.StatusFree); err != nil {
				s.log.Errorf("release picking basket failed: %s", err.Error())
			}
		}
		// tandai semua order dalam sesi → SORTED
		for _, slot := range slots {
			if slot.OrderID != nil {
				if err := s.sortSessionRepo.SetOrderStatusSorted(ctx, *slot.OrderID); err != nil {
					s.log.Errorf("set order sorted failed: %s", err.Error())
				}
			}
		}
		// bebaskan keranjang picking alur lama
		if session.PickingCartNumber != nil {
			if cart, err := s.sortSessionRepo.FindPickingCartByNumber(ctx, tid, *session.PickingCartNumber); err == nil && cart != nil {
				if err := s.sortSessionRepo.SetPickingCartStatus(ctx, cart.ID, "FREE"); err != nil {
					s.log.Errorf("release picking cart failed: %s", err.Error())
				}
			}
		}
	}

	msg := fmt.Sprintf("+%d unit ke Kotak %d", add, targetSlot.SlotNumber)
	if sessionComplete {
		msg = "Sortir selesai — semua kotak lengkap"
	}

	return &sortingV1.ScanSortSlotItemResponse{
		QtyScanned:     newQty,
		QtyRequired:    required,
		SlotComplete:   slotComplete,
		SessionComplete: sessionComplete,
		Message:         msg,
	}, nil
}

// === CancelSession ===

func (s *SortingService) CancelSession(ctx context.Context, req *sortingV1.CancelSortSessionRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetSessionId() == 0 {
		return nil, sortingV1.ErrorBadRequest("sessionId is required")
	}
	tid := tenantIDFromContext(ctx)

	session, err := s.sortSessionRepo.GetSession(ctx, req.GetSessionId())
	if err != nil {
		return nil, sortingV1.ErrorNotFound("session not found")
	}

	// reset slots + items
	slots, err := s.sortSessionRepo.ListSlots(ctx, session.ID)
	if err == nil {
		for _, slot := range slots {
			items, err := s.sortSessionRepo.ListSlotItems(ctx, slot.ID)
			if err == nil {
				for _, item := range items {
					_ = s.sortSessionRepo.UpdateItemQty(ctx, item.ID, 0, sortslotitem.StatusPending)
				}
			}
			_ = s.sortSessionRepo.UpdateSlotQty(ctx, slot.ID, 0, sortslot.StatusEmpty)
		}
	}

	if err := s.sortSessionRepo.SetSessionFinished(ctx, session.ID, sortsession.StatusCancelled); err != nil {
		return nil, sortingV1.ErrorInternalServerError("cancel session failed")
	}
	if session.SortBasketID != nil {
		_ = s.sortSessionRepo.SetSortBasketStatus(ctx, *session.SortBasketID, sortbasket.StatusFree)
	}
	if session.PickingBasketID != nil {
		_ = s.sortSessionRepo.SetPickingBasketStatus(ctx, *session.PickingBasketID, pickingbasket.StatusWaitingSort)
	}
	if session.PickingCartNumber != nil {
		if cart, err := s.sortSessionRepo.FindPickingCartByNumber(ctx, tid, *session.PickingCartNumber); err == nil && cart != nil {
			_ = s.sortSessionRepo.SetPickingCartStatus(ctx, cart.ID, "FREE")
		}
	}
	return &emptypb.Empty{}, nil
}

// === helpers ===

func (s *SortingService) loadSessionSlots(ctx context.Context, tid uint32, sessionID uint32) (*ent.SortSession, []*ent.SortSlot, error) {
	session, err := s.sortSessionRepo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, sortingV1.ErrorNotFound("session not found")
	}
	if session.TenantID != nil && *session.TenantID != tid {
		return nil, nil, sortingV1.ErrorNotFound("session not found")
	}
	if session.Status != nil && *session.Status != sortsession.StatusRunning {
		return nil, nil, sortingV1.ErrorConflict("sesi tidak sedang berjalan")
	}
	slots, err := s.sortSessionRepo.ListSlots(ctx, sessionID)
	if err != nil {
		return nil, nil, sortingV1.ErrorInternalServerError("list slots failed")
	}
	return session, slots, nil
}

func (s *SortingService) toSessionDTO(ctx context.Context, row *ent.SortSession, withSlots bool) *sortingV1.SortSession {
	status := sortingV1.SortSessionStatus_SORT_SESSION_RUNNING
	if row.Status != nil {
		switch *row.Status {
		case sortsession.StatusCompleted:
			status = sortingV1.SortSessionStatus_SORT_SESSION_COMPLETED
		case sortsession.StatusCancelled:
			status = sortingV1.SortSessionStatus_SORT_SESSION_CANCELLED
		case sortsession.StatusRunning:
			status = sortingV1.SortSessionStatus_SORT_SESSION_RUNNING
		}
	}
	dto := &sortingV1.SortSession{Status: &status}
	if row.TenantID != nil {
		dto.TenantId = row.TenantID
	}
	if row.ID > 0 {
		dto.Id = &row.ID
	}
	if row.PickingBasketID != nil {
		dto.PickingBasketId = row.PickingBasketID
	}
	if row.PickingCartNumber != nil {
		dto.PickingBasketCode = row.PickingCartNumber
	} else if row.PickingBasketID != nil {
		code := fmt.Sprintf("%d", *row.PickingBasketID)
		dto.PickingBasketCode = &code
	}
	if row.PickingTaskID != nil {
		dto.PickingTaskId = row.PickingTaskID
	}
	if row.SortBasketID != nil {
		dto.SortBasketId = row.SortBasketID
		if sb, err := s.sortSessionRepo.GetSortBasketByID(ctx, *row.SortBasketID); err == nil && sb != nil {
			dto.SortBasketCode = &sb.Code
		}
	}
	if row.WaveID != nil {
		dto.WaveId = row.WaveID
	}
	if row.Device != nil {
		dto.Device = row.Device
	}
	if row.StartedAt != nil {
		dto.StartedAt = timestamppb.New(*row.StartedAt)
	}
	if row.FinishedAt != nil {
		dto.FinishedAt = timestamppb.New(*row.FinishedAt)
	}
	if row.CreatedAt != nil {
		dto.CreatedAt = timestamppb.New(*row.CreatedAt)
	}

	if withSlots {
		slots, err := s.sortSessionRepo.ListSlots(ctx, row.ID)
		if err == nil {
			for _, slot := range slots {
				dto.Slots = append(dto.Slots, s.toSlotDTO(ctx, row, slot))
			}
		}
	}
	return dto
}

func (s *SortingService) toSlotDTO(ctx context.Context, session *ent.SortSession, slot *ent.SortSlot) *sortingV1.SortSlot {
	slotStatus := sortingV1.SortSlotStatus_SORT_SLOT_EMPTY
	if slot.Status != nil {
		switch *slot.Status {
		case sortslot.StatusFilled:
			slotStatus = sortingV1.SortSlotStatus_SORT_SLOT_FILLED
		case sortslot.StatusComplete:
			slotStatus = sortingV1.SortSlotStatus_SORT_SLOT_COMPLETE
		case sortslot.StatusPacked:
			slotStatus = sortingV1.SortSlotStatus_SORT_SLOT_PACKED
		case sortslot.StatusEmpty:
			slotStatus = sortingV1.SortSlotStatus_SORT_SLOT_EMPTY
		}
	}
	orderID := uint32(0)
	if slot.OrderID != nil {
		orderID = *slot.OrderID
	}
	orderNumber, customerName := "", ""
	if o, err := s.sortSessionRepo.OrderByID(ctx, derefUint32(session.TenantID), orderID); err == nil && o != nil {
		orderNumber = derefStr(o.OrderNumber)
		if o.CustomerName != nil {
			customerName = *o.CustomerName
		}
	}

	qd := []*sortingV1.SortSlotItem{}
	requiredTotal := int64(0)
	items, err := s.sortSessionRepo.ListSlotItems(ctx, slot.ID)
	if err == nil {
		for _, item := range items {
			requiredTotal += item.QtyRequired
			qd = append(qd, &sortingV1.SortSlotItem{
				Id:          ptrUint32(item.ID),
				SlotId:      ptrUint32(item.SlotID),
				SkuCode:     ptrString(item.SkuCode),
				ProductName: item.ProductName,
				QtyRequired: &item.QtyRequired,
				QtyScanned:  item.QtyScanned,
				Status:      ptrString(itemStatusString(item.Status)),
			})
		}
	}
	slotNum := int32(slot.SlotNumber)
	qtyScan := derefInt(slot.QtyScanned)
	return &sortingV1.SortSlot{
		Id:           ptrUint32(slot.ID),
		SessionId:    ptrUint32(session.ID),
		SortBasketId: ptrUint32(derefUint32(session.SortBasketID)),
		SlotNumber:   &slotNum,
		OrderId:      ptrUint32(orderID),
		OrderNumber:  &orderNumber,
		CustomerName: &customerName,
		Status:       &slotStatus,
		QtyScanned:   &qtyScan,
		QtyRequired:  &requiredTotal,
		Items:        qd,
	}
}

func ptrUint32(v uint32) *uint32 { return &v }
func ptrString(v string) *string  { return &v }
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func userIDPtr(v *uint32) *uint32 { return v }

func itemStatusString(s *sortslotitem.Status) string {
	if s == nil {
		return "PENDING"
	}
	return string(*s)
}

func derefInt(v *int) int64 {
	if v == nil {
		return 0
	}
	return int64(*v)
}
