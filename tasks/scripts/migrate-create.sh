#!/usr/bin/env bash
# Creates a new goose migration file for a specific service.
#
# Usage:
#   migrate-create.sh SERVICE NAME
set -euo pipefail

SERVICE="${1:-}"
NAME="${2:-}"

if [ -z "$SERVICE" ]; then
  echo "✗ SERVICE is required."
  echo "  Usage: migrate-create.sh SERVICE NAME"
  exit 1
fi

if [ -z "$NAME" ]; then
  echo "✗ NAME is required."
  echo "  Usage: migrate-create.sh SERVICE NAME"
  exit 1
fi

DIR="services/$SERVICE/migrations"

if [ ! -d "$DIR" ]; then
  echo "✗ No migrations directory found: $DIR"
  echo "  Create it first: mkdir -p $DIR"
  exit 1
fi

echo "→ Creating migration '$NAME' for $SERVICE..."
GOOSE_MIGRATION_DIR="$DIR" goose create "$NAME" sql
echo "✓ Done."
