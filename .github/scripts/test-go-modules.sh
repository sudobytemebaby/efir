#!/usr/bin/env bash
# Runs go test with coverage across all modules in the Go workspace.
# Uses -tags integration to run both unit and integration tests in a single pass.
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

for dir in $(go list -m -f '{{.Dir}}'); do
  echo "-> Testing $dir"
  tmp=$(mktemp)

  (
    cd "$dir"
    go test -tags integration -coverprofile="$tmp" -covermode=atomic ./...
  ) || {
    rm -f "$tmp"
    exit 1
  }

  merge_coverage "$tmp"
  rm -f "$tmp"
done

echo "✓ All modules passed tests. Coverage written to $OUTPUT."
