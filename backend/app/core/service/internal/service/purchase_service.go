package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/purchase"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/purchaseline"

	purchaseV1 "github.com/Mikdad-Force/Ghost/api/gen/go/purchase/service/v1"
	inventoryV1 "github.com/Mikdad-Force/Ghost/api/gen/go/inventory/service/v1"
)

// PurchaseService 到货通知单服务（M1 入库链：预约 → 收货 → 上架）。
//
// 职责边界：
//   - 单据 CRUD 与状态机流转（announce/cancel/close）
//   - Receive：防超收 → 生成 INCOMING 拣货单（源=SUPPLIER 虚拟位置，
//     目的=仓库接收位置）并在同事务确认 moves —— 库存经拣货 Validate 落账
//   - Putaway：防超上架 → 生成 INTERNAL 拣货单（接收位置 → 目标库位）
//     并在同事务确认 moves —— 上架经拣货 Validate 移入库位
//
// 单据数量口径（防超收/防超上架）在本服务维护；库存数量口径由
// StockPickingService.Validate 维护，两者以拣货单为唯一桥梁。
type PurchaseService struct {
	purchaseV1.UnimplementedPurchaseServiceServer

	log *log.Helper

	purchaseRepo          *data.PurchaseRepo
	locationRepo     *data.LocationRepo
	stockPickingRepo *data.StockPickingRepo
}

func NewPurchaseService(
	ctx *bootstrap.Context,
	purchaseRepo *data.PurchaseRepo,
	locationRepo *data.LocationRepo,
	stockPickingRepo *data.StockPickingRepo,
) *PurchaseService {
	return &PurchaseService{
		log:              ctx.NewLoggerHelper("purchase/service/core-service"),
		purchaseRepo:          purchaseRepo,
		locationRepo:     locationRepo,
		stockPickingRepo: stockPickingRepo,
	}
}

func (s *PurchaseService) List(ctx context.Context, req *paginationV1.PagingRequest) (*purchaseV1.ListPurchaseResponse, error) {
	return s.purchaseRepo.List(ctx, req)
}

func (s *PurchaseService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*purchaseV1.CountPurchaseResponse, error) {
	count, err := s.purchaseRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &purchaseV1.CountPurchaseResponse{Count: uint64(count)}, nil
}

func (s *PurchaseService) Get(ctx context.Context, req *purchaseV1.GetPurchaseRequest) (*purchaseV1.Purchase, error) {
	return s.purchaseRepo.Get(ctx, req)
}

func (s *PurchaseService) GetByNumber(ctx context.Context, req *purchaseV1.GetPurchaseByNumberRequest) (*purchaseV1.Purchase, error) {
	if req == nil || req.GetNumber() == "" {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}
	return s.purchaseRepo.GetByNumber(ctx, req.GetNumber())
}

