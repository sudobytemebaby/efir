#!/usr/bin/env bash
# Checks whether any .go files exist under services/.
# Writes has_go=true|false to GITHUB_OUTPUT if set, otherwise prints to stdout.
set -euo pipefail

echo "DEBUG: pwd=$(pwd)"
echo "DEBUG: ls services/"
ls -la services/ 2>&1 || echo "DEBUG: services/ does not exist"
echo "DEBUG: find results:"
find services/ -name "*.go" 2>&1 | head -20 || true

if find services/ -name "*.go" 2>/dev/null | grep -q .; then
  echo "Go files found."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_go=true" >> "$GITHUB_OUTPUT"
else
  echo "No Go files found, skipping."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_go=false" >> "$GITHUB_OUTPUT"
fi
