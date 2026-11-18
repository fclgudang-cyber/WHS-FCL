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

// WaveItem 波次 SKU 行：订单分析添加进来的 SKU，cek stok / allocate 的对象。
type WaveItem struct {
	ent.Schema
}

func (WaveItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_wave_items",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("波次SKU行表"),
	}
}

// Fields of the WaveItem.
func (WaveItem) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("wave_id").
			Comment("所属波次ID").
			Optional().
			Nillable(),

		field.String("sku_code").
			Comment("SKU 编码").
			Optional().
			Nillable(),

		field.String("product_name").
			Comment("商品名称").
			Optional().
			Nillable(),

		field.Int64("qty_required").
			Comment("需要数量").
			Optional().
			Nillable(),

		field.Int64("qty_allocated").
			Comment("已分配数量").
			Default(0).
			Optional().
			Nillable(),

		// cek stok 结果
		field.String("availability").
			Comment("cek stok 结果：AVAILABLE / PARTIAL / UNAVAILABLE / NOT_FOUND").
			Optional().
			Nillable(),

		field.String("allocated_from").
			Comment("分配 lokasi（人读摘要）").
			Optional().
			Nillable(),

		field.String("allocated_location_id").
			Comment("分配位置ID").
			Optional().
			Nillable(),
	}
}

// Mixin of the WaveItem.
func (WaveItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the WaveItem.
func (WaveItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("wave_id").StorageKey("idx_wms_wave_item_wave"),
		index.Fields("tenant_id", "sku_code").StorageKey("idx_wms_wave_item_tenant_sku"),
	}
}
