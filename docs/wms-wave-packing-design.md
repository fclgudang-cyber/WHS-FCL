# WMS: Wave & Picking Setting + Sortir Multi + Packing (Single & Multi)

> Status: **Spec (desain)** — implementasi bertahap, lihat §10 Roadmap.
> Dibuat sebagai lanjutan dari flow Order → Wave → Picking yang sudah live di local (`:11666`).

---

## 1. Ringkasan

Flow baru membedakan pesanan **SINGLE** (1 pesanan = 1 SKU) dan **MULTI** (1 pesanan = >1 SKU).

```
Analisis Pesanan
     │
     ▼
Wave (Single / Multi)        [BAGIAN 1]
     │  cek stok + reserve
     ▼
Picking Keranjang
     │
     ▼
Multi ──► Sortir Keranjang   [BAGIAN 3]
     │
Single ─┐   ┌────────────────┘
        ▼   ▼
      Packing (Single / Multi) [BAGIAN 4 / 5]
        │
        ▼
     Pesanan Selesai (Dipacking)
```

---

## 2. Database

Semua tabel ber-`tenant_id` dan memakai pola uuid/auto-increment seperti tabel lain.

### 2.1 `wave_settings` — pengaturan wave per tipe

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | bigint PK | |
| tenant_id | bigint | FK `sys_tenants` |
| wave_type | enum('SINGLE','MULTI') | konteks aturan |
| max_orders_per_basket | int | default `SINGLE=20`, `MULTI=10`; boleh di-override |
| max_sku_per_wave | int nullable | limit SKU per wave; NULL = tidak dibatasi |
| grouping_rule | enum('PER_WAREHOUSE','PER_ZONE','PER_COURIER','NONE') | aturan pengelompokan wave |
| auto_split | bool | wave dipecah otomatis kalau melebihi kapasitas |
| default_cart_capacity | int | kapasitas default keranjang (override-able) |
| created_at/updated_at/deleted_at/created_by/updated_by | timestamps | |

UNIQUE (tenant_id, wave_type)

### 2.2 `picking_baskets` — master keranjang picking

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | bigint PK | |
| tenant_id | bigint FK | |
| code | varchar(32) | kode unik keranjang (mis. `PS01`, `PM01`) |
| basket_type | enum('SINGLE','MULTI') | **tidak boleh campur** dengan keranjang lain |
| capacity | int | jumlah order maks; override dari `wave_settings.default_cart_capacity` |
| status | enum('FREE','IN_USE','WAITING_SORT','WAITING_PACK') | `Kosong/Dipakai/Menunggu Sortir/Menunggu Packing` |
| current_wave_id | bigint nullable FK | wave yang sedang dipakai |
| current_task_id | bigint nullable FK | tugas picking yang sedang berjalan |
| created_at/updated_at/... | timestamps | |

UNIQUE (tenant_id, code)

### 2.3 `sort_baskets` — keranjang sortir (khusus MULTI)

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | bigint PK | |
| tenant_id | bigint FK | |
| prefix | varchar(8) | mis. `AB` |
| number | int | nomor 01-99 (otomatis naik) |
| code | varchar(16) UNIQUE | `AB01` … `AB99`, lalu `AB100` |
| box_count | int | jumlah kotak (default dari parameter generate, mis. `20`) |
| status | enum('FREE','IN_USE','FULL') | `Kosong / Dipakai / Lengkap` |
| created_by/…/updated_at | timestamps | |

UNIQUE (tenant_id, prefix, number); UNIQUE (tenant_id, code)

Digit kotak: `number < 100` → 2 digit; else 3+ digit, otomatis.

### 2.4 `sort_slots` — kotak dalam keranjang sortir

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | bigint PK | |
| sort_basket_id | FK `sort_baskets` | |
| slot_number | int | `1..box_count` |
| order_id | bigint FK `ord_orders` | order yang dialokasikan ke kotak |
| status | enum('EMPTY','FILLED','COMPLETE') | `Kosong/Terisi/Lengkap` |
| qty_scanned | int default 0 | jumlah item masuk |
| created_at/updated_at | timestamps | |

UNIQUE (sort_basket_id, slot_number); UNIQUE (sort_basket_id, order_id)

