#!/usr/bin/env bash
# Builds Docker images for all services that have a Dockerfile.
# Images are tagged as efir-{service}:ci.
set -euo pipefail

found=0

for dockerfile in services/*/Dockerfile; do
  [ -f "$dockerfile" ] || continue
  svc=$(basename "$(dirname "$dockerfile")")
  echo "→ Building $svc..."
  docker build -t "efir-$svc:ci" -f "$dockerfile" .
  echo "✓ efir-$svc:ci built."
  found=1
done

if [ "$found" -eq 0 ]; then
  echo "No Dockerfiles found under services/."
fi
