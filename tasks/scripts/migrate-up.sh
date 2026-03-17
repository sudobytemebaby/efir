#!/usr/bin/env bash
# Applies goose migrations.
# Iterates over all services with a migrations/ directory, or runs for a single service.
#
# Usage:
#   migrate-up.sh [SERVICE]
#
# Required env vars:
#   POSTGRES_{SERVICE}_DSN for each service (e.g. POSTGRES_AUTH_DSN)
#   These are loaded automatically via dotenv in Taskfile.
set -euo pipefail

TARGET="${1:-}"

run_up() {
  local svc="$1"
  local dir="services/$svc/migrations"
  local var="POSTGRES_$(echo "$svc" | tr '[:lower:]' '[:upper:]')_DSN"
  local dsn
  dsn="$(printenv "$var" || true)"

  if [ -z "$dsn" ]; then
    echo "  Skipping $svc ($var is not set)"
    return 0
  fi

  echo "→ Applying migrations for $svc..."
  GOOSE_DRIVER=postgres GOOSE_DBSTRING="$dsn" GOOSE_MIGRATION_DIR="$dir" goose up
  echo "✓ $svc done."
}

if [ -n "$TARGET" ]; then
  run_up "$TARGET"
  exit 0
fi

for dir in services/*/migrations; do
  [ -d "$dir" ] || continue
  svc="$(basename "$(dirname "$dir")")"
  run_up "$svc"
done