### 2.5 `sort_slot_items` — SKU yang harus masuk ke kotak

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | bigint PK | |
| sort_slot_id | FK `sort_slots` | |
| sku_code | varchar(64) | |
| product_name | varchar(255) | |
| qty_required | int | target |
| qty_scanned | int default 0 | yang sudah discan |
| status | enum('PENDING','DONE') | default `PENDING` |
| created_at/updated_at | timestamps | |

UNIQUE (sort_slot_id, sku_code)

### 2.6 `sort_sessions` — sesi sortir (1 keranjang picking ↔ 1 keranjang sortir)

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | bigint PK | |
| tenant_id | bigint FK | |
| picking_basket_id | FK `picking_baskets` | keranjang asal (sudah selesai picking) |
| sort_basket_id | FK `sort_baskets` | keranjang tujuan sortir |
| user_id | bigint | operator |
| device | varchar(64) | identifier device |
| started_at | timestamptz | scan keranjang pertama |
| finished_at | timestamptz nullable | semua kotak complete |
| status | enum('RUNNING','COMPLETED') | |

UNIQUE (picking_basket_id, status='RUNNING') — satu sesi aktif per keranjang.

### 2.7 `packing_logs` — riwayat packing
| Kolom | Tipe |
|---|---|
| id | bigint PK |
| tenant_id | bigint FK |
| order_id | FK `ord_orders` |
| packing_type | enum('SINGLE','MULTI') |
| cart_id | FK `picking_baskets` / `sort_slots` (nullable) |
| resi | varchar(64) | no resi tercetak |
| user_id | bigint |
| packed_at | timestamptz |

### 2.8 `scan_logs` — semua scan tercatat
| Kolom | Tipe |
|---|---|
| id | bigint PK |
| tenant_id | bigint FK |
| scan_type | enum('CART','SLOT','SKU','QTY','RESI') |
| code | varchar(64) | kode yang discan |
| context | varchar(64) varchar | mis. `wave:1`, `task:7`, `slot:3` |
| result | enum('OK','FAILED') | |
| reason | varchar(255) nullable | pesan gagal |
| user_id | bigint |
| device | varchar(64) |
| created_at | timestamptz |

---

## 3. Logika Pembagian Order ke Keranjang

**Prinsip:** setiap keranjang = **satu tipe**, tidak boleh campur.

1. Ambil pesanan yang belum masuk wave (`status = ORDER_PENDING / ORDER_CONFIRMED` belum di-pick).
2. Pisahkan `SINGLE` vs `MULTI`:
   - `SINGLE`: order dengan `count(order_lines) == 1`
   - `MULTI`: `count(order_lines) > 1`
3. Terapkan **grouping_rule** (dari `wave_settings`) ke setiap tipe:
   - `PER_WAREHOUSE` → 1 wave per (type, warehouse_code)
   - `PER_ZONE` → 1 wave per (type, zone dari lokasi? perlu field `zone`) — dilengkapi kolom `inventory_location.zone` atau ikut pakai warehouse code sementara
   - `PER_COURIER` → 1 wave per (type, courier) — perlu field `ord_orders.courier`
   - `NONE` → 1 wave per (type, tenant)
4. Untuk tiap grup, buat `wave` + `picking_tasks` otomatis; keranjang dialokasikan dari `picking_baskets` dengan `status=FREE` dan `basket_type` sesuai; jika habis → wave dipecah lagi per sisa.
5. Jika `auto_split = true` dan total order > `max_orders_per_basket`, buat wave otomatis dipisah per `max_orders_per_basket`.
6. Sisa order yang belum kebagian keranjang → tetap di wave berikutnya (keterangan di UI "menunggu keranjang kosong").

**Sortir (MULTI)** — selesai picking, order masih MULTI dan belum disortir:
- Operator scan keranjang picking → pilih keranjang sortir `FREE`, bind 1:1 untuk sesi ini.
- Generate kotak otomatis sesuai jumlah order multi di keranjang tersebut (1 kotak = 1 order).
- Saat scan SKU dari hasil picking: sistem cocokkan item itu ke kotak **belum penuh** yang membutuhkan SKU tersebut (urut prioritas kotak terkecil). Jika SKU dipakai >1 order → urut kotak terkecil dulu (FIFO). Kalau semua butuh SKU ini penuh → error "SKU tidak ada di order manapun di keranjang ini".

