#!/usr/bin/env bash
# ==============================================================
# scripts/migrate-all.sh
#
# Jalankan golang-migrate untuk semua 4 service.
# Script ini membaca .env dari root project.
#
# PRASYARAT:
#   1. golang-migrate sudah terinstall:
#      https://github.com/golang-migrate/migrate/releases
#      Atau: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
#   2. Keempat database sudah dibuat (jalankan scripts/create-databases.sql dulu)
#   3. File .env sudah diisi dengan kredensial PostgreSQL kamu
#
# CARA MENJALANKAN (dari root folder order-processing-system/):
#   chmod +x scripts/migrate-all.sh   (pertama kali saja)
#   ./scripts/migrate-all.sh
#
# UNTUK ROLLBACK semua:
#   ./scripts/migrate-all.sh down
# ==============================================================

set -euo pipefail

# ── Cari root project (folder yang berisi .env) ───────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# ── Load .env ─────────────────────────────────────────────────
ENV_FILE="$PROJECT_ROOT/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ File .env tidak ditemukan di $PROJECT_ROOT"
    echo "   Salin .env.example ke .env dan isi kredensial PostgreSQL kamu:"
    echo "   cp .env.example .env"
    exit 1
fi

# Export semua variabel dari .env (abaikan baris komentar & kosong)
set -o allexport
# shellcheck source=/dev/null
source "$ENV_FILE"
set +o allexport

# ── Tentukan arah migrasi ─────────────────────────────────────
DIRECTION="${1:-up}"
if [[ "$DIRECTION" != "up" && "$DIRECTION" != "down" ]]; then
    echo "❌ Argument tidak valid: '$DIRECTION'. Gunakan 'up' atau 'down'."
    exit 1
fi

# ── Cek migrate CLI tersedia ──────────────────────────────────
if ! command -v migrate &>/dev/null; then
    echo "❌ golang-migrate CLI tidak ditemukan di PATH."
    echo ""
    echo "   Install dengan salah satu cara berikut:"
    echo "   • go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    echo "   • Download binary dari: https://github.com/golang-migrate/migrate/releases"
    echo ""
    echo "   Setelah install, pastikan binary 'migrate' ada di PATH kamu."
    exit 1
fi

echo ""
echo "=========================================================="
echo "  Order Processing System — Database Migration"
echo "  Direction : $DIRECTION"
echo "  Timestamp : $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================================="
echo ""

# ── Fungsi helper untuk migrate satu service ─────────────────
run_migrate() {
    local service_name="$1"
    local db_url="$2"
    local migration_path="$PROJECT_ROOT/services/${service_name}/migrations"

    echo "──────────────────────────────────────────────────────"
    echo "  📦 Service : $service_name"
    echo "  🗄️  Database: ${db_url%%\?*}"   # tampilkan URL tanpa query params
    echo "  📂 Path    : $migration_path"
    echo "──────────────────────────────────────────────────────"

    if [ ! -d "$migration_path" ]; then
        echo "  ⚠️  Folder migrations tidak ditemukan, skip."
        return
    fi

    if migrate -path "$migration_path" -database "$db_url" "$DIRECTION"; then
        echo "  ✅ $service_name migration '$DIRECTION' berhasil!"
    else
        echo "  ❌ $service_name migration '$DIRECTION' GAGAL!"
        exit 1
    fi
    echo ""
}

# ── Jalankan migrasi untuk semua service ─────────────────────
run_migrate "order-service"        "$ORDER_DB_URL"
run_migrate "inventory-service"    "$INVENTORY_DB_URL"
run_migrate "payment-service"      "$PAYMENT_DB_URL"
run_migrate "notification-service" "$NOTIFICATION_DB_URL"

echo "=========================================================="
echo "  ✅ Semua migrasi selesai dengan sukses!"
echo "=========================================================="
echo ""

# ── Hint seed data ────────────────────────────────────────────
if [[ "$DIRECTION" == "up" ]]; then
    echo "  💡 Jangan lupa jalankan seed data untuk inventory:"
    echo "     psql \"\$INVENTORY_DB_URL\" -f services/inventory-service/migrations/seed.sql"
    echo ""
fi
