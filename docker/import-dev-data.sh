#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
cd "$SCRIPT_DIR"

if [[ ! -f .env ]]; then
  echo "docker/.env not found. Copy .env.example first." >&2
  exit 1
fi

set -a
source .env
set +a

: "${POSTGRES_USER:=devops}"
: "${POSTGRES_DB:=autoops}"

if ! docker compose ps postgres >/dev/null 2>&1; then
  echo "postgres service is not available. Run 'docker compose up -d postgres' first." >&2
  exit 1
fi

if [[ "${1:-}" != "--force" && "${AUTOOPS_SEED_FORCE:-0}" != "1" ]]; then
  cat >&2 <<'EOF'
Refusing to reset seed tables without confirmation.

This script truncates and rebuilds:
  sys_admin_role, sys_role_menu, sys_admin, sys_role,
  sys_menu, sys_post, sys_dept, cmdb_group

Re-run with:
  ./import-dev-data.sh --force

or set:
  AUTOOPS_SEED_FORCE=1
EOF
  exit 1
fi

echo "Resetting seed tables in ${POSTGRES_DB}..."
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" <<'SQL'
TRUNCATE TABLE
  sys_admin_role,
  sys_role_menu,
  sys_admin,
  sys_role,
  sys_menu,
  sys_post,
  sys_dept,
  cmdb_group
RESTART IDENTITY CASCADE;
SQL

echo "Importing docker/postgres/seed_data.sql ..."
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" < postgres/seed_data.sql

echo "Seed import completed. Default login user: admin / 123456"