func (s *PurchaseService) Create(ctx context.Context, req *purchaseV1.CreatePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	if req.Data.GetWarehouseCode() == "" {
		return nil, purchaseV1.ErrorBadRequest("warehouse_code is required")
	}

	// 单号/状态由服务端生成，忽略客户端传入值。
	req.Data.PurchaseNumber = nil
	req.Data.Status = nil

	if _, err := s.purchaseRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PurchaseService) Update(ctx context.Context, req *purchaseV1.UpdatePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil || req.GetId() == 0 {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	old, err := s.purchaseRepo.Get(ctx, &purchaseV1.GetPurchaseRequest{QueryBy: &purchaseV1.GetPurchaseRequest_Id{Id: req.GetId()}})
	if err != nil {
		return nil, err
	}

	// 仅草稿可改。
	if old.GetStatus() != purchaseV1.PurchaseStatus_PURCHASE_DRAFT {
		return nil, purchaseV1.ErrorConflict("only draft purchase can be updated")
	}

	// 单号/状态不可改。
	req.Data.PurchaseNumber = nil
	req.Data.Status = nil

	if _, err := s.purchaseRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PurchaseService) Delete(ctx context.Context, req *purchaseV1.DeletePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	old, err := s.purchaseRepo.Get(ctx, &purchaseV1.GetPurchaseRequest{QueryBy: &purchaseV1.GetPurchaseRequest_Id{Id: req.GetId()}})
	if err != nil {
		return nil, err
	}

	// 仅草稿/已取消可删。
	if old.GetStatus() != purchaseV1.PurchaseStatus_PURCHASE_DRAFT &&
		old.GetStatus() != purchaseV1.PurchaseStatus_PURCHASE_CANCELLED {
		return nil, purchaseV1.ErrorConflict("only draft or cancelled purchase can be deleted")
	}

	if err := s.purchaseRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// transition 通用状态流转（状态机校验 + 条件更新）。
func (s *PurchaseService) transition(ctx context.Context, id uint32, action string) (*emptypb.Empty, error) {
	to := purchaseActionTarget(action)

	old, err := s.purchaseRepo.Get(ctx, &purchaseV1.GetPurchaseRequest{QueryBy: &purchaseV1.GetPurchaseRequest_Id{Id: id}})
	if err != nil {
		return nil, err
	}

	from := old.GetStatus()
	if !purchaseCanTransition(from, to) {
		return nil, purchaseV1.ErrorConflict("invalid purchase status transition (%s → %s)", from, to)
	}

	tx, err := s.purchaseRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { s.purchaseRepo.FinishTx(tx, err) }()

	if err = s.purchaseRepo.TransitionStatusTx(ctx, tx, id,
		entPurchaseStatus(from), entPurchaseStatus(to)); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PurchaseService) Announce(ctx context.Context, req *purchaseV1.AnnouncePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}
	return s.transition(ctx, req.GetId(), "announce")
}

func (s *PurchaseService) Cancel(ctx context.Context, req *purchaseV1.CancelPurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}
	return s.transition(ctx, req.GetId(), "cancel")
}

func (s *PurchaseService) Close(ctx context.Context, req *purchaseV1.ClosePurchaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}
	return s.transition(ctx, req.GetId(), "close")
}

// Receive 收货（按明细增量）：
//  1. 校验单据状态（ANNOUNCED/RECEIVING）；
//  2. 防超收预检（纯函数快速失败）；
//  3. 先创建 INCOMING 拣货单（DRAFT），再在单事务内逐行原子累加已收数
//     （行级条件更新兜底并发）+ 首收迁移状态 + 确认 moves。
//
// 事务失败时拣货单停留 DRAFT（无库存影响，可人工取消）——单据数量与
// moves 状态原子一致，不存在半收账窗口。
func (s *PurchaseService) Receive(ctx context.Context, req *purchaseV1.ReceivePurchaseRequest) (resp *purchaseV1.ReceivePurchaseResponse, err error) {
	if req == nil || req.GetPurchaseId() == 0 || len(req.GetLines()) == 0 {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}
	for _, l := range req.GetLines() {
		if l.GetPurchaseLineId() == 0 || l.GetQuantity() <= 0 {
			return nil, purchaseV1.ErrorBadRequest("each receive line requires purchase_line_id and positive quantity")
		}
	}

	old, lines, err := s.purchaseRepo.GetWithLinesTx(ctx, nil, req.GetPurchaseId())
	if err != nil {
		return nil, err
	}

	if old.Status == nil || (*old.Status != purchase.StatusAnnounced && *old.Status != purchase.StatusReceiving) {
		return nil, purchaseV1.ErrorConflict("purchase is not receivable in current status")
	}

	// 防超收预检（并发下由 ApplyReceivedQtyTx 的行级条件更新兜底）。
	reqQty := make(map[uint32]int64, len(req.GetLines()))
	for _, l := range req.GetLines() {
		reqQty[l.GetPurchaseLineId()] += l.GetQuantity()
	}
	if violated := validateReceiveQuantities(purchaseLineFactsOf(lines), reqQty); len(violated) > 0 {
		return nil, purchaseV1.ErrorConflict(
			"receive exceeds expected quantity on line id(s) %v (over-receipt blocked)", violated)
	}

	// 解析位置：source = 租户供应商虚拟位置，dest = 仓库接收位置。
	sourceLocID, err := s.locationRepo.GetSupplierLocationID(ctx)
	if err != nil {
		return nil, err
	}
	destLocID, err := s.locationRepo.GetLocationID(ctx, deref(old.WarehouseCode))
	if err != nil {
		return nil, err
	}

	// 本次收货按 (SKU, 批次, 效期) 细分为 moves——批次信息随 move 走，
	// Validate 时据此 get-or-create 批次（多批次分次收互不覆盖）。
	moves := aggregateMoves(lines, req.GetLines())

	picking, err := s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			PickingType:           inventoryV1.StockPicking_INCOMING.Enum(),
			SourceLocationId:      &sourceLocID,
			DestinationLocationId: &destLocID,
			PurchaseId:                 &old.ID,
			PartnerCode:           old.PartnerCode,
			Moves:                 moves,
		},
	})
	if err != nil {
		return nil, err
	}

	tx, err := s.purchaseRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { s.purchaseRepo.FinishTx(tx, err) }()

	for _, l := range req.GetLines() {
		qc := purchaseV1.PurchaseQcStatus_QC_PASS
		if l.QcStatus != nil {
			qc = l.GetQcStatus()
		}
		var expiry *time.Time
		if l.ExpiryDate != nil {
			t := l.GetExpiryDate().AsTime()
			expiry = &t
		}
		if err = s.purchaseRepo.ApplyReceivedQtyTx(ctx, tx, l.GetPurchaseLineId(), l.GetQuantity(),
			entQcStatus(qc), l.LotName, expiry); err != nil {
			return nil, err
		}
	}

	if *old.Status == purchase.StatusAnnounced {
		if err = s.purchaseRepo.TransitionStatusTx(ctx, tx, req.GetPurchaseId(),
			entPurchaseStatus(purchaseV1.PurchaseStatus_PURCHASE_ANNOUNCED),
			entPurchaseStatus(purchaseV1.PurchaseStatus_PURCHASE_RECEIVING)); err != nil {
			return nil, err
		}
	}

	// 确认拣货单 moves（DRAFT → CONFIRMED，同事务）。
	if err = s.confirmPickingMovesTx(ctx, tx, picking); err != nil {
		return nil, err
	}

	resp = &purchaseV1.ReceivePurchaseResponse{PickingId: picking.GetId()}
	err = nil
	return resp, nil
}

