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

| Variable | Contoh | Keterangan |
|:---|:---|:---|
| `POSTGRES_PASSWORD` | `*Abcd123456` → **ganti** | Password DB. **Harus sama** dengan `source=` pada inline `configs.core_data` di `docker-compose.yaml` |
| `REDIS_PASSWORD` | `*Abcd123456` → **ganti** | Password Redis. **Harus sama** dengan `password:` pada inline `configs.*_data` di `docker-compose.yaml` |
| `MINIO_ROOT_USER` | `root` | User MinIO |
| `MINIO_ROOT_PASSWORD` | `*Abcd123456` → **ganti** | Password MinIO; juga harus sama dengan `secret_key` pada inline `configs.oss` di `docker-compose.yaml` |
| `APP_VERSION` | `1.0.0` | Tag image hasil build |

> ⚠️ **Penting**: password DB/Redis/MinIO pada inline `configs` di `docker-compose.yaml` adalah statis. Jika Anda mengubahnya di Coolify, nilai pada inline `configs` harus diedit juga lalu push, atau gunakan nilai default yang sama. Ini keterbatasan kratos config (YAML statis, tanpa env substitution). Config di `backend/deploy/compose/*.yaml` hanya dipakai oleh deploy lokal (docker-compose.full.yaml) — untuk Coolify yang relevan adalah inline configs.

## 4. Domain & Port (tab Domains di Coolify)

Tambahkan domain **hanya untuk `web`** (cukup satu domain):

| Service | Container Port | Domain contoh |
|:---|:---|:---|
| `web` | `8080` | `https://ghost.cloud` |

Nginx di `web` mem-proxy `/admin/*` → `admin-service:6900` dan `/events` → `admin-service:6901` secara otomatis, jadi frontend langsung bisa login tanpa variabel environment tambahan. Subdomain API (`api.*`, dst.) tidak diperlukan. Kalau tetap mau pakai subdomain API, set `VITE_GLOB_API_URL`/`VITE_GLOB_SSE_URL` sebagai build env (lihat §Dockerfile).

Catatan: tidak ada port yang dipublish ke host — Coolify merutekan domain langsung ke port internal container (`web`→8080, `admin-service`→6900/6901, `app-service`→6700/6701). Port host kosong untuk kebutuhan Anda sendiri. Port debug lokal bisa ditambahkan sementara di `docker-compose.yaml` jika perlu.

## 5. Konfigurasi yang masih perlu diedit (sekali saja)

1. **CORS** — `backend/app/admin/service/configs/server.yaml` dan `app/service/configs/server.yaml`: tambahkan origin production (`https://ghost.cloud`, dst.) ke `server.cors.allowed_origins` (saat ini hanya `http://127.0.0.1:11666` / `:5667`).
2. **MinIO publik** — `backend/deploy/compose/oss.yaml`: ganti `upload_host` / `download_host` dari `127.0.0.1:9000` ke alamat yang bisa diakses browser/App (mis. `https://minio.ghost.cloud` jika minio dipublish lewat domain, atau `http://<IP-server>:9000` jika port 9000 dibind `0.0.0.0`). Port 9000 di compose hanya bind `127.0.0.1`; ubah binding ke `0.0.0.0:9000` bila browser perlu akses langsung tanpa domain.
3. **Jaeger** — opsional. Jika tidak mau tracing, hapus service `jaeger` dan mount `trace.yaml` bawaan (atau kosongkan), atau biarkan saja (tracing gagal = log error, service tetap jalan).

## 6. Deploy & inisialisasi database

1. Klik **Deploy**. Build pertama memakan waktu (Go + pnpm).
2. Migration DB jalan otomatis saat `core-service` start (`migrate: true` di `core-data.yaml`).
3. (Opsional) Load data demo PostgreSQL:
   ```bash
   docker compose -f docker-compose.yaml exec -T postgres \
     psql -U postgres -d ghost_wms < backend/sql/postgresql-demo-data.sql
   ```
4. Login pertama memakai kredensial dari seed data (lihat `backend/sql/postgresql-demo-data.sql` / docs `deployment.md` bagian seed).

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
| Login gagal 500 | DB belum migrasi / core-service mati | Cek logs `core-service`; pastikan postgres sehat |
| Upload file gagal di browser | `oss.yaml` `upload_host` masih `127.0.0.1:9000` | Edit `oss.yaml` (lihat §5.2) → Redeploy |
| Image pull `quay.io/coreos/etcd` gagal | Koneksi ke quay.io bermasalah | Ganti image etcd ke mirror, mis. `gcr.io/etcd-development/etcd:v3.6.8` |
| `pull access denied for minio/minio` | Image MinIO ditarik dari Docker Hub oleh upstream | Sudah ditangani — MinIO dibangun dari GitHub release; pastikan service `minio` pakai `build:` seperti di compose |
| Password tidak cocok | `*-data.yaml` tidak sinkron dengan env Coolify | Samakan, push, redeploy |

## 10. Backup

Volume yang perlu dibackup: `postgres-data`, `minio-data`, `etcd-data`. Dengan Coolify: tab **Storages** di resource → backup otomatis ke S3-compatible storage. Untuk manual:

```bash
docker compose -f docker-compose.yaml exec postgres pg_dump -U postgres ghost_wms > backup.sql
```
