## ADR-003: Modular Docker Compose

## Status

Accepted

## Context

A microservices project needs Docker Compose for local development. A single monolithic `docker-compose.yml` becomes unwieldy as services and infrastructure grow.

## Decision

Split docker-compose into modular files with the `include` directive (Docker Compose v2.20+):

- `docker-compose.network.yml` — network declaration
- `docker-compose.infra.yml` — infrastructure (Traefik, Postgres, NATS, Valkey)
- `docker-compose.services.yml` — Go microservices
- `docker-compose.sidecar.yml` — PEP sidecar containers
- `docker-compose.observability.yml` — monitoring stack (optional)
- `docker-compose.yml` — entrypoint with includes

## Rationale

- **Partial stack startup**: Run only infrastructure during development, add observability when needed
- **Separation of concerns**: Clear boundaries between infrastructure and services
- **Selective testing**: Start specific services without entire system
- **Team flexibility**: Developers can choose what to run

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
- `docker-compose up` runs everything
- `docker-compose -f docker-compose.infra.yml up` runs infrastructure only
- Development workflow: `task infra:up`, then selective service startup
