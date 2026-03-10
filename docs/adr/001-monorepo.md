## ADR-001: Monorepo Structure

## Status

Accepted

## Context

The messenger system requires multiple services: Auth, User, Room, Message, WebSocket, Gateway. We need to decide whether to use a monorepo (single repository) or polyrepo (separate repositories per service).

## Decision

Use a monorepo with Go workspaces (`go.work`).

## Rationale

- **Single source of truth**: Proto files in one place, no duplication
- **Cross-service refactoring**: Changes affecting multiple services in one PR
- **Shared code**: Common utilities in `services/shared` without publishing packages
- **Single CI/CD pipeline**: One repository to configure
- **Consistent tooling**: golangci-lint, buf, go version across all services

## Alternatives Considered

- **Polyrepo**: Each service in its own repository
  - Pros: Independent deployments, clear boundaries
  - Cons: Proto duplication or separate proto repo, cross-service changes span multiple PRs, no shared code without publishing

## Consequences

- All services built and tested together in CI
- Proto files generated once, shared across services
- `docker-compose` manages entire stack
- Requires discipline to avoid coupling between services
