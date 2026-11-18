package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/purchase"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/purchaseline"
	"github.com/Mikdad-Force/Ghost/app/core/service/internal/data/ent/predicate"

	purchaseV1 "github.com/Mikdad-Force/Ghost/api/gen/go/purchase/service/v1"
)

// PurchaseRepo 到货通知单仓储：单据 CRUD + 收货/上架的事务内原子数量操作。
// 防超收/防超上架依赖数据库行级条件更新（UPDATE ... WHERE 余量足够），
// 并发下由受影响行数判定成败，无检查-写入竞态窗口。
type PurchaseRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	statusConverter *mapper.EnumTypeConverter[purchaseV1.PurchaseStatus, purchase.Status]
	qcConverter     *mapper.EnumTypeConverter[purchaseV1.PurchaseQcStatus, purchaseline.QcStatus]

	mapper *mapper.CopierMapper[purchaseV1.Purchase, ent.Purchase]

	repository *entCrud.Repository[
		ent.PurchaseQuery, ent.PurchaseSelect,
		ent.PurchaseCreate, ent.PurchaseCreateBulk,
		ent.PurchaseUpdate, ent.PurchaseUpdateOne,
		ent.PurchaseDelete,
		predicate.Purchase,
		purchaseV1.Purchase, ent.Purchase,
	]
}

func NewPurchaseRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PurchaseRepo {
	repo := &PurchaseRepo{
		log:       ctx.NewLoggerHelper("purchase/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *PurchaseRepo) init() {
	r.statusConverter = mapper.NewEnumTypeConverter[purchaseV1.PurchaseStatus, purchase.Status](purchaseV1.PurchaseStatus_name, purchaseV1.PurchaseStatus_value)
	r.qcConverter = mapper.NewEnumTypeConverter[purchaseV1.PurchaseQcStatus, purchaseline.QcStatus](purchaseV1.PurchaseQcStatus_name, purchaseV1.PurchaseQcStatus_value)

	r.mapper = mapper.NewCopierMapper[purchaseV1.Purchase, ent.Purchase]()

	r.repository = entCrud.NewRepository[
		ent.PurchaseQuery, ent.PurchaseSelect,
		ent.PurchaseCreate, ent.PurchaseCreateBulk,
		ent.PurchaseUpdate, ent.PurchaseUpdateOne,
		ent.PurchaseDelete,
		predicate.Purchase,
		purchaseV1.Purchase, ent.Purchase,
	](r.mapper)

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// purchaseStatusToProto ent 状态 → proto 状态。两侧枚举字符串不同（ent 存
// "DRAFT"，proto 为 "PURCHASE_DRAFT"——NamedValues 与 proto value 前缀差异），
// 不能走通用 EnumTypeConverter（valueMap 按 proto 值建，查 ent 值必 miss）。
func purchaseStatusToProto(s purchase.Status) *purchaseV1.PurchaseStatus {
	switch s {
	case purchase.StatusDraft:
		return purchaseV1.PurchaseStatus_PURCHASE_DRAFT.Enum()
	case purchase.StatusAnnounced:
		return purchaseV1.PurchaseStatus_PURCHASE_ANNOUNCED.Enum()
	case purchase.StatusReceiving:
		return purchaseV1.PurchaseStatus_PURCHASE_RECEIVING.Enum()
	case purchase.StatusCompleted:
		return purchaseV1.PurchaseStatus_PURCHASE_COMPLETED.Enum()
	case purchase.StatusCancelled:
		return purchaseV1.PurchaseStatus_PURCHASE_CANCELLED.Enum()
	default:
		return nil
	}
}

func timeToTimestamppb(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

func derefUint32(v *uint32) uint32 {
	if v == nil {
		return 0
	}
	return *v
}

func (r *PurchaseRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Purchase.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, purchaseV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PurchaseRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (ret *purchaseV1.ListPurchaseResponse, err error) {
	if req == nil {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Purchase.Query()

	ret2, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret2 == nil {
		return &purchaseV1.ListPurchaseResponse{Total: 0, Items: nil}, nil
	}

	r.backfillStatus(ctx, ret2.Items)
	if err = r.attachLines(ctx, ret2.Items); err != nil {
		return nil, err
	}

	_ = ret2
	return &purchaseV1.ListPurchaseResponse{
		Total: ret2.Total,
		Items: ret2.Items,
	}, nil
}

func (r *PurchaseRepo) Get(ctx context.Context, req *purchaseV1.GetPurchaseRequest) (dto *purchaseV1.Purchase, err error) {
	if req == nil {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Purchase.Query()

	var whereCond []func(s *sql.Selector)
	if id := req.GetId(); id != 0 {
		whereCond = append(whereCond, purchase.IDEQ(id))
	}

	dto, err = r.repository.Get(ctx, builder, nil, whereCond...)
	if err != nil {
		return nil, err
	}

	r.backfillStatus(ctx, []*purchaseV1.Purchase{dto})
	if err = r.attachLines(ctx, []*purchaseV1.Purchase{dto}); err != nil {
		return nil, err
	}

	return dto, nil
}

func (r *PurchaseRepo) GetByNumber(ctx context.Context, number string) (dto *purchaseV1.Purchase, err error) {
	if number == "" {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Purchase.Query().
		Where(purchase.PurchaseNumberEQ(number))

	dto, err = r.repository.Get(ctx, builder, nil)
	if err != nil {
		return nil, err
	}

	r.backfillStatus(ctx, []*purchaseV1.Purchase{dto})
	if err = r.attachLines(ctx, []*purchaseV1.Purchase{dto}); err != nil {
		return nil, err
	}

	return dto, nil
}

// backfillStatus 回填单据状态/审计字段：ent 的 Nillable 字段是指针类型，
// CopierMapper 注册的值类型枚举转换 pair 匹配不到指针字段，status 会被
// 静默丢弃——这里按 id 从 ent 批量取回补上（List/Get/Create/Update 统一走此路径）。
func (r *PurchaseRepo) backfillStatus(ctx context.Context, items []*purchaseV1.Purchase) {
	if len(items) == 0 {
		return
	}

	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.GetId())
	}

	rows, err := r.entClient.Client().Purchase.Query().
		Where(purchase.IDIn(ids...)).
		Select(purchase.FieldID, purchase.FieldStatus, purchase.FieldExpectedDate).
		All(ctx)
	if err != nil {
		r.log.Errorf("backfill purchase status failed: %s", err.Error())
		return
	}

	byID := make(map[uint32]*ent.Purchase, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	for _, it := range items {
		row, ok := byID[it.GetId()]
		if !ok {
			continue
		}
		if row.Status != nil {
			it.Status = purchaseStatusToProto(*row.Status)
		}
		if row.ExpectedDate != nil && it.ExpectedDate == nil {
			it.ExpectedDate = timeToTimestamppb(*row.ExpectedDate)
		}
	}
}

// attachLines 批量装配明细（按 purchase_id 一次查询，按单分组回填）。
func (r *PurchaseRepo) attachLines(ctx context.Context, items []*purchaseV1.Purchase) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.GetId())
	}

	lines, err := r.entClient.Client().PurchaseLine.Query().
		Where(purchaseline.PurchaseIDIn(ids...)).
		Order(ent.Asc(purchaseline.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query purchase lines failed: %s", err.Error())
		return purchaseV1.ErrorInternalServerError("query purchase lines failed")
	}

	byPurchase := make(map[uint32][]*purchaseV1.PurchaseLine, len(items))
	for _, l := range lines {
		dto := r.lineToDTO(l)
		tid := derefUint32(l.PurchaseID)
		byPurchase[tid] = append(byPurchase[tid], dto)
	}
	for _, it := range items {
		it.Lines = append(it.Lines, byPurchase[it.GetId()]...)
	}

	return nil
}

func (r *PurchaseRepo) lineToDTO(t *ent.PurchaseLine) *purchaseV1.PurchaseLine {
	dto := &purchaseV1.PurchaseLine{
		Id:          &t.ID,
		PurchaseId:       t.PurchaseID,
		SkuCode:     t.SkuCode,
		ExpectedQty: t.ExpectedQty,
		ReceivedQty: t.ReceivedQty,
		PutawayQty:  t.PutawayQty,
		LotName:     t.LotName,
	}
	if t.ExpiryDate != nil {
		dto.ExpiryDate = timestamppb.New(*t.ExpiryDate)
	}
	if t.QcStatus != nil {
		if v := r.qcConverter.ToDTO(t.QcStatus); v != nil {
			dto.QcStatus = v
		}
	}
	return dto
}

func (r *PurchaseRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Purchase.Query().
		Where(purchase.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, purchaseV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// Create 创建到货通知单（草稿）+ 嵌套明细（单事务）。
// 单号服务端生成（Purchase + 毫秒时间戳），忽略客户端传入值。
func (r *PurchaseRepo) Create(ctx context.Context, req *purchaseV1.CreatePurchaseRequest) (_ *purchaseV1.Purchase, err error) {
	if req == nil || req.Data == nil {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	purchaseNumber := "Purchase" + fmt.Sprintf("%d", time.Now().UnixMilli())

	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, purchaseV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() { r.FinishTx(tx, err) }()

	builder := tx.Purchase.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableWarehouseCode(req.Data.WarehouseCode).
		SetNillablePartnerCode(req.Data.PartnerCode).
		SetStatus(purchase.StatusDraft).
		SetPurchaseNumber(purchaseNumber).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.ExpectedDate != nil {
		builder.SetExpectedDate(req.Data.ExpectedDate.AsTime())
	}
	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert purchase failed: %s", err.Error())
		return nil, purchaseV1.ErrorInternalServerError("insert purchase failed")
	}

	if err = r.createLines(ctx, tx, t.ID, derefUint32(t.TenantID), req.Data.Lines, req.Data.CreatedBy); err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(t)
	dto.Status = purchaseV1.PurchaseStatus_PURCHASE_DRAFT.Enum()
	if err = r.attachLinesTx(ctx, tx, []*purchaseV1.Purchase{dto}); err != nil {
		return nil, err
	}

	return dto, nil
}

// createLines 批量创建明细。
func (r *PurchaseRepo) createLines(ctx context.Context, tx *ent.Tx, purchaseID uint32, tenantID uint32, lines []*purchaseV1.PurchaseLine, createdBy *uint32) error {
	if len(lines) == 0 {
		return purchaseV1.ErrorBadRequest("purchase lines are required")
	}

	for _, l := range lines {
		if l == nil || l.GetSkuCode() == "" || l.GetExpectedQty() <= 0 {
			return purchaseV1.ErrorBadRequest("each purchase line requires sku_code and positive expected_qty")
		}

		lb := tx.PurchaseLine.Create().
			SetNillableTenantID(&tenantID).
			SetPurchaseID(purchaseID).
			SetNillableSkuCode(l.SkuCode).
			SetExpectedQty(l.GetExpectedQty()).
			SetReceivedQty(0).
			SetPutawayQty(0).
			SetNillableLotName(l.LotName).
			SetNillableRemark(l.Remark).
			SetNillableCreatedBy(createdBy).
			SetCreatedAt(time.Now())
		if l.QcStatus != nil {
			entQc := r.qcConverter.ToEntity(l.QcStatus)
			if entQc == nil {
				return purchaseV1.ErrorBadRequest("invalid qc_status")
			}
			lb.SetQcStatus(*entQc)
		}
		if _, err := lb.Save(ctx); err != nil {
			r.log.Errorf("insert purchase line failed: %s", err.Error())
			return purchaseV1.ErrorInternalServerError("insert purchase line failed")
		}
	}

	return nil
}

// Update 更新单据头 + 整体替换明细（仅草稿；状态机校验在服务层）。
func (r *PurchaseRepo) Update(ctx context.Context, req *purchaseV1.UpdatePurchaseRequest) (result *purchaseV1.Purchase, err error) {
	if req == nil || req.Data == nil {
		return nil, purchaseV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Purchase.UpdateOneID(req.GetId())
	builder.Where(purchase.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(purchase.TenantIDEQ(tid))
	}

	if req.Data.ExpectedDate != nil {
		builder.SetExpectedDate(req.Data.ExpectedDate.AsTime())
	}

	result, err = r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *purchaseV1.Purchase) {
			builder.
				SetNillableWarehouseCode(req.Data.WarehouseCode).
				SetNillablePartnerCode(req.Data.PartnerCode).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			// updated_by 强制由服务端 viewer context 推导，忽略客户端传入值
			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(purchase.FieldID, req.GetId()))
		},
	)
	if err != nil {
		return nil, err
	}

	// 明细整体替换（仅草稿；调用方保证）。单事务内先删后建。
	if req.Data.Lines != nil {
		tx, terr := r.entClient.Client().Tx(ctx)
		if terr != nil {
			return nil, purchaseV1.ErrorInternalServerError("start transaction failed")
		}
		defer func() { r.FinishTx(tx, err) }()

		if _, err = tx.PurchaseLine.Delete().
			Where(purchaseline.PurchaseIDEQ(req.GetId())).
			Exec(ctx); err != nil {
			r.log.Errorf("delete purchase lines failed: %s", err.Error())
			return nil, purchaseV1.ErrorInternalServerError("delete purchase lines failed")
		}

		if err = r.createLines(ctx, tx, req.GetId(), result.GetTenantId(), req.Data.Lines, result.CreatedBy); err != nil {
			return nil, err
		}
	}

	r.backfillStatus(ctx, []*purchaseV1.Purchase{result})
	if err = r.attachLines(ctx, []*purchaseV1.Purchase{result}); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *PurchaseRepo) Delete(ctx context.Context, req *purchaseV1.DeletePurchaseRequest) error {
	if req == nil {
		return purchaseV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)

	// 级联删除明细
	lineDel := r.entClient.Client().PurchaseLine.Delete().
		Where(purchaseline.PurchaseIDEQ(req.GetId()))
	if hasTenant {
		lineDel.Where(purchaseline.TenantIDEQ(tid))
	}
	if _, err := lineDel.Exec(ctx); err != nil {
		r.log.Errorf("delete purchase lines failed: %s", err.Error())
		return purchaseV1.ErrorInternalServerError("delete failed")
	}

	delBuilder := r.entClient.Client().Purchase.Delete()
	delBuilder.Where(purchase.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(purchase.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return purchaseV1.ErrorNotFound("purchase not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return purchaseV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

// ═══════════════════════ 收货/上架事务操作 ═══════════════════════

// GetWithLinesTx 事务内取单据 + 明细（收货/上架主流程用）。
func (r *PurchaseRepo) GetWithLinesTx(ctx context.Context, tx *ent.Tx, id uint32) (*ent.Purchase, []*ent.PurchaseLine, error) {
	client := r.entClient.Client()
	if tx != nil {
		client = tx.Client()
	}

	t, err := client.Purchase.Query().
		Where(purchase.IDEQ(id)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, purchaseV1.ErrorNotFound("purchase not found")
		}
		r.log.Errorf("query purchase failed: %s", err.Error())
		return nil, nil, purchaseV1.ErrorInternalServerError("query purchase failed")
	}

	lines, err := client.PurchaseLine.Query().
		Where(purchaseline.PurchaseIDEQ(id)).
		Order(ent.Asc(purchaseline.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query purchase lines failed: %s", err.Error())
		return nil, nil, purchaseV1.ErrorInternalServerError("query purchase lines failed")
	}

	return t, lines, nil
}

// attachLinesTx 事务内装配明细（Create 主流程用，读未提交数据）。
func (r *PurchaseRepo) attachLinesTx(ctx context.Context, tx *ent.Tx, items []*purchaseV1.Purchase) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]uint32, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.GetId())
	}

	lines, err := tx.PurchaseLine.Query().
		Where(purchaseline.PurchaseIDIn(ids...)).
		Order(ent.Asc(purchaseline.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query purchase lines failed: %s", err.Error())
		return purchaseV1.ErrorInternalServerError("query purchase lines failed")
	}

	byPurchase := make(map[uint32][]*purchaseV1.PurchaseLine, len(items))
	for _, l := range lines {
		tid := derefUint32(l.PurchaseID)
		byPurchase[tid] = append(byPurchase[tid], r.lineToDTO(l))
	}
	for _, it := range items {
		it.Lines = append(it.Lines, byPurchase[it.GetId()]...)
	}

	return nil
}

// TransitionStatusTx 状态机流转（服务端条件更新：仅 from 态可迁 to 态，
// 并发下 0 行命中即冲突）。
func (r *PurchaseRepo) TransitionStatusTx(ctx context.Context, tx *ent.Tx, id uint32, from, to purchase.Status) error {
	n, err := tx.Purchase.Update().
		Where(
			purchase.IDEQ(id),
			purchase.StatusEQ(from),
		).
		SetStatus(to).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("transition purchase status failed: %s", err.Error())
		return purchaseV1.ErrorInternalServerError("transition purchase status failed")
	}
	if n == 0 {
		return purchaseV1.ErrorConflict("purchase status changed concurrently or invalid transition")
	}
	return nil
}

// ApplyReceivedQtyTx 原子累加收货数：仅当 累计收货 + delta ≤ 预期 时生效
// （防超收，行级条件更新）。同时回写质检状态、批次与效期（最近一次收货口径）。
// 0 行命中返回 CONFLICT。
func (r *PurchaseRepo) ApplyReceivedQtyTx(ctx context.Context, tx *ent.Tx, lineID uint32, delta int64, qc purchaseline.QcStatus, lotName *string, expiry *time.Time) error {
	guard := func(s *sql.Selector) {
		// received + delta ≤ expected
		s.Where(sql.ExprP(
			fmt.Sprintf("%s + %d <= %s", purchaseline.FieldReceivedQty, delta, purchaseline.FieldExpectedQty),
		))
	}

	query := tx.PurchaseLine.Update().
		Where(
			purchaseline.IDEQ(lineID),
			guard,
		).
		AddReceivedQty(delta).
		SetQcStatus(qc).
		SetUpdatedAt(time.Now())
	if lotName != nil {
		query.SetLotName(*lotName)
	}
	if expiry != nil {
		query.SetExpiryDate(*expiry)
	}

	n, err := query.Save(ctx)
	if err != nil {
		r.log.Errorf("apply received qty failed: %s", err.Error())
		return purchaseV1.ErrorInternalServerError("apply received qty failed")
	}
	if n == 0 {
		return purchaseV1.ErrorConflict("receive quantity exceeds expected quantity (over-receipt blocked)")
	}
	return nil
}

// ApplyPutawayQtyTx 原子累加上架数：仅当质检通过且 累计上架 + delta ≤ 已收 时生效。
func (r *PurchaseRepo) ApplyPutawayQtyTx(ctx context.Context, tx *ent.Tx, lineID uint32, delta int64) error {
	guard := func(s *sql.Selector) {
		// putaway + delta ≤ received
		s.Where(sql.ExprP(
			fmt.Sprintf("%s + %d <= %s", purchaseline.FieldPutawayQty, delta, purchaseline.FieldReceivedQty),
		))
	}

	n, err := tx.PurchaseLine.Update().
		Where(
			purchaseline.IDEQ(lineID),
			purchaseline.QcStatusEQ(purchaseline.QcStatusQCPass),
			guard,
		).
		AddPutawayQty(delta).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("apply putaway qty failed: %s", err.Error())
		return purchaseV1.ErrorInternalServerError("apply putaway qty failed")
	}
	if n == 0 {
		return purchaseV1.ErrorConflict("putaway quantity exceeds received quantity or line failed QC")
	}
	return nil
}

// BeginTx 开启事务。
func (r *PurchaseRepo) BeginTx(ctx context.Context) (*ent.Tx, error) {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, purchaseV1.ErrorInternalServerError("start transaction failed")
	}
	return tx, nil
}

// FinishTx 提交/回滚事务（err 非 nil 回滚）。
func (r *PurchaseRepo) FinishTx(tx *ent.Tx, err error) {
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
