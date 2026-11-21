# Deploy GhostWMS dengan Coolify

Panduan ini untuk deploy **seluruh stack GhostWMS** (3 backend service + PostgreSQL + Redis + MinIO + etcd + Jaeger + frontend admin) ke [Coolify](https://coolify.io) menggunakan **Docker Compose build pack**.

File compose: [`docker-compose.yaml`](../docker-compose.yaml) (repo root) — dipakai apa adanya oleh Coolify, juga bisa dipakai manual dengan `docker compose -f docker-compose.yaml up -d --build`.

---

> 📦 **Catatan image MinIO**: image resmi `minio/minio` sudah ditarik dari Docker Hub oleh upstream (2025-09). Compose membangun MinIO dari binary resmi GitHub release (`backend/deploy/minio/Dockerfile`, sha256 terverifikasi), jadi tidak perlu menarik image manapun.

## 1. Prasyarat

- Coolify v4 yang sudah jalan di server (bisa di root/public domain).
- DNS A records untuk tiap domain/subdomain yang dipakai, mis.:
  - `ghost.cloud` → IP server (frontend admin)
  - `api.ghost.cloud` → IP server (admin-service, REST :6900)
  - `sse.ghost.cloud` → IP server (admin-service, SSE :6901, path `/events`)
  - `api-app.ghost.cloud` → IP server (app-service, untuk Flutter App :6700)
  - `sse-app.ghost.cloud` → IP server (app-service :6701)
  - `minio.ghost.cloud` → IP server (opsional; untuk upload file via browser)
- Repo `Mikdad-Force/Ghost` (private) — hubungkan Coolify dengan GitHub App / deploy key agar bisa clone repo private.

## 2. Buat resource di Coolify

1. **New Project → New Resource → Docker Compose**
2. **Source**: pilih repository `Mikdad-Force/Ghost`, branch `main`
3. **Docker Compose Location**: `docker-compose.yaml`
4. **Build**: Coolify akan mem-build image `core-service`, `app-service`, `admin-service` (dari `backend/Dockerfile`) dan `web` (dari `frontend/admin/scripts/deploy/Dockerfile`) langsung dari source — tidak perlu registry.

## 3. Environment Variables (isi di tab Environment Coolify)

Config YAML sudah ditanamkan ke dalam image dan **dirender dari environment variable saat container start** oleh `backend/deploy/entrypoint.sh`. Jadi cukup set variabel di Coolify — tidak perlu mengedit YAML atau push ulang hanya untuk mengganti password.

| Variable | Default | Keterangan |
|:---|:---|:---|
| `POSTGRES_PASSWORD` | `*Abcd123456` | Password Postgres. Otomatis dipakai oleh `postgres`, `core-service` dan `admin-service` |
| `REDIS_PASSWORD` | `*Abcd123456` | Password Redis. Otomatis dipakai oleh `redis`, `core-service`, `admin-service`, `app-service` |
| `MINIO_ROOT_PASSWORD` | `*Abcd123456` | Password MinIO; otomatis dipakai sebagai `secret_key` OSS |
| `JWT_SIGNING_KEY` | `dev_only_change_me_in_prod` | **Wajib diganti di produksi** — kunci tanda tangan JWT |
| `AES_KEY` | `f51d66a73d8a0927` | **Wajib diganti di produksi** — 16/24/32 byte |
| `APP_VERSION` | `1.0.0` | Tag image hasil build |

Opsional (default sudah benar untuk Compose/Coolify): `DB_HOST=postgres`, `DB_PORT=5432`, `DB_USER=postgres`, `DB_NAME=ghost_wms`, `REDIS_ADDR=redis:6379`, `ETCD_ENDPOINT=etcd:2379`, `MINIO_ENDPOINT=minio:9000`, `MINIO_UPLOAD_HOST`, `MINIO_DOWNLOAD_HOST`, `TRACE_ENDPOINT=jaeger:4317`.

> ⚠️ Password harus diset **sekali di Environment Coolify**, lalu **Redeploy** supaya image ter-build ulang dengan variabel tersebut. Jika Anda mengubah password hanya di satu service, service lain akan tetap memakai default dan login gagal dengan `db error`.

## 4. Domain & Port (tab Domains di Coolify)

Tambahkan domain **hanya untuk `web`** (cukup satu domain):

| Service | Container Port | Domain contoh |
|:---|:---|:---|
| `web` | `8080` | `https://ghost.cloud` |

Nginx di `web` mem-proxy `/admin/*` → `admin-service:6900` dan `/events` → `admin-service:6901` secara otomatis, jadi frontend langsung bisa login tanpa variabel environment tambahan. Subdomain API (`api.*`, dst.) tidak diperlukan. Kalau tetap mau pakai subdomain API, set `VITE_GLOB_API_URL`/`VITE_GLOB_SSE_URL` sebagai build env (lihat §Dockerfile).

Catatan: tidak ada port yang dipublish ke host — Coolify merutekan domain langsung ke port internal container (`web`→8080, `admin-service`→6900/6901, `app-service`→6700/6701). Port host kosong untuk kebutuhan Anda sendiri. Port debug lokal bisa ditambahkan sementara di `docker-compose.yaml` jika perlu.

## 5. Konfigurasi yang masih perlu diedit (sekali saja)

1. **CORS** — `backend/app/admin/service/configs/server.yaml` dan `app/service/configs/server.yaml`: tambahkan origin production (`https://ghost.cloud`, dst.) ke `server.cors.allowed_origins` (saat ini hanya `http://127.0.0.1:11666` / `:5667`). Perubahan ini butuh rebuild image.
2. **MinIO publik** — cukup set `MINIO_UPLOAD_HOST` / `MINIO_DOWNLOAD_HOST` di Environment Coolify ke alamat yang bisa diakses browser/App (mis. `https://minio.ghost.cloud`). Port 9000 di compose hanya bind `127.0.0.1`; ubah binding ke `0.0.0.0:9000` bila browser perlu akses langsung tanpa domain.
3. **Jaeger** — opsional. Jika tidak mau tracing, hapus service `jaeger` dan set `TRACE_ENDPOINT` ke alamat lain, atau biarkan saja (tracing gagal = log error, service tetap jalan).

## 6. Deploy & inisialisasi database

1. Pastikan Environment variables di §3 sudah diisi (minimal password kalau diganti), lalu klik **Deploy**. Build pertama memakan waktu (Go + pnpm).
2. **Migration DB jalan otomatis** saat `core-service` start (`migrate: true` di `backend/deploy/compose/core-data.yaml`). Tidak perlu menjalankan SQL manual untuk membuat tabel.
3. Saat tabel `sys_users` masih kosong, `core-service` otomatis membuat akun super admin **`admin` / `admin`**. Segera ganti passwordnya setelah login.
4. (Opsional) Data demo: `WMS_DEMO_SEED=true` pada `core-service` akan membuat tenant demo + admin `admin/admin` beserta sample warehouse/produk/stock. Jangan dipakai di produksi.
5. (Opsional) Load data demo PostgreSQL — **hanya setelah tabel ada**:
   ```bash
   docker compose -f docker-compose.yaml exec -T postgres \
     psql -U postgres -d ghost_wms < backend/sql/postgresql-demo-data.sql
   ```

## 7. Update / redeploy

- Otomatis: Coolify bisa poll git / webhook — setiap push ke `main` memicu redeploy otomatis.
- Gambar hasil build diberi tag `APP_VERSION` (default `1.0.0`). Naikkan `APP_VERSION` untuk rollback cepat.

## 8. Verifikasi setelah deploy

- Frontend: `https://ghost.cloud` terbuka, login page tampil.
- API: `https://api.ghost.cloud/` (atau endpoint publik admin-service) menjawab, bukan 502/504.
- SSE: DevTools Network → `https://sse.ghost.cloud/events` terhubung (status `text/event-stream`).
- Logs per service di Coolify (tab **Logs**): `core-service` harus "service registered", `admin-service` "started".
- Jaeger UI: `http://127.0.0.1:16686` (SSH tunnel) menampilkan trace.

## 9. Troubleshooting

| Gejala | Kemungkinan penyebab | Solusi |
|:---|:---|:---|
| 502/504 ke API | admin/app service belum start / salah port domain | Cek logs `admin-service`; pastikan domain mengarah ke port 6900, bukan 6901 |
| Frontend kosong / API tidak kepanggil | `VITE_GLOB_API_URL` salah saat build | Set env di Coolify → Redeploy (env build-time, bukan runtime) |
| **Login gagal, pesan `db error`** | Tabel belum ada (migration tidak jalan) atau koneksi DB ditolak / password salah | 1) Cek logs `core-service` — cari `[ENT] failed creating schema resources` atau `dial tcp ... connection refused` / `password authentication failed`. 2) Pastikan `migrate: true` di `backend/deploy/compose/core-data.yaml` dan image di-build ulang. 3) Pastikan `POSTGRES_PASSWORD` di Environment Coolify sama untuk `postgres` dan `core-service`. 4) Cek healthcheck `postgres`. Setelah diperbaiki: **Redeploy**, lalu login `admin` / `admin` |
| Upload file gagal di browser | `oss.yaml` `upload_host` masih `127.0.0.1:9000` | Edit `oss.yaml` (lihat §5.2) → Redeploy |
| Image pull `quay.io/coreos/etcd` gagal | Koneksi ke quay.io bermasalah | Ganti image etcd ke mirror, mis. `gcr.io/etcd-development/etcd:v3.6.8` |
| `pull access denied for minio/minio` | Image MinIO ditarik dari Docker Hub oleh upstream | Sudah ditangani — MinIO dibangun dari GitHub release; pastikan service `minio` pakai `build:` seperti di compose |
| Password tidak cocok | Env password diubah tapi lupa redeploy / hanya di satu service | Set password sekali di Environment Coolify (tab Project, bukan per-service), lalu Redeploy semua service |

