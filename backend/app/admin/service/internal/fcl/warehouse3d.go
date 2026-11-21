// warehouse3d.go — Digital Twin Warehouse 3D.
// Semua data operasional tetap dibaca dari tabel WMS/FCL yang sudah ada
// (stock, order, karyawan). File ini hanya menyimpan/menyajikanLAYOUT visual.
package fcl

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed warehouse3d_schema.sql
var h3dSchemaSQL string

//go:embed fcl_bootstrap_schema.sql
var fclBootstrapSQL string

//go:embed fcl_access_permissions.sql
var fclAccessSQL string

// h3dApplySchema menerapkan schema secara idempoten saat service start.
// Bagian permission dibuat best-effort karena tabel RBAC core-service
// mungkin belum termigrasi saat admin-service pertama kali boot.
func h3dApplySchema() {
	marker := "-- 11. Permission"
	parts := strings.SplitN(h3dSchemaSQL, marker, 2)
	if len(parts) > 0 {
		if _, err := db.Exec(parts[0]); err != nil {
			log.Printf("warehouse3d: schema gagal: %v", err)
			return
		}
	}
	if len(parts) == 2 {
		if _, err := db.Exec(marker + parts[1]); err != nil {
			log.Printf("warehouse3d: permission seed dilewati: %v", err)
		}
	}
	if _, err := db.Exec(fclBootstrapSQL); err != nil {
		log.Printf("fcl bootstrap schema gagal: %v", err)
	} else {
		log.Printf("fcl bootstrap schema siap")
	}
	if _, err := db.Exec(fclAccessSQL); err != nil {
		log.Printf("fcl access/menu seed gagal: %v", err)
	} else {
		log.Printf("fcl access/menu seed siap")
	}
	log.Printf("warehouse3d: schema siap")
}

// ---------------------------------------------------------------- helpers

func h3dWrite(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func h3dFail(w http.ResponseWriter, err error) {
	writeError(w, err)
}

func h3dID(r *http.Request, key string) int64 {
	n, _ := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	return n
}

func h3dNum(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case string:
		n, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return n
	case []byte:
		n, _ := strconv.ParseFloat(strings.TrimSpace(string(x)), 64)
		return n
	default:
		return 0
	}
}

func h3dStr(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	// config / path dikirim frontend sebagai objek JSON — jangan fmt.Sprint
	switch v.(type) {
	case map[string]any, []any, map[string]string:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return fmt.Sprint(v)
}

func h3dJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// h3dScalar menjalankan query agregat dan mengembalikan 0 bila tabel belum ada.
func h3dScalar(query string, args ...any) float64 {
	row, err := queryOne(query, args...)
	if err != nil || row == nil {
		return 0
	}
	for _, v := range row {
		return h3dNum(v)
	}
	return 0
}

func h3dRows(query string, args ...any) []map[string]any {
	rows, err := queryAll(query, args...)
	if err != nil {
		return []map[string]any{}
	}
	return rows
}

// h3dMapRows mengubah nama kolom DB (snake_case) menjadi kontrak JSON frontend
// (camelCase), sekaligus mengubah nilai numerik/boolean/JSON ke tipe yang tepat.
func h3dMapRows(rows []map[string]any, mapping map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		mapped := map[string]any{}
		for dbKey, value := range row {
			key := dbKey
			if k, ok := mapping[dbKey]; ok {
				key = k
			}
			switch {
			case h3dJSONKeys[key]:
				if str, ok := h3dStr(value), true; ok && str != "" {
					var parsed any
					if json.Unmarshal([]byte(str), &parsed) == nil {
						mapped[key] = parsed
						continue
					}
				}
				mapped[key] = value
			case h3dBoolKeys[key]:
				mapped[key] = value == true || h3dStr(value) == "true" || h3dStr(value) == "t"
			case h3dNumericKeys[key]:
				// employee_id bisa berupa UUID(text) — jangan dipaksa jadi 0.
				if str := h3dStr(value); str != "" {
					if _, err := strconv.ParseFloat(str, 64); err != nil {
						mapped[key] = value
						continue
					}
				}
				mapped[key] = h3dNum(value)
			default:
				mapped[key] = value
			}
		}
		out = append(out, mapped)
	}
	return out
}

