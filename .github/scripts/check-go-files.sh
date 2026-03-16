#!/usr/bin/env bash
# Checks whether any .go files exist under services/.
# Writes has_go=true|false to GITHUB_OUTPUT if set, otherwise prints to stdout.
set -euo pipefail

if find services/ -name "*.go" 2>/dev/null | grep -q .; then
  echo "has_go=true"
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_go=true" >> "$GITHUB_OUTPUT"
else
  echo "has_go=false"
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_go=false" >> "$GITHUB_OUTPUT"
  echo "No Go files found, skipping."
fi