## 10. Backup

Volume yang perlu dibackup: `postgres-data`, `minio-data`, `etcd-data`. Dengan Coolify: tab **Storages** di resource → backup otomatis ke S3-compatible storage. Untuk manual:

```bash
docker compose -f docker-compose.yaml exec postgres pg_dump -U postgres ghost_wms > backup.sql
```

## 12. Catatan backend FCL terintegrasi

Backend FCL sekarang **berjalan di dalam `admin-service`** (port internal `6905`), bukan service terpisah. Nginx mem-proxy `/api/*`, `/iclock/*` dan `/fp` ke `admin-service:6905`.

- Pastikan folder `backend/deploy/uploads` di server bisa ditulis oleh container:
  ```bash
  mkdir -p backend/deploy/uploads && chmod 777 backend/deploy/uploads
  ```
- Kredensial Google Drive tetap dibaca dari `backend/deploy/warehouse-drive.env` (opsional).
- Jika DB FCL gagal dibuka, `admin-service` tetap berjalan; hanya endpoint FCL yang tidak aktif (log: `fcl backend disabled`).

### Autentikasi FCL = login utama

Modul FCL tidak punya login sendiri. Frontend mengirim access token JWT yang sama
dengan request axios (`Authorization: Bearer <token>`), dan backend FCL di
`admin-service` memverifikasi token tersebut (HS256, kunci `JWT_SIGNING_KEY`).

