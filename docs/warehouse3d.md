# Warehouse 3D — Digital Twin & Custom Layout Designer

Fitur ini menambahkan **Warehouse 3D** ke modul FCL tanpa mengganti WMS yang sudah ada.

## Arsitektur

| Lapisan | Implementasi |
|---|---|
| Database | `backend/sql/warehouse3d_schema.sql` — auto-apply saat `admin-service` start (`go:embed`) |
| Backend | `backend/app/admin/service/internal/warehouse3d.go` — REST di `/api/warehouse3d/*` |
| Frontend | `frontend/admin/apps/admin/src/views/app/warehouse-3d/` — Vue 3 + Three.js |
| Widget 3D | `three` + `OrbitControls`, kamera isometric, raycasting, animasi worker/forklift |
| Editor 2D | Canvas 2D sinkron dengan scene 3D (grid, snap, zoom, pan, move, resize) |
| Real-time | SSE `/api/warehouse3d/activity/stream` (3 detik) |
| Karyawan | `wh3d_employee.employee_id → karyawan.id` (tanpa duplikat data) |
| Inventory | dibaca dari tabel `stock` (tidak ada inventory baru) |

## Endpoint

| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/warehouse3d/bootstrap` | Semua data layout + KPI + aktivitas + alert |
| GET | `/api/warehouse3d/kpi` | KPI dari `stock`, `order`, `wh3d_*` |
| POST | `/api/warehouse3d/save` | Simpan seluruh layout (draft/publish) |
| POST | `/api/warehouse3d/publish` | Buat snapshot versi + status PUBLISHED |
| GET | `/api/warehouse3d/versions?layoutId=` | Riwayat versi |
| POST | `/api/warehouse3d/restore` | Pulihkan versi |
| GET | `/api/warehouse3d/validate?layoutId=&warehouseId=` | Validasi tabrakan / batas / akses |
| GET | `/api/warehouse3d/search?q=` | Cari SKU, bin, karyawan, objek, order |
| GET | `/api/warehouse3d/activity/stream` | SSE aktivitas + alert + KPI |

## Deploy

1. Pull perubahan ini di server, lalu **redeploy** `admin-service` dan `admin` (frontend).
2. Saat start, `admin-service` otomatis membuat tabel `wh3d_*`, permission `fcl:warehouse3d`,
   dan menu **Inventaris & Logistik → Warehouse 3D**.
3. Buka menu Warehouse 3D. Gudang default `WH-001 / Gudang Utama` dibuat otomatis
   bila belum ada — semua dimensi bisa diedit lewat tombol **Info Gudang**
   (luas = panjang × lebar dihitung database).
4. Pekerja 3D otomatis terisi dari tabel `karyawan` yang aktif (bukan nama dummy).
   Aktivitas awal diturunkan dari `jabatan`/`divisi` (Receiving, Picking, Packing,
   Forklift, Supervisor, Manager, dll.).

## Catatan integrasi

- Semua objek layout disimpan di `wh3d_object` sebagai data (bukan kode hardcoded).
- Rack/bin dibuat dari **Rack Designer**; bin memiliki `location_code` yang bisa
  diisi dengan lokasi WMS (`stock.lokasi` / `sys_stock_location`).
- Tombol Publish tidak mengubah transaksi inventory; validasi tersedia sebelum publish.
- Restore versi mengganti geometri layout, bukan data transaksi.
- Layout bisa di-Export / Import JSON untuk backup atau perpindahan server.
