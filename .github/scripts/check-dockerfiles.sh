#!/usr/bin/env bash
# Checks whether any Dockerfiles exist under services/.
# Writes has_dockerfiles=true|false to GITHUB_OUTPUT if set, otherwise prints to stdout.
set -euo pipefail

if find services/ -name "Dockerfile" 2>/dev/null | grep -q .; then
  echo "Dockerfiles found."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_dockerfiles=true" >> "$GITHUB_OUTPUT"
else
  echo "No Dockerfiles found, skipping."
  [ -n "${GITHUB_OUTPUT:-}" ] && echo "has_dockerfiles=false" >> "$GITHUB_OUTPUT"
fi
