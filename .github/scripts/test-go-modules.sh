#!/usr/bin/env bash
# Runs go test with coverage across all modules in the Go workspace.
# Tests are run in two phases:
#   1. Unit tests (fast, no external deps)
#   2. Integration tests (require Docker via testcontainers)
# Merges per-module coverage profiles into a single coverage.out at repo root.
set -euo pipefail

OUTPUT="coverage.out"
rm -f "$OUTPUT"
touch "$OUTPUT"
first=1

merge_coverage() {
  local tmp="$1"
  if [ -s "$tmp" ]; then
    if [ "$first" -eq 1 ]; then
      cat "$tmp" > "$OUTPUT"
      first=0
    else
      tail -n +2 "$tmp" >> "$OUTPUT"
    fi
  fi
}

echo "=== Phase 1: Unit tests ==="
for dir in $(go list -m -f '{{.Dir}}'); do
  echo "→ Unit testing $dir"
  tmp=$(mktemp)

  (
    cd "$dir"
    go test -coverprofile="$tmp" -covermode=atomic ./...
  ) || {
    rm -f "$tmp"
    exit 1
  }

  merge_coverage "$tmp"
  rm -f "$tmp"
done

echo "=== Phase 2: Integration tests (requires Docker) ==="
for dir in $(go list -m -f '{{.Dir}}'); do
  echo "→ Integration testing $dir"
  tmp=$(mktemp)

  (
    cd "$dir"
    go test -tags integration -coverprofile="$tmp" -covermode=atomic ./...
  ) || {
    rm -f "$tmp"
    exit 1
  }

  # Only append integration coverage (skip duplicate mode line)
  if [ -s "$tmp" ]; then
    tail -n +2 "$tmp" >> "$OUTPUT"
  fi
  rm -f "$tmp"
done

echo "✓ All modules passed tests. Coverage written to $OUTPUT."
