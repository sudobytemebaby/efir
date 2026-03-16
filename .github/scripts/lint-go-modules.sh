#!/usr/bin/env bash
# Runs golangci-lint across all modules in the Go workspace.
# Requires: golangci-lint on PATH, GITHUB_WORKSPACE set (or run from repo root).
set -euo pipefail

ROOT="${GITHUB_WORKSPACE:-$(git rev-parse --show-toplevel)}"
CONFIG="$ROOT/.golangci.yml"

echo "Running golangci-lint with config: $CONFIG"

for dir in $(go list -m -f '{{.Dir}}'); do
  echo "→ Linting $dir"
  (
    cd "$dir"
    golangci-lint run --config "$CONFIG" ./...
  ) || exit 1
done

echo "✓ All modules passed lint."
