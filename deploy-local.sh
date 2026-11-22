#!/usr/bin/env bash
# Deploy lokal GhostWMS / Warehouse FCL dengan Docker Compose.
# Usage:
#   ./deploy-local.sh          # deploy / update
#   RESET=1 ./deploy-local.sh  # hapus volume DB lalu deploy dari kosong
set -euo pipefail

cd "$(dirname "$0")"

echo "==> Cek Docker"
if ! docker info >/dev/null 2>&1; then
  echo "Docker belum jalan. Coba: sudo systemctl start docker" >&2
  exit 1
fi

if [ ! -f .env ]; then
  cp .env.example .env
  echo "==> .env dibuat dari .env.example (silakan edit jika perlu)"
fi

mkdir -p backend/deploy/uploads
chmod 777 backend/deploy/uploads

if [ "${RESET:-0}" = "1" ]; then
  echo "==> RESET: menghapus container & volume"
  docker compose down -v --remove-orphans || true
fi

echo "==> Build & start container (bisa 15-25 menit untuk pertama kali)"
docker compose up -d --build

echo "==> Menunggu frontend siap di port ${PORT:-8082} ..."
for i in $(seq 1 120); do
  if curl -fsS -o /dev/null "http://localhost:${PORT:-8082}/"; then
    echo "==> Frontend siap"
    break
  fi
  if [ "$i" -eq 120 ]; then
    echo "==> Frontend belum siap setelah 120 detik. Log terakhir:" >&2
    docker compose logs --tail=60 >&2
    exit 1
  fi
  sleep 1
done

echo
echo "==> Status container"
docker compose ps

echo
echo "===================================================="
echo " Deploy selesai"
echo " Frontend : http://localhost:${PORT:-8082}/#/warehouse/dashboard"
echo " Login    : admin / admin (jika database baru)"
echo " Webhook  : http://localhost:${FCL_WEBHOOK_PORT:-8085}/fp"
echo " Log      : docker compose logs -f"
echo " Reset DB : RESET=1 ./deploy-local.sh"
echo "===================================================="
