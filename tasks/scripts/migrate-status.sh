#!/usr/bin/env bash
# Shows goose migration status.
# Iterates over all services with a migrations/ directory, or runs for a single service.
#
# Usage:
#   migrate-status.sh [SERVICE]
#
# Required env vars:
#   POSTGRES_{SERVICE}_DSN for each service (e.g. POSTGRES_AUTH_DSN)
#   These are loaded automatically via dotenv in Taskfile.
set -euo pipefail

TARGET="${1:-}"

print_status() {
  local svc="$1"
  local dir="services/$svc/migrations"
  local var="POSTGRES_$(echo "$svc" | tr '[:lower:]' '[:upper:]')_DSN"
  local dsn
  dsn="$(printenv "$var" || true)"

  if [ -z "$dsn" ]; then
    echo "  Skipping $svc ($var is not set)"
    return 0
  fi

  echo "=== $svc ==="
  GOOSE_DRIVER=postgres GOOSE_DBSTRING="$dsn" GOOSE_MIGRATION_DIR="$dir" goose status
  echo ""
}

if [ -n "$TARGET" ]; then
  print_status "$TARGET"
  exit 0
fi

for dir in services/*/migrations; do
  [ -d "$dir" ] || continue
  svc="$(basename "$(dirname "$dir")")"
  print_status "$svc"
done
