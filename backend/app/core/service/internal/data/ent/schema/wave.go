package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
	appmixin "github.com/Mikdad-Force/Ghost/pkg/entgo/mixin"
)

// Wave 波次：订单分析 SKU 进入波次 → cek stok → lepaskan → 拣货任务。
// 状态机：DRAFT → CHECKED → READY_TO_RELEASE → RELEASED → COMPLETED。
// DRAFT / CHECKED / READY_TO_RELEASE 可 CANCELLED。
type Wave struct {
	ent.Schema
}

func (Wave) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_waves",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("波次表"),
	}
}

// Fields of the Wave.
func (Wave) Fields() []ent.Field {
	return []ent.Field{
		field.String("wave_number").
			Comment("波次号（服务端生成，租户范围内唯一）").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态：DRAFT=草稿，CHECKED=已检查，READY_TO_RELEASE=Siap Dilepas，RELEASED=已 Lepaskan，COMPLETED=已完成，CANCELLED=已取消").
			NamedValues(
				"Draft", "DRAFT",
				"Running", "RUNNING",
				"Completed", "COMPLETED",
				"Cancelled", "CANCELLED",
				"Checked", "CHECKED",
				"ReadyToRelease", "READY_TO_RELEASE",
				"Released", "RELEASED",
			).
			Default("DRAFT").
			Optional().
			Nillable(),

		field.Uint32("cart_id").
			Comment("分配的 keranjang（nullable）").
			Optional().
			Nillable(),
	}
}

// Mixin of the Wave.
func (Wave) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Wave.
func (Wave) Indexes() []ent.Index {
	return []ent.Index{
		// 租户范围内波次号唯一
		index.Fields("tenant_id", "wave_number").
			Unique().
			StorageKey("idx_wms_wave_tenant_number"),
	}
}