// Putaway 上架（按明细增量）：
//  1. 校验单据状态（RECEIVING——ANNOUNCED 无已收库存）；
//  2. 防超上架预检（质检闸门 + 余量）；
//  3. 目标库位必须为 INTERNAL；
//  4. 按目标库位分组生成 INTERNAL 拣货单（接收位置 → 目标库位），
//     单事务内原子累加上架数 + 确认 moves。
func (s *PurchaseService) Putaway(ctx context.Context, req *purchaseV1.PutawayPurchaseRequest) (resp *purchaseV1.PutawayPurchaseResponse, err error) {
	if req == nil || req.GetPurchaseId() == 0 || len(req.GetLines()) == 0 {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}
	for _, l := range req.GetLines() {
		if l.GetPurchaseLineId() == 0 || l.GetQuantity() <= 0 || l.GetTargetLocationId() == 0 {
			return nil, purchaseV1.ErrorBadRequest("each putaway line requires purchase_line_id, positive quantity and target_location_id")
		}
	}

	old, lines, err := s.purchaseRepo.GetWithLinesTx(ctx, nil, req.GetPurchaseId())
	if err != nil {
		return nil, err
	}

	if old.Status == nil || *old.Status != purchase.StatusReceiving {
		return nil, purchaseV1.ErrorConflict("purchase must be in RECEIVING status for putaway")
	}

	// 防超上架预检（质检闸门 + 余量；并发由 ApplyPutawayQtyTx 兜底）。
	reqQty := make(map[uint32]int64, len(req.GetLines()))
	for _, l := range req.GetLines() {
		reqQty[l.GetPurchaseLineId()] += l.GetQuantity()
	}
	if violated := validatePutawayQuantities(purchaseLineFactsOf(lines), reqQty); len(violated) > 0 {
		return nil, purchaseV1.ErrorConflict(
			"putaway blocked on line id(s) %v (QC failed or exceeds received quantity)", violated)
	}

	// 目标库位必须是 INTERNAL（真实库位）。
	for _, l := range req.GetLines() {
		usage, uerr := s.locationRepo.GetUsageTx(ctx, nil, l.GetTargetLocationId())
		if uerr != nil {
			return nil, uerr
		}
		if usage != inventoryV1.StockLocation_INTERNAL {
			return nil, purchaseV1.ErrorConflict("target location %d is not an internal location", l.GetTargetLocationId())
		}
	}

	// source = 仓库接收位置；明细按目标库位分组（每组一张 INTERNAL picking）。
	sourceLocID, err := s.locationRepo.GetLocationID(ctx, deref(old.WarehouseCode))
	if err != nil {
		return nil, err
	}

	// 按目标库位分组 → 组内按 SKU 聚合 moves。
	byTarget := aggregatePutawayMoves(lines, req.GetLines())

	tx, err := s.purchaseRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { s.purchaseRepo.FinishTx(tx, err) }()

	var firstID uint32
	for target, moves := range byTarget {
		picking, perr := s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
			Data: &inventoryV1.StockPicking{
				PickingType:           inventoryV1.StockPicking_INTERNAL.Enum(),
				SourceLocationId:      &sourceLocID,
				DestinationLocationId: &target,
				PurchaseId:                 &old.ID,
				PartnerCode:           old.PartnerCode,
				Moves:                 moves,
			},
		})
		if perr != nil {
			err = perr
			return nil, err
		}
		if err = s.confirmPickingMovesTx(ctx, tx, picking); err != nil {
			return nil, err
		}
		if firstID == 0 {
			firstID = picking.GetId()
		}
	}

	for _, l := range req.GetLines() {
		if err = s.purchaseRepo.ApplyPutawayQtyTx(ctx, tx, l.GetPurchaseLineId(), l.GetQuantity()); err != nil {
			return nil, err
		}
	}

	resp = &purchaseV1.PutawayPurchaseResponse{PickingId: firstID}
	err = nil
	return resp, nil
}

