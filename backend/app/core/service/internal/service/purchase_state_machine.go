package service

import (
	purchaseV1 "github.com/Mikdad-Force/Ghost/api/gen/go/purchase/service/v1"
)

// Purchase 状态机（纯函数，独立于仓储，便于单测）。
// 流转图：
//
//	DRAFT ──announce──→ ANNOUNCED ──receive──→ RECEIVING ──close──→ COMPLETED
//	  │                     │    ↖ receive ──┘ │
//	  │                     └───cancel────────┤（→ CANCELLED）
//	  ├──cancel──→ CANCELLED                  └───close──→ COMPLETED（允许短收完结）
//
// COMPLETED / CANCELLED 为终态，任何流转都被拒绝。

// purchaseCanTransition 判定状态流转是否合法。
func purchaseCanTransition(from, to purchaseV1.PurchaseStatus) bool {
	switch from {
	case purchaseV1.PurchaseStatus_PURCHASE_DRAFT:
		return to == purchaseV1.PurchaseStatus_PURCHASE_ANNOUNCED ||
			to == purchaseV1.PurchaseStatus_PURCHASE_CANCELLED
	case purchaseV1.PurchaseStatus_PURCHASE_ANNOUNCED:
		return to == purchaseV1.PurchaseStatus_PURCHASE_RECEIVING ||
			to == purchaseV1.PurchaseStatus_PURCHASE_COMPLETED ||
			to == purchaseV1.PurchaseStatus_PURCHASE_CANCELLED
	case purchaseV1.PurchaseStatus_PURCHASE_RECEIVING:
		return to == purchaseV1.PurchaseStatus_PURCHASE_COMPLETED ||
			to == purchaseV1.PurchaseStatus_PURCHASE_CANCELLED
	default:
		// COMPLETED / CANCELLED 为终态
		return false
	}
}

// purchaseActionTarget 状态动作 → 目标状态映射。
func purchaseActionTarget(action string) purchaseV1.PurchaseStatus {
	switch action {
	case "announce":
		return purchaseV1.PurchaseStatus_PURCHASE_ANNOUNCED
	case "cancel":
		return purchaseV1.PurchaseStatus_PURCHASE_CANCELLED
	case "close":
		return purchaseV1.PurchaseStatus_PURCHASE_COMPLETED
	default:
		return purchaseV1.PurchaseStatus_PURCHASE_DRAFT
	}
}

// purchaseReceiveTarget 收货动作允许的来源状态（收货是幂等多次操作，不改状态
// 语义：ANNOUNCED → RECEIVING 仅首收触发；RECEIVING 保持不变）。
func purchaseReceiveAllowed(from purchaseV1.PurchaseStatus) bool {
	return from == purchaseV1.PurchaseStatus_PURCHASE_ANNOUNCED ||
		from == purchaseV1.PurchaseStatus_PURCHASE_RECEIVING
}

// purchaseLineFacts 明细数量口径快照（纯数据，脱离 proto/ent 依赖，便于单测）。
type purchaseLineFacts struct {
	ID          uint32
	ExpectedQty int64
	ReceivedQty int64
	PutawayQty  int64
	QcStatus    purchaseV1.PurchaseQcStatus
}

// validateReceiveQuantities 防超收口径校验（纯函数）。
// 对每行：本次数量必须为正；累计已收 + 本次 ≤ 预期。
// 返回违规的明细 ID，空切片表示全部通过。
func validateReceiveQuantities(lines []purchaseLineFacts, reqLines map[uint32]int64) []uint32 {
	var violated []uint32
	for _, line := range lines {
		delta, ok := reqLines[line.ID]
		if !ok || delta == 0 {
			continue
		}
		if delta < 0 {
			violated = append(violated, line.ID)
			continue
		}
		if line.ReceivedQty+delta > line.ExpectedQty {
			violated = append(violated, line.ID)
		}
	}
	return violated
}

// validatePutawayQuantities 防超上架口径校验（纯函数）。
// 对每行：本次数量必须为正；质检必须通过；累计已上架 + 本次 ≤ 累计已收。
func validatePutawayQuantities(lines []purchaseLineFacts, reqLines map[uint32]int64) []uint32 {
	var violated []uint32
	for _, line := range lines {
		delta, ok := reqLines[line.ID]
		if !ok || delta == 0 {
			continue
		}
		if delta < 0 {
			violated = append(violated, line.ID)
			continue
		}
		if line.QcStatus != purchaseV1.PurchaseQcStatus_QC_PASS {
			violated = append(violated, line.ID)
			continue
		}
		if line.PutawayQty+delta > line.ReceivedQty {
			violated = append(violated, line.ID)
		}
	}
	return violated
}
