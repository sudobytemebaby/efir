#!/usr/bin/env bash
# Checks whether any .go files exist under services/.
# Writes has_go=true|false to GITHUB_OUTPUT if set, otherwise prints to stdout.
set -euo pipefail

# Use -quit to avoid SIGPIPE issues with pipefail + grep -q
if find services/ -name "*.go" -print -quit 2>/dev/null | read -r _; then
  echo "Go files found."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_go=true" >> "$GITHUB_OUTPUT"
else
  echo "No Go files found, skipping."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_go=false" >> "$GITHUB_OUTPUT"
fi