// confirmPickingMovesTx 将拣货单下所有 DRAFT move 迁至 CONFIRMED（同事务，
// 镜像 StockPickingService.Confirm 的语义但不另开事务）。
func (s *PurchaseService) confirmPickingMovesTx(ctx context.Context, tx *ent.Tx, picking *inventoryV1.StockPicking) error {
	for _, move := range picking.GetMoves() {
		if move == nil || move.GetState() != inventoryV1.StockMove_DRAFT {
			continue
		}
		if err := s.stockPickingRepo.TransitionMoveStateTx(ctx, tx, move.GetId(),
			inventoryV1.StockMove_DRAFT, inventoryV1.StockMove_CONFIRMED); err != nil {
			return err
		}
	}
	return nil
}

// aggregateMoves 按 (SKU, 批次, 效期) 聚合收货 moves。
// 批次/效期事实来自本次收货请求行（ReceivePurchaseLine），而非 Purchase 行快照——
// 同一行多批分次收时 Purchase 行是"最近一次"口径，历史批次关联会丢。
// 每个组合一条 move（批次信息随 move 走，Validate 落批次表）；
// 未带批次的行归入 (sku, "", nil) 组（未跟踪库存）。
func aggregateMoves(lines []*ent.PurchaseLine, reqLines []*purchaseV1.ReceivePurchaseLine) []*inventoryV1.StockMove {
	lineByID := make(map[uint32]*ent.PurchaseLine, len(lines))
	for _, l := range lines {
		lineByID[l.ID] = l
	}

	type moveKey struct {
		sku    string
		lot    string
		expiry string
	}
	merged := make(map[moveKey]int64)
	var keyOrder []moveKey
	for _, rl := range reqLines {
		line, ok := lineByID[rl.GetPurchaseLineId()]
		if !ok {
			continue
		}
		k := moveKey{sku: deref(line.SkuCode)}
		if rl.LotName != nil {
			k.lot = rl.GetLotName()
		}
		if rl.ExpiryDate != nil {
			k.expiry = rl.GetExpiryDate().AsTime().UTC().Format(time.RFC3339)
		}
		if _, seen := merged[k]; !seen {
			keyOrder = append(keyOrder, k)
		}
		merged[k] += rl.GetQuantity()
	}

	moves := make([]*inventoryV1.StockMove, 0, len(keyOrder))
	for _, k := range keyOrder {
		qty := merged[k]
		mv := &inventoryV1.StockMove{
			ProductCode:     &k.sku,
			PlannedQuantity: &qty,
		}
		if k.lot != "" {
			mv.LotName = &k.lot
		}
		if k.expiry != "" {
			t, _ := time.Parse(time.RFC3339, k.expiry)
			mv.ExpiryDate = timestamppb.New(t)
		}
		moves = append(moves, mv)
	}
	return moves
}

