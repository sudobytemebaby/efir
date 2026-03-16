#!/usr/bin/env bash
# Verifies that all required foundation files and directories exist.
set -euo pipefail

status=0

check_files() {
  local -a required=(
    .env.example
    .gitignore
    .golangci.yml
    README.md
    Taskfile.yml
    buf.gen.yaml
    buf.yaml
    docker-compose.yml
    deploy/compose/docker-compose.dev.example.yml
    deploy/compose/docker-compose.infra.yml
    deploy/compose/docker-compose.network.yml
    deploy/compose/docker-compose.observability.yml
    deploy/compose/docker-compose.services.yml
    deploy/compose/docker-compose.sidecar.yml
    docs/git-workflow.md
    docs/adr/001-monorepo.md
    docs/adr/002-sidecar-pep.md
    docs/adr/003-modular-compose.md
    docs/adr/004-go-workspace.md
    docs/adr/005-git-hooks-without-husky.md
    docs/adr/006-ci-pipeline.md
    infra/postgres/init/01_create_databases.sql
    infra/postgres/init/02_create_users.sh
    infra/nats/nats.conf
    infra/valkey/valkey.conf
    infra/traefik/traefik.yml
    infra/traefik/dynamic/middleware.yml
  )

  echo "Checking required files..."
  for path in "${required[@]}"; do
    if test -e "$path"; then
      echo "  ✓ $path"
    else
      echo "  ✗ Missing: $path"
      status=1
    fi
  done
}

check_dirs() {
  local -a required=(
    proto/efir/auth
    proto/efir/user
    proto/efir/room
    proto/efir/message
    services/auth
    services/user
    services/room
    services/message
    services/websocket
    services/gateway
    services/sidecar
    services/shared
  )

  echo "Checking required directories..."
  for path in "${required[@]}"; do
    if test -d "$path"; then
      echo "  ✓ $path"
    else
      echo "  ✗ Missing: $path"
      status=1
    fi
  done
}

check_files
check_dirs

exit "$status"