var (
	h3dJSONKeys    = map[string]bool{"config": true, "path": true, "permissions": true}
	h3dBoolKeys    = map[string]bool{"locked": true, "visible": true}
	h3dNumericKeys = map[string]bool{
		"id": true, "warehouseId": true, "layoutId": true, "employeeId": true,
		"lengthM": true, "widthM": true, "heightM": true, "areaM2": true,
		"totalFloor": true, "capacity": true, "wallThickness": true, "version": true,
		"positionX": true, "positionY": true, "positionZ": true,
		"length": true, "width": true, "height": true, "rotation": true,
		"zIndex": true, "levels": true, "bays": true, "bins": true,
		"occupancy": true, "speed": true, "operatorId": true,
	}

	h3dWarehouseMap = map[string]string{
		"length_m": "lengthM", "width_m": "widthM", "height_m": "heightM",
		"area_m2": "areaM2", "total_floor": "totalFloor",
		"building_shape": "buildingShape", "wall_thickness": "wallThickness",
		"created_at": "createdAt", "updated_at": "updatedAt",
	}
	h3dLayoutMap = map[string]string{
		"warehouse_id": "warehouseId", "published_at": "publishedAt",
		"created_at": "createdAt", "updated_at": "updatedAt",
	}
	h3dObjectMap = map[string]string{
		"object_uid": "objectUid", "object_type": "objectType",
		"position_x": "positionX", "position_y": "positionY", "position_z": "positionZ",
		"length_m": "length", "width_m": "width", "height_m": "height",
		"parent_uid": "parentUid", "group_uid": "groupUid", "z_index": "zIndex",
		"area_uid": "areaUid", "rack_uid": "rackUid", "bin_uid": "binUid",
		"location_code": "locationCode", "employee_id": "employeeId",
		"equipment_uid": "equipmentUid", "created_at": "createdAt", "updated_at": "updatedAt",
	}
	h3dAreaMap = map[string]string{
		"area_uid": "areaUid", "area_type": "areaType",
		"position_x": "positionX", "position_y": "positionY", "position_z": "positionZ",
		"length_m": "length", "width_m": "width", "height_m": "height", "area_m2": "areaM2",
		"created_at": "createdAt", "updated_at": "updatedAt",
	}
	h3dRackMap = map[string]string{
		"rack_uid": "rackUid", "rack_type": "rackType",
		"position_x": "positionX", "position_y": "positionY", "position_z": "positionZ",
		"length_m": "length", "width_m": "width", "height_m": "height",
		"created_at": "createdAt", "updated_at": "updatedAt",
	}
	h3dBinMap = map[string]string{
		"rack_uid": "rackUid", "bin_uid": "binUid",
		"position_x": "positionX", "position_y": "positionY", "position_z": "positionZ",
		"location_code": "locationCode", "created_at": "createdAt",
	}
	h3dEquipmentMap = map[string]string{
		"equipment_uid": "equipmentUid", "equipment_type": "equipmentType",
		"operator_id": "operatorId", "task_uid": "taskUid",
		"position_x": "positionX", "position_y": "positionY", "position_z": "positionZ",
		"created_at": "createdAt", "updated_at": "updatedAt",
	}
	h3dEmployeeMap = map[string]string{
		"employee_id": "employeeId", "full_name": "fullName",
		"photo_url": "photoUrl", "warehouse_id": "warehouseId",
		"current_area": "currentArea", "activity_label": "activityLabel",
		"task_code": "taskCode", "current_task": "currentTask",
		"position_x": "positionX", "position_y": "positionY", "position_z": "positionZ",
	}
)

// ---------------------------------------------------------------- default data