// aggregatePutawayMoves 按目标库位分组聚合上架 moves：
// 组 = 目标库位，组内同 SKU 的上架数量合并为一条 move。
func aggregatePutawayMoves(lines []*ent.PurchaseLine, reqLines []*purchaseV1.PutawayPurchaseLine) map[uint32][]*inventoryV1.StockMove {
	lineByID := make(map[uint32]*ent.PurchaseLine, len(lines))
	for _, l := range lines {
		lineByID[l.ID] = l
	}

	// (target, sku) → qty
	type groupKey struct {
		target uint32
		sku    string
	}
	qtys := make(map[groupKey]int64)
	var skuOrderPerTarget = make(map[uint32][]string)

	for _, rl := range reqLines {
		line, ok := lineByID[rl.GetPurchaseLineId()]
		if !ok {
			continue
		}
		target := rl.GetTargetLocationId()
		sku := deref(line.SkuCode)
		k := groupKey{target, sku}
		if _, seen := qtys[k]; !seen {
			skuOrderPerTarget[target] = append(skuOrderPerTarget[target], sku)
		}
		qtys[k] += rl.GetQuantity()
	}

	result := make(map[uint32][]*inventoryV1.StockMove, len(skuOrderPerTarget))
	for target, skus := range skuOrderPerTarget {
		moves := make([]*inventoryV1.StockMove, 0, len(skus))
		for _, sku := range skus {
			qty := qtys[groupKey{target, sku}]
			moves = append(moves, &inventoryV1.StockMove{
				ProductCode:     &sku,
				PlannedQuantity: &qty,
			})
		}
		result[target] = moves
	}
	return result
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefUint32(v *uint32) uint32 {
	if v == nil {
		return 0
	}
	return *v
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// purchaseLineFactsOf ent 明细 → 数量口径快照（纯函数校验入参）。
func purchaseLineFactsOf(lines []*ent.PurchaseLine) []purchaseLineFacts {
	facts := make([]purchaseLineFacts, 0, len(lines))
	for _, l := range lines {
		f := purchaseLineFacts{
			ID:          l.ID,
			ExpectedQty: derefInt64(l.ExpectedQty),
			ReceivedQty: derefInt64(l.ReceivedQty),
			PutawayQty:  derefInt64(l.PutawayQty),
			QcStatus:    purchaseV1.PurchaseQcStatus_QC_PENDING,
		}
		if l.QcStatus != nil {
			switch *l.QcStatus {
			case purchaseline.QcStatusQCPass:
				f.QcStatus = purchaseV1.PurchaseQcStatus_QC_PASS
			case purchaseline.QcStatusQCFail:
				f.QcStatus = purchaseV1.PurchaseQcStatus_QC_FAIL
			}
		}
		facts = append(facts, f)
	}
	return facts
}

// entPurchaseStatus proto → ent 状态枚举。
func entPurchaseStatus(s purchaseV1.PurchaseStatus) purchase.Status {
	switch s {
	case purchaseV1.PurchaseStatus_PURCHASE_DRAFT:
		return purchase.StatusDraft
	case purchaseV1.PurchaseStatus_PURCHASE_ANNOUNCED:
		return purchase.StatusAnnounced
	case purchaseV1.PurchaseStatus_PURCHASE_RECEIVING:
		return purchase.StatusReceiving
	case purchaseV1.PurchaseStatus_PURCHASE_COMPLETED:
		return purchase.StatusCompleted
	case purchaseV1.PurchaseStatus_PURCHASE_CANCELLED:
		return purchase.StatusCancelled
	default:
		return purchase.StatusDraft
	}
}

// entQcStatus proto → ent 质检枚举。
func entQcStatus(s purchaseV1.PurchaseQcStatus) purchaseline.QcStatus {
	switch s {
	case purchaseV1.PurchaseQcStatus_QC_PASS:
		return purchaseline.QcStatusQCPass
	case purchaseV1.PurchaseQcStatus_QC_FAIL:
		return purchaseline.QcStatusQCFail
	default:
		return purchaseline.QcStatusQCPending
	}
}
