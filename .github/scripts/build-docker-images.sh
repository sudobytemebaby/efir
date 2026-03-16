#!/usr/bin/env bash
# Builds Docker images for all services that have a Dockerfile.
# Images are tagged as efir-{service}:ci.
set -euo pipefail

SERVICES=(gateway auth user message room websocket sidecar)

for svc in "${SERVICES[@]}"; do
  dockerfile="services/$svc/Dockerfile"
  if [ -f "$dockerfile" ]; then
    echo "→ Building $svc..."
    docker build -t "efir-$svc:ci" -f "$dockerfile" .
    echo "✓ efir-$svc:ci built."
  else
    echo "  Skipping $svc (no Dockerfile)"
  fi
done