// h3dEnsureDefault membuat gudang + layout kosong saat pertama kali dipakai.
// Ini BUKAN layout hardcoded di UI — pengguna bebas mengubah/hapus semua objek.
func h3dEnsureDefault() (int64, int64, error) {
	var wid, lid int64
	row, err := queryOne(`select id from wh3d_warehouse order by id limit 1`)
	if err != nil {
		return 0, 0, err
	}
	if row == nil {
		res, err := db.Exec(`insert into wh3d_warehouse
			(code, name, description, address, length_m, width_m, height_m, total_floor, capacity, status, building_shape, wall_thickness)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			"WH-001", "Gudang Utama", "Digital twin gudang utama", "",
			100, 50, 12, 1, 5000, "ACTIVE", "RECTANGLE", 0.2)
		if err != nil {
			return 0, 0, err
		}
		wid, _ = res.LastInsertId()
		// bigserial via lib/pq tidak selalu mengisi LastInsertId; ambil dari code
		row2, e := queryOne(`select id from wh3d_warehouse where code = $1`, "WH-001")
		if e != nil || row2 == nil {
			return 0, 0, e
		}
		wid = int64(h3dNum(row2["id"]))
	} else {
		wid = int64(h3dNum(row["id"]))
	}

	row, err = queryOne(`select id from wh3d_layout where warehouse_id = $1 order by id limit 1`, wid)
	if err != nil {
		return 0, 0, err
	}
	if row == nil {
		_, err = db.Exec(`insert into wh3d_layout (warehouse_id, code, name, status, version, description)
			values ($1,$2,$3,$4,$5,$6)`, wid, "LAYOUT-001", "Layout Utama", "DRAFT", 1, "Layout digital twin gudang utama")
		if err != nil {
			return 0, 0, err
		}
		row, err = queryOne(`select id from wh3d_layout where warehouse_id = $1 order by id limit 1`, wid)
		if err != nil || row == nil {
			return 0, 0, err
		}
	}
	lid = int64(h3dNum(row["id"]))
	return wid, lid, nil
}

// ---------------------------------------------------------------- building objects

type h3dBuildingPart struct {
	uid, typ, name string
	x, y, z        float64
	l, w, h        float64
	color          string
	config         map[string]any
}

// h3dSeedBuildingObjects membuat komponen bangunan sebagai objek layout yang
// bisa dipilih dan dipindah. Dipanggil sekali per layout (flag building_seeded),
// atau dipaksa ulang lewat endpoint regenerate.
func h3dSeedBuildingObjects(layoutID int64, force bool) {
	row, err := queryOne(`select l.building_seeded, w.length_m, w.width_m, w.height_m, w.wall_thickness
		from wh3d_layout l join wh3d_warehouse w on w.id = l.warehouse_id where l.id = $1`, layoutID)
	if err != nil || row == nil {
		return
	}
	if !force && h3dStr(row["building_seeded"]) == "true" {
		return
	}

	L := h3dNum(row["length_m"])
	W := h3dNum(row["width_m"])
	H := h3dNum(row["height_m"])
	t := h3dNum(row["wall_thickness"])
	if L <= 0 {
		L = 100
	}
	if W <= 0 {
		W = 50
	}
	if H <= 0 {
		H = 12
	}
	if t <= 0 {
		t = 0.2
	}

	doorW := 8.0
	parts := []h3dBuildingPart{
		{"bld-floor", "FLOOR", "Lantai Gudang", 0, 0, 0, L, W, 0.1, "#D9D3C7", map[string]any{}},
		{"bld-wall-back", "WALL", "Dinding Belakang", 0, -W/2 + t/2, 0, L, t, H, "#E8E4DA", map[string]any{"wallThickness": t}},
		{"bld-wall-front-left", "WALL", "Dinding Depan Kiri", -(L/2 + doorW/2), W/2 - t/2, 0, L - doorW, t, H, "#E8E4DA", map[string]any{"wallThickness": t}},
		{"bld-wall-front-right", "WALL", "Dinding Depan Kanan", (L/2 + doorW/2), W/2 - t/2, 0, L - doorW, t, H, "#E8E4DA", map[string]any{"wallThickness": t}},
		{"bld-wall-left", "WALL", "Dinding Kiri", -L/2 + t/2, 0, 0, t, W, H, "#E8E4DA", map[string]any{"wallThickness": t}},
		{"bld-wall-right", "WALL", "Dinding Kanan", L/2 - t/2, 0, 0, t, W, H, "#E8E4DA", map[string]any{"wallThickness": t}},
		{"bld-door", "DOOR", "Pintu Utama", 0, W/2 - t/2, 0, doorW, t, 6, "#3F7D8C", map[string]any{}},
		{"bld-dock", "LOADING_DOCK", "Loading Dock", 0, W/2 - 2.5, 0, 10, 4, 1, "#7A8791", map[string]any{}},
	}

	i := 0
	for x := -L/2 + 12; x < L/2-6; x += 18 {
		i++
		parts = append(parts, h3dBuildingPart{
			uid: fmt.Sprintf("bld-window-%02d", i), typ: "WINDOW",
			name: fmt.Sprintf("Jendela %02d", i), x: x, y: -W/2 + t/2 + 0.1, z: H - 3,
			l: 6, w: 0.2, h: 2.4, color: "#9CC9D4", config: map[string]any{},
		})
	}
	i = 0
	for x := -L/2 + 8; x <= L/2-8; x += 16 {
		for j, z := range []float64{-W/2 + 6, 0, W/2 - 6} {
			i++
			parts = append(parts, h3dBuildingPart{
				uid: fmt.Sprintf("bld-pillar-%02d", i), typ: "PILLAR",
				name: fmt.Sprintf("Tiang %02d-%d", i, j+1), x: x, y: z, z: 0,
				l: 0.7, w: 0.7, h: H, color: "#8C9AA3", config: map[string]any{},
			})
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	for _, p := range parts {
		_, err = tx.Exec(`insert into wh3d_object
			(layout_id, object_uid, object_type, category, name, label,
			 position_x, position_y, position_z, length_m, width_m, height_m, rotation, color,
			 visible, locked, z_index, config)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::jsonb)
			on conflict (layout_id, object_uid) do nothing`,
			layoutID, p.uid, p.typ, "BUILDING", p.name, p.name,
			p.x, p.y, p.z, p.l, p.w, p.h, 0, p.color,
			true, false, 0, h3dJSON(p.config))
		if err != nil {
			log.Printf("warehouse3d: seed building %s gagal: %v", p.uid, err)
			return
		}
	}
	if _, err = tx.Exec(`update wh3d_layout set building_seeded = true where id = $1`, layoutID); err != nil {
		return
	}
	_ = tx.Commit()
}

func h3dHandleBuildingRegenerate(w http.ResponseWriter, r *http.Request) {
	lid := h3dID(r, "layoutId")
	if lid == 0 {
		_, lid, _ = h3dEnsureDefault()
	}
	if _, err := db.Exec(`delete from wh3d_object where layout_id = $1 and category = 'BUILDING'`, lid); err != nil {
		h3dFail(w, err)
		return
	}
	if _, err := db.Exec(`update wh3d_layout set building_seeded = false where id = $1`, lid); err != nil {
		h3dFail(w, err)
		return
	}
	h3dSeedBuildingObjects(lid, true)
	h3dWrite(w, 200, map[string]any{"ok": true, "layoutId": lid})
}

// ---------------------------------------------------------------- bootstrap

func h3dHandleBootstrap(w http.ResponseWriter, r *http.Request) {
	wid, lid, err := h3dEnsureDefault()
	if err != nil {
		h3dFail(w, err)
		return
	}
	if q := h3dID(r, "warehouseId"); q > 0 {
		wid = q
	}
	if q := h3dID(r, "layoutId"); q > 0 {
		lid = q
	}
	h3dSeedBuildingObjects(lid, false)

	warehouse := h3dMapRows(h3dRows(`select * from wh3d_warehouse where id = $1`, wid), h3dWarehouseMap)
	layout := h3dMapRows(h3dRows(`select * from wh3d_layout where id = $1`, lid), h3dLayoutMap)
	objects := h3dMapRows(h3dRows(`select * from wh3d_object where layout_id = $1 order by z_index, id`, lid), h3dObjectMap)
	areas := h3dMapRows(h3dRows(`select * from wh3d_area where layout_id = $1 order by id`, lid), h3dAreaMap)
	racks := h3dMapRows(h3dRows(`select * from wh3d_rack where layout_id = $1 order by id`, lid), h3dRackMap)
	bins := h3dMapRows(h3dRows(`select * from wh3d_bin where layout_id = $1 order by id`, lid), h3dBinMap)
	equipment := h3dMapRows(h3dRows(`select * from wh3d_equipment where layout_id = $1 order by id`, lid), h3dEquipmentMap)

	// Pekerja 3D = join ke master karyawan, TANPA tabel karyawan duplikat.
	h3dSeedEmployees(wid)
	employees := h3dMapRows(h3dRows(`select
			k.id as employee_id, k.nama as full_name, k.jabatan, k.divisi, k.cabang,
			k.status as employee_status, coalesce(k."fingerprintId", k.fingerprint_id) as fingerprint_id,
			e.photo_url, e.warehouse_id, e.current_area, e.activity, e.activity_label,
			e.task_code, e.current_task, e.position_x, e.position_y, e.position_z,
			e.rotation, e.animation
		from wh3d_employee e
		join karyawan k on k.id = e.employee_id
		where ($1 = 0 or e.warehouse_id = $1)
		  and lower(coalesce(k.status,'')) <> 'resign'
		order by k.nama`, wid), h3dEmployeeMap)

	activities := h3dRows(`select * from wh3d_activity
		where ($1 = 0 or warehouse_id = $1) order by event_time desc limit 50`, wid)
	alerts := h3dRows(`select * from wh3d_alert
		where ($1 = 0 or warehouse_id = $1) and status = 'OPEN' order by created_at desc limit 50`, wid)
	library := h3dRows(`select * from wh3d_object_library order by category, name`)

	h3dWrite(w, 200, map[string]any{
		"warehouseId": wid,
		"layoutId":    lid,
		"warehouse":   firstOrEmpty(warehouse),
		"layout":      firstOrEmpty(layout),
		"objects":     objects,
		"areas":       areas,
		"racks":       racks,
		"bins":        bins,
		"equipment":   equipment,
		"employees":   employees,
		"activities":  activities,
		"alerts":      alerts,
		"library":     library,
		"kpi":         h3dKPI(wid),
	})
}

// h3dSeedEmployees menghubungkan pekerja 3D ke MASTER KARYAWAN yang sudah ada.
// Tidak ada nama dummy: yang dipakai adalah id + jabatan asli dari tabel karyawan.
func h3dSeedEmployees(warehouseID int64) {
	if h3dScalar(`select count(*) from wh3d_employee`) > 0 {
		return
	}
	people, err := queryAll(`select id, jabatan, divisi from karyawan
		where lower(coalesce(status,'')) <> 'resign' order by id limit 48`)
	if err != nil {
		return
	}
	for i, p := range people {
		activity, animation, area := h3dActivityForRole(h3dStr(p["jabatan"]), h3dStr(p["divisi"]))
		x := -32 + float64(i%8)*9
		y := -16 + float64(i/8)*8
		_, _ = db.Exec(`insert into wh3d_employee
			(employee_id, warehouse_id, position_x, position_y, activity, animation, current_area, activity_label)
			values ($1,$2,$3,$4,$5,$6,$7,$8)
			on conflict (employee_id) do nothing`,
			nullable(h3dStr(p["id"])), warehouseID, x, y, activity, animation, area, activity)
	}
}

func h3dActivityForRole(jabatan, divisi string) (activity, animation, area string) {
	j := strings.ToLower(jabatan + " " + divisi)
	switch {
	case strings.Contains(j, "manager"):
		return "MONITORING", "IDLE", "Office Area"
	case strings.Contains(j, "supervisor"):
		return "MONITORING", "WALKING", "Storage Area"
	case strings.Contains(j, "forklift"):
		return "MOVING", "OPERATING_FORKLIFT", "Storage Area"
	case strings.Contains(j, "receiving"), strings.Contains(j, "penerimaan"), strings.Contains(j, "unloading"):
		return "RECEIVING", "UNLOADING", "Receiving Area"
	case strings.Contains(j, "putaway"), strings.Contains(j, "penempatan"):
		return "PUTAWAY", "CARRYING_BOX", "Putaway Area"
	case strings.Contains(j, "picking"):
		return "PICKING", "PICKING", "Picking Area"
	case strings.Contains(j, "packing"):
		return "PACKING", "PACKING", "Packing Area"
	case strings.Contains(j, "shipping"), strings.Contains(j, "pengiriman"), strings.Contains(j, "loading"):
		return "LOADING", "LOADING", "Loading Area"
	case strings.Contains(j, "gudang"), strings.Contains(j, "warehouse"), strings.Contains(j, "operator"):
		return "WORKING", "WALKING", "Storage Area"
	default:
		return "IDLE", "IDLE", "Storage Area"
	}
}

func firstOrEmpty(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return map[string]any{}
	}
	return rows[0]
}

// ---------------------------------------------------------------- KPI

func h3dKPI(warehouseID int64) map[string]any {
	num := func(v string) string {
		return strconv.FormatFloat(h3dScalar(v), 'f', 0, 64)
	}

	totalStock := num(`select coalesce(sum(case when stok ~ '^-?[0-9]+(\.[0-9]+)?$' then stok::numeric else 0 end),0) from stock`)
	totalSku := num(`select count(distinct sku) from stock where sku is not null and sku <> ''`)
	available := num(`select coalesce(sum(case when stok ~ '^-?[0-9]+(\.[0-9]+)?$' and stok::numeric > 0 then stok::numeric else 0 end),0) from stock`)
	lowStock := num(`select count(*) from (
		select sku, sum(case when stok ~ '^-?[0-9]+(\.[0-9]+)?$' then stok::numeric else 0 end) as total
		from stock group by sku
	) s join (select sku, max(case when stokMin ~ '^-?[0-9]+(\.[0-9]+)?$' then stokMin::numeric else 0 end) as minimum from stock group by sku) m
	  on m.sku = s.sku where s.total <= m.minimum`)
	expiring := num(`select count(*) from stock
		where expDate is not null and expDate <> ''
		  and to_date(expDate, 'YYYY-MM-DD') between current_date and current_date + interval '30 days'`)
	ordersToday := num(`select count(*) from "order" where createdAt::date = current_date`)
	pendingShip := num(`select count(*) from "order" where lower(coalesce(status,'')) not in ('terkirim','selesai','dibatalkan')`)
	activeWorkers := num(`select count(*) from wh3d_employee where activity <> 'IDLE'`)
	occupied := num(`select count(*) from wh3d_bin where status <> 'EMPTY'`)
	emptyBins := num(`select count(*) from wh3d_bin where status = 'EMPTY'`)
	picking := num(`select count(*) from wh3d_task where task_type = 'PICKING' and status = 'IN_PROGRESS'`)
	packing := num(`select count(*) from wh3d_task where task_type = 'PACKING' and status = 'IN_PROGRESS'`)
	receiving := num(`select count(*) from wh3d_task where task_type = 'RECEIVING' and status = 'PENDING'`)

	return map[string]any{
		"totalStock": totalStock, "totalSku": totalSku, "availableStock": available,
		"lowStock": lowStock, "expiring": expiring, "orderToday": ordersToday,
		"pendingShipping": pendingShip, "activeWorkers": activeWorkers,
		"occupiedLocations": occupied, "emptyLocations": emptyBins,
		"activePicking": picking, "activePacking": packing,
		"pendingReceiving": receiving,
		"reservedStock":    "0", "totalInventory": totalStock, "onHand": totalSku,
	}
}

func h3dHandleKPI(w http.ResponseWriter, r *http.Request) {
	wid := h3dID(r, "warehouseId")
	if wid == 0 {
		wid, _, _ = h3dEnsureDefault()
	}
	h3dWrite(w, 200, h3dKPI(wid))
}

// ---------------------------------------------------------------- save layout

type h3dSavePayload struct {
	LayoutID  int64            `json:"layoutId"`
	Draft     bool             `json:"draft"`
	Note      string           `json:"note"`
	Objects   []map[string]any `json:"objects"`
	Areas     []map[string]any `json:"areas"`
	Racks     []map[string]any `json:"racks"`
	Bins      []map[string]any `json:"bins"`
	Equipment []map[string]any `json:"equipment"`
	Employees []map[string]any `json:"employees"`
}

func h3dHandleSave(w http.ResponseWriter, r *http.Request) {
	var p h3dSavePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		h3dWrite(w, 400, map[string]any{"message": "JSON tidak valid", "detail": err.Error()})
		return
	}
	if p.LayoutID == 0 {
		_, p.LayoutID, _ = h3dEnsureDefault()
	}
	tx, err := db.Begin()
	if err != nil {
		h3dFail(w, err)
		return
	}
	defer tx.Rollback()

	tables := []string{"wh3d_object", "wh3d_area", "wh3d_rack", "wh3d_bin", "wh3d_equipment"}
	for _, t := range tables {
		if _, err = tx.Exec(fmt.Sprintf("delete from %s where layout_id = $1", t), p.LayoutID); err != nil {
			h3dFail(w, err)
			return
		}
	}

	for _, o := range p.Objects {
		cfg := h3dStr(o["config"])
		if cfg == "" {
			cfg = "{}"
		}
		_, err = tx.Exec(`insert into wh3d_object
			(layout_id, object_uid, object_type, category, name, label,
			 position_x, position_y, position_z, length_m, width_m, height_m, rotation, color,
			 parent_uid, group_uid, locked, visible, z_index, area_uid, rack_uid, bin_uid,
			 location_code, sku, employee_id, equipment_uid, config)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27::jsonb)`,
			p.LayoutID, h3dStr(o["objectUid"]), h3dStr(o["objectType"]), h3dStr(o["category"]),
			h3dStr(o["name"]), h3dStr(o["label"]),
			h3dNum(o["positionX"]), h3dNum(o["positionY"]), h3dNum(o["positionZ"]),
			h3dNum(o["length"]), h3dNum(o["width"]), h3dNum(o["height"]), h3dNum(o["rotation"]), h3dStr(o["color"]),
			nullable(h3dStr(o["parentUid"])), nullable(h3dStr(o["groupUid"])),
			o["locked"] == true, o["visible"] != false, h3dNum(o["zIndex"]),
			nullable(h3dStr(o["areaUid"])), nullable(h3dStr(o["rackUid"])), nullable(h3dStr(o["binUid"])),
			nullable(h3dStr(o["locationCode"])), nullable(h3dStr(o["sku"])),
			nullable(h3dStr(o["employeeId"])), nullable(h3dStr(o["equipmentUid"])), cfg)
		if err != nil {
			h3dFail(w, err)
			return
		}
	}

	for _, a := range p.Areas {
		cfg := h3dStr(a["config"])
		if cfg == "" {
			cfg = "{}"
		}
		_, err = tx.Exec(`insert into wh3d_area
			(layout_id, area_uid, code, name, area_type, shape, length_m, width_m, height_m,
			 position_x, position_y, position_z, rotation, color, capacity, description, config)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb)`,
			p.LayoutID, h3dStr(a["areaUid"]), h3dStr(a["code"]), h3dStr(a["name"]),
			h3dStr(a["areaType"]), h3dStr(a["shape"]), h3dNum(a["length"]), h3dNum(a["width"]),
			h3dNum(a["height"]), h3dNum(a["positionX"]), h3dNum(a["positionY"]), h3dNum(a["positionZ"]),
			h3dNum(a["rotation"]), h3dStr(a["color"]), h3dNum(a["capacity"]), nullable(h3dStr(a["description"])), cfg)
		if err != nil {
			h3dFail(w, err)
			return
		}
	}

	for _, rk := range p.Racks {
		cfg := h3dStr(rk["config"])
		if cfg == "" {
			cfg = "{}"
		}
		_, err = tx.Exec(`insert into wh3d_rack
			(layout_id, rack_uid, code, name, rack_type, pattern, length_m, width_m, height_m,
			 levels, bays, bins, capacity, position_x, position_y, position_z, rotation, color, config)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19::jsonb)`,
			p.LayoutID, h3dStr(rk["rackUid"]), h3dStr(rk["code"]), h3dStr(rk["name"]),
			h3dStr(rk["rackType"]), h3dStr(rk["pattern"]), h3dNum(rk["length"]), h3dNum(rk["width"]),
			h3dNum(rk["height"]), h3dNum(rk["levels"]), h3dNum(rk["bays"]), h3dNum(rk["bins"]),
			h3dNum(rk["capacity"]), h3dNum(rk["positionX"]), h3dNum(rk["positionY"]),
			h3dNum(rk["positionZ"]), h3dNum(rk["rotation"]), h3dStr(rk["color"]), cfg)
		if err != nil {
			h3dFail(w, err)
			return
		}
	}

	for _, b := range p.Bins {
		cfg := h3dStr(b["config"])
		if cfg == "" {
			cfg = "{}"
		}
		_, err = tx.Exec(`insert into wh3d_bin
			(layout_id, rack_uid, bin_uid, code, level, bay, position_x, position_y, position_z,
			 location_code, capacity, occupancy, status, config)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb)`,
			p.LayoutID, h3dStr(b["rackUid"]), h3dStr(b["binUid"]), h3dStr(b["code"]),
			h3dNum(b["level"]), h3dNum(b["bay"]), h3dNum(b["positionX"]), h3dNum(b["positionY"]),
			h3dNum(b["positionZ"]), nullable(h3dStr(b["locationCode"])), h3dNum(b["capacity"]),
			h3dNum(b["occupancy"]), h3dStr(b["status"]), cfg)
		if err != nil {
			h3dFail(w, err)
			return
		}
	}

	for _, e := range p.Equipment {
		cfg := h3dStr(e["config"])
		if cfg == "" {
			cfg = "{}"
		}
		path := h3dStr(e["path"])
		if path == "" {
			path = "[]"
		}
		_, err = tx.Exec(`insert into wh3d_equipment
			(layout_id, equipment_uid, code, name, equipment_type, status, operator_id, task_uid,
			 position_x, position_y, position_z, rotation, speed, path, color, config)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15,$16::jsonb)`,
			p.LayoutID, h3dStr(e["equipmentUid"]), h3dStr(e["code"]), h3dStr(e["name"]),
			h3dStr(e["equipmentType"]), h3dStr(e["status"]), nullable(h3dStr(e["operatorId"])),
			nullable(h3dStr(e["taskUid"])), h3dNum(e["positionX"]), h3dNum(e["positionY"]),
			h3dNum(e["positionZ"]), h3dNum(e["rotation"]), h3dNum(e["speed"]), path,
			h3dStr(e["color"]), cfg)
		if err != nil {
			h3dFail(w, err)
			return
		}
	}

	for _, e := range p.Employees {
		_, err = tx.Exec(`update wh3d_employee set position_x=$2, position_y=$3, position_z=$4,
			rotation=$5, animation=$6, activity=$7, current_area=$8, current_task=$9, updated_at=now()
			where employee_id=$1`,
			nullable(h3dStr(e["employeeId"])), h3dNum(e["positionX"]), h3dNum(e["positionY"]),
			h3dNum(e["positionZ"]), h3dNum(e["rotation"]), h3dStr(e["animation"]),
			h3dStr(e["activity"]), nullable(h3dStr(e["currentArea"])), nullable(h3dStr(e["currentTask"])))
		if err != nil {
			h3dFail(w, err)
			return
		}
	}

	if !p.Draft {
		if _, err = tx.Exec(`update wh3d_layout set status='PUBLISHED', published_at=now(), updated_at=now() where id=$1`, p.LayoutID); err != nil {
			h3dFail(w, err)
			return
		}
	} else {
		if _, err = tx.Exec(`update wh3d_layout set status='DRAFT', updated_at=now() where id=$1`, p.LayoutID); err != nil {
			h3dFail(w, err)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		h3dFail(w, err)
		return
	}
	h3dWrite(w, 200, map[string]any{"ok": true, "layoutId": p.LayoutID, "objects": len(p.Objects)})
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullableInt(v any) any {
	n := h3dNum(v)
	if n <= 0 {
		return nil
	}
	return int64(n)
}

// ---------------------------------------------------------------- validation

type h3dIssue struct {
	Severity  string  `json:"severity"`
	Code      string  `json:"code"`
	Message   string  `json:"message"`
	ObjectUID string  `json:"objectUid"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
}

