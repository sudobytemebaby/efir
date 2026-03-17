#!/usr/bin/env bash
# Rolls back the last goose migration for a specific service.
#
# Usage:
#   migrate-down.sh SERVICE
#
# Required env vars:
#   POSTGRES_{SERVICE}_DSN (e.g. POSTGRES_AUTH_DSN)
#   These are loaded automatically via dotenv in Taskfile.
set -euo pipefail

SERVICE="${1:-}"

if [ -z "$SERVICE" ]; then
  echo "✗ SERVICE is required."
  echo "  Usage: migrate-down.sh SERVICE"
  exit 1
fi

DIR="services/$SERVICE/migrations"

if [ ! -d "$DIR" ]; then
  echo "✗ No migrations directory found: $DIR"
  exit 1
fi

VAR="POSTGRES_$(echo "$SERVICE" | tr '[:lower:]' '[:upper:]')_DSN"
DSN="$(printenv "$VAR" || true)"

if [ -z "$DSN" ]; then
  echo "✗ $VAR is not set."
  exit 1
fi

echo "→ Rolling back last migration for $SERVICE..."
GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DSN" GOOSE_MIGRATION_DIR="$DIR" goose down
echo "✓ Done."