- Endpoint mesin (`/iclock/cdata`, `/fp`, `/api/fingerprint-webhook`,
  `/api/fingerprint-check`, `/api/health`) tetap tanpa token.
- SSE Warehouse 3D mengirim token lewat query `access_token` karena `EventSource`
  tidak bisa mengirim header.
- Endpoint lama `/api/auth/login` dan `/api/auth/change-password` sudah dihapus.
- Jika Anda mengganti kunci JWT, set `JWT_SIGNING_KEY` di Environment Coolify
  (dan samakan dengan konfigurasi authenticator core-service).

### Webhook Fingerprint di port 80

Backend FCL membuka listener khusus di **port 80** (`FCL_WEBHOOK_ADDR`, default `:80`) untuk:
`/iclock/cdata`, `/fp`, `/api/fingerprint-webhook`, `/api/fingerprint-check`, `/api/health`.
Endpoint lain tetap di `FCL_ADDR` (6905) dan memakai JWT utama.

- Image `admin-service` diberi capability `cap_net_bind_service` agar user non-root bisa bind port 80.
- Port 80 **hanya terbuka di dalam jaringan Docker** (`expose: ["80"]`), tidak di-publish ke host
  karena port 80 host biasanya dipakai proxy Coolify. Mesin mengakses lewat domain aplikasi:
  `https://<domain>/iclock/cdata?SN=...` dan `https://<domain>/fp`.
- Nginx mem-proxy `/iclock/` dan `/fp` ke `admin-service:80`.
- Jika memang perlu akses langsung IP:port, tambahkan sendiri mapping host di Compose, misalnya
  `"8085:80"`, lalu sesuaikan URL di mesin.
