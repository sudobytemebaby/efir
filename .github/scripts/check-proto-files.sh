#!/usr/bin/env bash
# Checks whether any .proto files exist under proto/.
# Writes has_proto=true|false to GITHUB_OUTPUT if set, otherwise prints to stdout.
set -euo pipefail

if find proto/ -name "*.proto" 2>/dev/null | grep -q .; then
  echo "Proto files found."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_proto=true" >> "$GITHUB_OUTPUT"
else
  echo "No proto files found, skipping."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_proto=false" >> "$GITHUB_OUTPUT"
fi
