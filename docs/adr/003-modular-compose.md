## ADR-003: Modular Docker Compose

## Status

Accepted

## Context

A microservices project needs Docker Compose for local development. A single monolithic `docker-compose.yml` becomes unwieldy as services and infrastructure grow.

## Decision

Split docker-compose into modular files with the `include` directive (Docker Compose v2.20+) and keep host port exposure in a dev-only override file:

- `docker-compose.network.yml` — network declaration
- `docker-compose.infra.yml` — infrastructure (Traefik, Postgres, NATS, Valkey)
- `docker-compose.services.yml` — Go microservices
- `docker-compose.sidecar.yml` — PEP sidecar containers
- `docker-compose.observability.yml` — monitoring stack (optional)
- `docker-compose.dev.yml` — local-only port publishing override copied from `docker-compose.dev.example.yml`
- `docker-compose.yml` — entrypoint with includes

## Rationale

- **Partial stack startup**: Run only infrastructure during development, add observability when needed
- **Separation of concerns**: Clear boundaries between infrastructure and services
- **Selective testing**: Start specific services without entire system
- **Team flexibility**: Developers can choose what to run
- **Safer defaults**: Base compose files stay internal-only; only the dev override publishes ports to the host

## Alternatives Considered

- **Single monolithic file**:
  - Pros: Simple, single command
  - Cons: Hard to maintain, can't run partial stacks, everyone runs everything

- **Shell scripts to compose**:
  - Pros: Flexible
  - Cons: Additional abstraction, inconsistent UX

## Consequences

- Order of includes matters: network first, dev last (overrides)
- Requires Docker Compose v2.20+
- `task setup` must create `docker-compose.dev.yml` before the first full `docker compose up`
- Base compose files do not expose ports directly; local access comes from the dev override
- Partial startup is done through Task commands that target named services via the unified entrypoint
- Sidecar and observability services stay behind Compose profiles until their modules are implemented, so the default startup path remains focused on MVP dependencies
