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

// SortSession sesi sortir: mengikat keranjang picking (scan selesai) ke keranjang sortir.
type SortSession struct {
	ent.Schema
}

func (SortSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_sort_sessions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("Sortir session 表"),
	}
}

func (SortSession) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("picking_basket_id").
			Comment("Keranjang picking (wms_picking_baskets.id)，nullable bila alur scan lama dipakai").
			Optional().
			Nillable(),

		field.Uint32("picking_task_id").
			Comment("Tugas picking (wms_picking_tasks.id)").
			Optional().
			Nillable(),

		field.String("picking_cart_number").
			Comment("Kode keranjang picking (scanner alur lama)").
			Optional().
			Nillable(),

		field.Uint32("sort_basket_id").
			Comment("FK wms_sort_baskets.id").
			Optional().
			Nillable(),

		field.Uint32("wave_id").
			Comment("FK waves.id").
			Optional().
			Nillable(),

		field.Uint32("user_id").
			Comment("Operator").
			Optional().
			Nillable(),

		field.String("device").
			Comment("").
			Optional().
			Nillable(),

		field.Time("started_at").
			Comment("").
			Optional().
			Nillable(),

		field.Time("finished_at").
			Comment("").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("RUNNING/COMPLETED/CANCELLED").
			NamedValues(
				"Running", "RUNNING",
				"Completed", "COMPLETED",
				"Cancelled", "CANCELLED",
			).
			Default("RUNNING").
			Optional().
			Nillable(),
	}
}

func (SortSession) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.TenantID[uint32]{},
	}
}

func (SortSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "picking_task_id").
			StorageKey("idx_wms_sort_session_task"),
		index.Fields("tenant_id", "sort_basket_id").
			StorageKey("idx_wms_sort_session_sort_basket"),
	}
}
