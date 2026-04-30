#!/bin/sh
set -eu

template=/app/config.template.yaml
output=/app/config.yaml

mkdir -p /app/log /app/upload /home/devops/.ssh

escape_sed_replacement() {
  printf '%s' "$1" | sed 's/[&|\\]/\\&/g'
}

db_host=$(escape_sed_replacement "${DB_HOST:-postgres}")
db_port=$(escape_sed_replacement "${DB_PORT:-5432}")
db_name=$(escape_sed_replacement "${DB_NAME:-autoops}")
db_user=$(escape_sed_replacement "${DB_USER:-devops}")
db_password=$(escape_sed_replacement "${DB_PASSWORD:-devops@2025}")
redis_addr=$(escape_sed_replacement "${REDIS_ADDR:-valkey:6379}")
redis_password=$(escape_sed_replacement "${REDIS_PASSWORD:-devops@2025}")
image_host=$(escape_sed_replacement "${IMAGE_HOST:-http://localhost:18088}")
gitops_repo_path=$(escape_sed_replacement "${GITOPS_REPO_PATH:-/workspace/pukka-gitops}")

sed \
  -e "s|__DB_HOST__|$db_host|g" \
  -e "s|__DB_PORT__|$db_port|g" \
  -e "s|__DB_NAME__|$db_name|g" \
  -e "s|__DB_USER__|$db_user|g" \
  -e "s|__DB_PASSWORD__|$db_password|g" \
  -e "s|__REDIS_ADDR__|$redis_addr|g" \
  -e "s|__REDIS_PASSWORD__|$redis_password|g" \
  -e "s|__IMAGE_HOST__|$image_host|g" \
  -e "s|__GITOPS_REPO_PATH__|$gitops_repo_path|g" \
  "$template" > "$output"

exec /app/devops -c "$output"