**Packing SINGLE**:
- Scan keranjang picking → list SKU/order; scan SKU → sistem ambil order dengan SKU+qty terkecil (atau order pertama di wave); cetak resi; qty di order line dikurangi 1 (untuk qty>1, qty berkurang; order di-packing kalau habis).
- Kalau SKU tidak ada → error, sound, tidak lanjut.

**Packing MULTI**:
- Scan keranjang sortir / kotak → tampilkan list SKU order.
- Scan SKU → verifikasi ada; centang; order → Dipacking kalau lengkap; kotak → Dipacking.
- Kalau kotak belum Lengkap (semua SKU masuk) → tolak, tampilkan SKU yang belum masuk.

---

## 4. Status & Pemetaan Teknis

| UI | Ent enum | Proto/DB |
|---|---|---|
| Wave: DRAFT → CHECKED → READY_TO_RELEASE → RELEASED → COMPLETED | `DRAFT/CHECKED/READY_TO_RELEASE/RELEASED/COMPLETED` | `CHECKED/READY_TO_RELEASE/RELEASED/COMPLETED/CANCELLED` |
| Keranjang Picking: FREE / IN_USE / WAITING_SORT / WAITING_PACK | `FREE/IN_USE/WAITING_SORT/WAITING_PACK` | `FREE/IN_USE` (tambah status) |
| Keranjang Sortir: FREE / IN_USE / FULL | `FREE/IN_USE/FULL` | sama |
| Kotak: EMPTY / FILLED / COMPLETE / PACKED | `EMPTY/FILLED/COMPLETE/PACKED` | sama |
| Tugas Picking: WAITING / RUNNING / DONE | `WAITING/RUNNING/DONE/CANCELLED` | sama |
| Order Multi: PICKED / SORTED / PACKED | `ORDER_PICKING/ORDER_SORTED/ORDER_PACKED` | `ORDER_FULFILLING/ORDER_COMPLETED` dipakai; tambah varian `ORDER_PACKED` |

---

## 5. API Endpoints (per langkah)

Semua di bawah `/admin/v1/` kecuali login.

### 5.1 Pengaturan Wave & Keranjang

| Method | Path | Keterangan |
|---|---|---|
| GET | `/admin/v1/wave-settings?type=SINGLE|MULTI` | ambil pengaturan |
| PUT | `/admin/v1/wave-settings` | update pengaturan (per tenant) |
| GET | `/admin/v1/picking-baskets?basketType=&status=` | list keranjang picking |
| POST | `/admin/v1/picking-baskets` | create (kode unik) |
| PUT | `/admin/v1/picking-baskets/{id}` | update (kapasitas, status) |
| DELETE | `/admin/v1/picking-baskets/{id}` | hapus (hanya FREE) |

### 5.2 Master Keranjang Sortir

| Method | Path | Keterangan |
|---|---|---|
| GET | `/admin/v1/sort-baskets?status=` | list |
| POST | `/admin/v1/sort-baskets/generate` | **generate batch**: `{prefix, count, startFrom=01, boxCount=20}` → kode otomatis `AB01..AB10`, tolak prefix+nomor duplikat |
| PUT | `/admin/v1/sort-baskets/{id}` | update box_count/status (hanya FREE) |
| DELETE | `/admin/v1/sort-baskets/{id}` | hapus (hanya FREE & kosong) |
| GET | `/admin/v1/sort-baskets/{id}/slots` | list kotak |
| POST | `/admin/v1/sort-baskets/{id}/slots` | tambah kotak manual (jarang dipakai) |
| PUT | `/admin/v1/sort-baskets/{id}/slots/{slotId}` | update slot (assign order) |

### 5.3 Sortir (MULTI, setelah picking selesai)