func h3dHandleValidate(w http.ResponseWriter, r *http.Request) {
	lid := h3dID(r, "layoutId")
	wid := h3dID(r, "warehouseId")
	if lid == 0 || wid == 0 {
		wid, lid, _ = h3dEnsureDefault()
	}
	wh := firstOrEmpty(h3dRows(`select length_m, width_m, height_m from wh3d_warehouse where id=$1`, wid))
	L := h3dNum(wh["length_m"])
	W := h3dNum(wh["width_m"])
	if L <= 0 || W <= 0 {
		L, W = 100, 50
	}
	objects := h3dRows(`select object_uid,object_type,category,position_x,position_y,length_m,width_m,rotation from wh3d_object where layout_id=$1`, lid)
	issues := []h3dIssue{}

	// 1. objek keluar bangunan
	for _, o := range objects {
		x, y := h3dNum(o["position_x"]), h3dNum(o["position_y"])
		halfL, halfW := h3dNum(o["length_m"])/2, h3dNum(o["width_m"])/2
		if x-halfL < -L/2 || x+halfL > L/2 || y-halfW < -W/2 || y+halfW > W/2 {
			issues = append(issues, h3dIssue{"ERROR", "OUT_OF_BOUNDS",
				fmt.Sprintf("%s berada di luar bangunan", h3dStr(o["object_type"])),
				h3dStr(o["object_uid"]), x, y})
		}
	}

	// 2. tabrakan antar rak / peralatan / area penyimpanan (AABB)
	type box struct {
		uid, kind  string
		x, y, l, w float64
	}
	boxes := []box{}
	for _, o := range objects {
		kind := strings.ToUpper(h3dStr(o["category"]))
		if kind == "BUILDING" {
			continue
		}
		boxes = append(boxes, box{h3dStr(o["object_uid"]), h3dStr(o["object_type"]),
			h3dNum(o["position_x"]), h3dNum(o["position_y"]), h3dNum(o["length_m"]), h3dNum(o["width_m"])})
	}
	for i := 0; i < len(boxes); i++ {
		for j := i + 1; j < len(boxes); j++ {
			a, b := boxes[i], boxes[j]
			if a.uid == b.uid {
				continue
			}
			overlapX := mathAbs(a.x-b.x) < (a.l+b.l)/2-0.05
			overlapY := mathAbs(a.y-b.y) < (a.w+b.w)/2-0.05
			if overlapX && overlapY {
				issues = append(issues, h3dIssue{"WARNING", "OVERLAP",
					fmt.Sprintf("%s bertabrakan dengan %s", a.kind, b.kind), a.uid, a.x, a.y})
			}
		}
	}

	// 3. pintu terhalang
	doors := h3dRows(`select object_uid,position_x,position_y from wh3d_object where layout_id=$1 and lower(object_type) like '%door%'`, lid)
	for _, d := range doors {
		dx, dy := h3dNum(d["position_x"]), h3dNum(d["position_y"])
		for _, b := range boxes {
			if mathAbs(b.x-dx) < b.l/2+1.2 && mathAbs(b.y-dy) < b.w/2+1.2 {
				issues = append(issues, h3dIssue{"ERROR", "DOOR_BLOCKED",
					"Akses pintu terhalang objek " + b.kind, h3dStr(d["object_uid"]), dx, dy})
			}
		}
	}

	// 4. jalur forklift melewati rak
	eqs := h3dRows(`select equipment_uid,path from wh3d_equipment where layout_id=$1`, lid)
	for _, e := range eqs {
		var path []map[string]any
		_ = json.Unmarshal([]byte(h3dStr(e["path"])), &path)
		for _, wp := range path {
			px, py := h3dNum(wp["x"]), h3dNum(wp["y"])
			for _, b := range boxes {
				if strings.Contains(strings.ToLower(b.kind), "rack") &&
					mathAbs(b.x-px) < b.l/2+0.4 && mathAbs(b.y-py) < b.w/2+0.4 {
					issues = append(issues, h3dIssue{"ERROR", "PATH_BLOCKED",
						"Jalur forklift melewati rak " + b.uid, h3dStr(e["equipment_uid"]), px, py})
				}
			}
		}
	}

	// 5. bin tanpa lokasi WMS
	missing := h3dScalar(`select count(*) from wh3d_bin where layout_id=$1 and (location_code is null or location_code='')`, lid)
	if missing > 0 {
		issues = append(issues, h3dIssue{"WARNING", "BIN_NO_LOCATION",
			fmt.Sprintf("%d bin belum terhubung ke lokasi WMS", int64(missing)), "", 0, 0})
	}

	h3dWrite(w, 200, map[string]any{
		"valid":  len(issues) == 0,
		"issues": issues,
		"count":  len(issues),
	})
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// ---------------------------------------------------------------- publish & version

func h3dHandlePublish(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LayoutID int64  `json:"layoutId"`
		Note     string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.LayoutID == 0 {
		_, body.LayoutID, _ = h3dEnsureDefault()
	}
	// Validasi penuh tersedia lewat /validate; publish tidak memblokir transaksi WMS.
	var err error
	var row map[string]any

	payload := map[string]any{
		"objects":   h3dRows(`select * from wh3d_object where layout_id=$1`, body.LayoutID),
		"areas":     h3dRows(`select * from wh3d_area where layout_id=$1`, body.LayoutID),
		"racks":     h3dRows(`select * from wh3d_rack where layout_id=$1`, body.LayoutID),
		"bins":      h3dRows(`select * from wh3d_bin where layout_id=$1`, body.LayoutID),
		"equipment": h3dRows(`select * from wh3d_equipment where layout_id=$1`, body.LayoutID),
	}
	var version int
	row, err = queryOne(`select version from wh3d_layout where id=$1`, body.LayoutID)
	if err == nil && row != nil {
		version = int(h3dNum(row["version"]))
	}
	version++
	tx, err := db.Begin()
	if err != nil {
		h3dFail(w, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`insert into wh3d_layout_version (layout_id,version,payload,note,created_by)
		values ($1,$2,$3::jsonb,$4,$5)`, body.LayoutID, version, h3dJSON(payload), nullable(body.Note), h3dUser(r)); err != nil {
		h3dFail(w, err)
		return
	}
	if _, err = tx.Exec(`update wh3d_layout set status='PUBLISHED', version=$2, published_at=now(), updated_at=now() where id=$1`, body.LayoutID, version); err != nil {
		h3dFail(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		h3dFail(w, err)
		return
	}
	h3dWrite(w, 200, map[string]any{"ok": true, "version": version})
}

func h3dUser(r *http.Request) any {
	if u, ok := r.Context().Value("fcl_user").(map[string]any); ok {
		return h3dStr(u["username"])
	}
	return nil
}

func h3dHandleVersions(w http.ResponseWriter, r *http.Request) {
	lid := h3dID(r, "layoutId")
	if lid == 0 {
		_, lid, _ = h3dEnsureDefault()
	}
	h3dWrite(w, 200, h3dRows(`select id,version,note,created_at,created_by from wh3d_layout_version where layout_id=$1 order by version desc`, lid))
}

func h3dHandleRestore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LayoutID int64 `json:"layoutId"`
		Version  int   `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	row, err := queryOne(`select payload from wh3d_layout_version where layout_id=$1 and version=$2`, body.LayoutID, body.Version)
	if err != nil || row == nil {
		h3dWrite(w, 404, map[string]any{"message": "Versi tidak ditemukan"})
		return
	}
	var p struct {
		Objects   []map[string]any `json:"objects"`
		Areas     []map[string]any `json:"areas"`
		Racks     []map[string]any `json:"racks"`
		Bins      []map[string]any `json:"bins"`
		Equipment []map[string]any `json:"equipment"`
	}
	_ = json.Unmarshal([]byte(h3dStr(row["payload"])), &p)
	tx, err := db.Begin()
	if err != nil {
		h3dFail(w, err)
		return
	}
	defer tx.Rollback()
	for _, t := range []string{"wh3d_object", "wh3d_area", "wh3d_rack", "wh3d_bin", "wh3d_equipment"} {
		if _, err = tx.Exec(fmt.Sprintf("delete from %s where layout_id=$1", t), body.LayoutID); err != nil {
			h3dFail(w, err)
			return
		}
	}
	reinsert := func(table string, rows []map[string]any, cols string, argsFn func(map[string]any) []any) error {
		for _, row := range rows {
			vals := argsFn(row)
			ph := make([]string, len(vals))
			for i := range vals {
				ph[i] = fmt.Sprintf("$%d", i+1)
			}
			if _, e := tx.Exec(fmt.Sprintf("insert into %s (%s) values (%s)", table, cols, strings.Join(ph, ",")), vals...); e != nil {
				return e
			}
		}
		return nil
	}
	// Restore memakai payload hasil publish (kolom DB asli) — disalin langsung.
	insertRaw := func(table string, rows []map[string]any) error {
		for _, row := range rows {
			cols, vals := []string{}, []any{}
			for k, v := range row {
				// id dan kolom generated tidak boleh disisipkan saat restore
				if k == "id" || k == "area_m2" {
					continue
				}
				cols = append(cols, quoteIdent(k))
				vals = append(vals, v)
			}
			ph := make([]string, len(vals))
			for i := range vals {
				ph[i] = fmt.Sprintf("$%d", i+1)
			}
			if _, e := tx.Exec(fmt.Sprintf("insert into %s (%s) values (%s)", table, strings.Join(cols, ","), strings.Join(ph, ",")), vals...); e != nil {
				return e
			}
		}
		return nil
	}
	if err = insertRaw("wh3d_object", p.Objects); err != nil {
		h3dFail(w, err)
		return
	}
	if err = insertRaw("wh3d_area", p.Areas); err != nil {
		h3dFail(w, err)
		return
	}
	if err = insertRaw("wh3d_rack", p.Racks); err != nil {
		h3dFail(w, err)
		return
	}
	if err = insertRaw("wh3d_bin", p.Bins); err != nil {
		h3dFail(w, err)
		return
	}
	if err = insertRaw("wh3d_equipment", p.Equipment); err != nil {
		h3dFail(w, err)
		return
	}
	_ = reinsert // disimpan untuk kompatibilitas payload lama
	if _, err = tx.Exec(`update wh3d_layout set status='DRAFT', updated_at=now() where id=$1`, body.LayoutID); err != nil {
		h3dFail(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		h3dFail(w, err)
		return
	}
	h3dWrite(w, 200, map[string]any{"ok": true, "restoredVersion": body.Version})
}

// ---------------------------------------------------------------- search

func h3dHandleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		h3dWrite(w, 200, []any{})
		return
	}
	like := "%" + strings.ToLower(q) + "%"
	results := []map[string]any{}
	for _, row := range h3dRows(`select sku as code, nama as name, 'SKU' as type, lokasi as location, barcode as ref
		from stock where lower(coalesce(sku,'')) like $1 or lower(coalesce(nama,'')) like $1
		  or lower(coalesce(barcode,'')) like $1 or lower(coalesce(lokasi,'')) like $1 limit 20`, like) {
		results = append(results, row)
	}
	for _, row := range h3dRows(`select bin_uid as code, code as name, 'BIN' as type, location_code as location, rack_uid as ref
		from wh3d_bin where lower(coalesce(bin_uid,'')) like $1 or lower(coalesce(code,'')) like $1
		  or lower(coalesce(location_code,'')) like $1 limit 20`, like) {
		results = append(results, row)
	}
	for _, row := range h3dRows(`select k.id as code, k.nama as name, 'EMPLOYEE' as type, k.jabatan as location,
			coalesce(k."fingerprintId", k.fingerprint_id) as ref
		from karyawan k where lower(coalesce(k.nama,'')) like $1 or lower(coalesce(k.jabatan,'')) like $1
		  or cast(k.id as text) like $1
		  or lower(coalesce(k."fingerprintId", k.fingerprint_id, '')) like $1 limit 20`, like) {
		results = append(results, row)
	}
	for _, row := range h3dRows(`select object_uid as code, name, 'OBJECT' as type, object_type as location, object_uid as ref
		from wh3d_object where lower(coalesce(name,'')) like $1 or lower(coalesce(object_uid,'')) like $1 limit 20`, like) {
		results = append(results, row)
	}
	for _, row := range h3dRows(`select coalesce("noResi", noresi) as code, keterangan as name, 'ORDER' as type, status as location, cast(id as text) as ref
		from "order" where lower(coalesce("noResi", noresi, '')) like $1 or lower(coalesce(keterangan,'')) like $1 limit 20`, like) {
		results = append(results, row)
	}
	h3dWrite(w, 200, results)
}

// ---------------------------------------------------------------- live SSE

func h3dHandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		h3dWrite(w, 500, map[string]any{"message": "streaming tidak didukung"})
		return
	}
	wid := h3dID(r, "warehouseId")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	last := time.Time{}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			activities := h3dRows(`select * from wh3d_activity where ($1=0 or warehouse_id=$1) and event_time > $2 order by event_time desc limit 20`, wid, last)
			alerts := h3dRows(`select * from wh3d_alert where ($1=0 or warehouse_id=$1) and status='OPEN' order by created_at desc limit 20`, wid)
			payload := map[string]any{"activities": activities, "alerts": alerts, "kpi": h3dKPI(wid), "time": time.Now().UTC()}
			b, _ := json.Marshal(payload)
			fmt.Fprintf(w, "event: tick\ndata: %s\n\n", b)
			flusher.Flush()
			last = time.Now().Add(-2 * time.Second)
		}
	}
}

// ---------------------------------------------------------------- router

func handleWarehouse3D(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/warehouse3d")
	switch {
	case p == "/bootstrap":
		h3dHandleBootstrap(w, r)
	case p == "/kpi":
		h3dHandleKPI(w, r)
	case p == "/save" && r.Method == http.MethodPost:
		h3dHandleSave(w, r)
	case p == "/publish" && r.Method == http.MethodPost:
		h3dHandlePublish(w, r)
	case p == "/versions":
		h3dHandleVersions(w, r)
	case p == "/restore" && r.Method == http.MethodPost:
		h3dHandleRestore(w, r)
	case p == "/building/regenerate" && r.Method == http.MethodPost:
		h3dHandleBuildingRegenerate(w, r)
	case p == "/validate":
		h3dHandleValidate(w, r)
	case p == "/search":
		h3dHandleSearch(w, r)
	case p == "/activity/stream":
		h3dHandleStream(w, r)
	default:
		h3dWrite(w, 404, map[string]any{"message": "endpoint warehouse3d tidak ditemukan"})
	}
}