| Method | Path | Keterangan |
|---|---|---|
| POST | `/admin/v1/sort-sessions` | mulai sesi: `{pickingBasketId, sortBasketId}` → bind, tolak kalau sortBasket dipakai |
| GET | `/admin/v1/sort-sessions/{id}` | detail: slots + items |
| POST | `/admin/v1/sort-sessions/{id}/scan-slot` | scan slot (konfirmasi kotak tujuan) |
| POST | `/admin/v1/sort-sessions/{id}/scan-sku` | scan SKU → kembalikan `{slotId, slotNumber, orderNumber, qtyRequired, qtyScanned}` |
| POST | `/admin/v1/sort-sessions/{id}/scan-qty` | tambah qty ke kotak: `{slotId, sku, qty}` |
| POST | `/admin/v1/sort-sessions/{id}/complete` | semua kotak complete → tandai `sort_session.status = COMPLETED`, keranjang picking → FREE |
| POST | `/admin/v1/sort-sessions/{id}/cancel` | batalkan sesi (keranjang picking & sortir dilepas) |

### 5.4 Packing

| Method | Path | Keterangan |
|---|---|---|
| POST | `/admin/v1/packing/scan-basket` | scan keranjang (single/multi) → daftar SKU/order di dalamnya |
| POST | `/admin/v1/packing/scan-sku` | scan SKU → untuk SINGLE langsung cetak resi & order→PACKED; untuk MULTI verifikasi |
| GET | `/admin/v1/packing/resi/{orderId}` | ambil resi (HTML/PDF, data siap cetak) |
| POST | `/admin/v1/packing/orders/{id}/pack` | manual pack (admin) |
| POST | `/admin/v1/packing/orders/{id}/problem` | lapor masalah (alasan) → order ditahan |

### 5.5 Scan umum

| Method | Path | Keterangan |
|---|---|---|
| POST | `/admin/v1/scan-logs` | catat scan (device, result, reason) |
| GET | `/admin/v1/scan-logs?from=&to=&user=` | audit trail |

---

## 6. Layout Halaman

### 6.1 Pengaturan Wave (`/wms/settings/wave`)

```
┌─────────────────────────────────────────────────┐
│ Pengaturan Wave                    [Tab: Single][Tab: Multi] │
├─────────────────────────────────────────────────┤
│ Maks order / keranjang      [ 20 ]              │
│ Maks SKU per wave (opsional)[    ]              │
│ Aturan pengelompokan        [Per Gudang ▾]      │
│ Auto-split wave             [x] saat order > cap │
│ Kapasitas keranjang default [ 20 ]              │
├─────────────────────────────────────────────────┤
│ [Simpan]  (per tab, terpisah Single & Multi)     │
└─────────────────────────────────────────────────┘
```

Sub-halaman `Master Keranjang Picking` (`/wms/settings/picking-baskets`):

```
Code   Tipe   Kapasitas Status   Aksi
PS01   SINGLE 20        FREE     [edit][hapus]
PM01   MULTI  10        IN_USE   [edit]
```

### 6.2 Master Keranjang Sortir (`/wms/settings/sort-baskets`)

```
Prefix [AB]  Jumlah [10]  Mulai dari [01]  Kotak [20]   [Generate]
        ↓
Kode     Kotak Status   Aksi
AB01     20    FREE      [lihat][edit][hapus]
AB02     20    IN_USE    [lihat]
AB10     20    FREE      [lihat][edit][hapus]
```

### 6.3 Sortir MULTI (`/wms/sorting`) — mobile-first

```
Langkah 1/3: Scan Keranjang Picking
[ Kotak input scan di sini ]

Langkah 2/3: Scan Keranjang Sortir
[ Kotak input scan di sini ]

Langkah 3/3: Kotak 1..20
┌────┬───────────────┬──────────┬─────────┐
│ #  │ Order         │ SKU      │ Progress│
├────┼───────────────┼──────────┼─────────┤
│ 1  │ SO-…-0001     │ 2 SKU    │ 0/2     │ ◄─ highlight saat butuh SKU ini
│ 2  │ SO-…-0002     │ 3 SKU    │ 0/3     │
│ ...│ ...           │ ...      │ ...     │
└────┴───────────────┴──────────┴─────────┘
Scan SKU → [SKU…] → sistem highlight kotak tujuan, operator tap konfirmasi.
```

### 6.4 Packing SINGLE (`/wms/packing/single`)

```
Keranjang [PS01]  (scan di sini)

┌────────┬──────────────┬──────┬────────┐
│ SKU    │ Produk       │ Qty  │ Orders │
│ A001   │ Baut M8      │ 100  │ 20     │
│ A002   │ Sekrup       │ 50   │ 10     │
└────────┴──────────────┴──────┴────────┘
Scan SKU → cetak resi → baris hilang.
[Cetak Ulang Resi] [Lapor Masalah]
```

### 6.5 Packing MULTI (`/wms/packing/multi`)

```
Scan Keranjang Sortir / Kotak: [ AB03 ]

Kotak 3 — Order SO-…-0007
┌────┬──────────────┬──────┬───────┐
│ SKU │ Produk      │ Qty  │ Scan  │
│ A001│ Baut M8     │ 10   │  [ ]  │
│ A003│ Sarung Tangan│ 2   │  [x]  │
└────┴──────────────┴──────┴───────┘
Semua tercentang → order Dipacking, kotak dikosongkan.
```

---

## 7. Error & Edge Cases

| Kode | Pesan (id-ID) | Kondisi |
|---|---|---|
| `E_CART_IN_USE` | Keranjang ini sedang dipakai tugas lain | scan cart yang IN_USE |
| `E_CART_MIXED` | Keranjang SINGLE tidak boleh dipakai untuk order MULTI (atau sebaliknya) | tipe campur |
| `E_SKU_NOT_IN_ORDER` | SKU tidak ada di order mana pun di keranjang ini | sortir/packing |
| `E_SKU_EXHAUSTED` | SKU ini sudah habis di keranjang ini | packing single |
| `E_BOX_NOT_FULL` | Kotak belum lengkap — masih ada SKU yang belum masuk | packing multi |
| `E_CART_FULL` | Keranjang sortir sudah dipakai | sesi sortir |
| `E_SLOT_FULL` | Kotak sudah lengkap | scan qty sortir |
| `E_QTY_SHORT` | Qty kurang dari yang dibutuhkan | short pick |
| `E_ORDER_ON_HOLD` | Pesanan sedang ditahan / dibatalkan | packing |
| `E_CART_EMPTY` | Keranjang kosong — tidak ada SKU untuk dipacking | packing single |
| `E_DUP_CODE` | Kode keranjang sudah ada | generate sortir |

Semua error: **bunyi + pesan singkat**, aksi tidak dilanjutkan.

---

## 8. Audit & Keamanan

- Setiap scan → `scan_logs` (user, device, result, reason, timestamp).
- Setiap pack → `packing_logs` + update `ord_orders.status` → `ORDER_PACKED`.
- **Tidak boleh** keranjang dipakai 2 proses bersamaan: unique `(picking_basket_id, status='IN_USE')`.
- Order batal/hold → saat packing, sistem tolak + tampilkan alasan.

---

## 9. Status Mapping Implementasi (local, `127.0.0.1:11666`)

Tabel baru yang sudah ada: `wms_waves`, `wms_wave_items`, `wms_picking_carts`,
`wms_picking_tasks`, `wms_picking_task_items`, `wms_picking_task_logs` — perlu **perluasan** ke skema baru di §2 (tabel `wave_settings`, `picking_baskets` perlu ditambah kolom tipe/status/current_*, tabel `sort_*` baru).

---

## 10. Roadmap Implementasi

| Fase | Lingkup | Estimasi |
|---|---|---|
| **A** | `wave_settings` + `picking_baskets` (perluasan) + `sort_baskets`/`sort_slots`/`sort_slot_items` + generate batch API | 1 modul |
| **B** | Sortir MULTI: sesi bind, scan SKU→kotak, complete | 1 modul |
| **C** | Packing SINGLE + resi cetak | 1 modul |
| **D** | Packing MULTI (verifikasi SKU) + release kotak | 1 modul |
| **E** | Pengaturan Wave UI (2 tab) + Master Keranjang Sortir UI | 1 modul |
| **F** | Frontend store/composable & integrasi scanner mobile | 1 modul |

**Implementasi sekarang (spec ini)**: hanya dokumen ini. Kode akan dibuat bertahap; konfirmasi fase mana yang dikerjakan duluan.
