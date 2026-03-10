# Efir

Efir is a Go-based microservice messenger built as a single monorepo. Module 0 establishes the development foundation: repository layout, modular Docker Compose, a shared Taskfile, Git workflow, CI, and the base infrastructure needed for the next modules.

## Technology Stack

- Go 1.23 for all backend services and shared packages
- gRPC + Protocol Buffers for service-to-service contracts
- Buf for proto linting and code generation
- PostgreSQL for persistent service data
- NATS JetStream for asynchronous events
- Valkey for caching, refresh tokens, rate limiting, and later Pub/Sub
- Traefik as the local entrypoint and reverse proxy
- Docker Compose as the local orchestration layer
- Task as the single developer entrypoint
- GitHub Actions for CI
- OpenTelemetry + Prometheus + Loki + Tempo + Grafana for observability in Module 2

Observability and sidecar support are scaffolded in Module 0, but they are kept out of the default startup path until the matching modules are implemented.

## Architecture Direction

The repository follows a monorepo layout: every service, proto contract, infrastructure config, and architectural decision lives in one place. The goal is to keep cross-service changes atomic and make local development predictable.

Core runtime flow in the MVP:

1. Traefik accepts HTTP/WebSocket traffic.
2. Gateway handles public API requests and forwards internal calls over gRPC.
3. Domain services own their data and publish events through NATS.
4. WebSocket Connector pushes realtime updates to clients.

## Sidecar PEP

The sidecar is planned as a Policy Enforcement Point (PEP) that sits next to a service and validates protobuf traffic before it reaches the upstream gRPC server.

Expected flow:

1. Request enters sidecar.
2. Sidecar validates route, payload shape, and required fields against policy.
3. Valid traffic is proxied to the upstream service.
4. Invalid traffic is rejected before it reaches business logic.

Why Go:

- same language and toolchain as the rest of the backend
- native gRPC and protobuf support
- simpler operational model for one-developer monorepo
- easier shared libraries for logging, config, healthchecks, tracing, and middleware

## Initial Setup

Prerequisites:

- Docker + Docker Compose v2.20+
- Go 1.23+
- Task
- Buf
- `golangci-lint`
- `goose`

Bootstrap the project:

```bash
task setup
```

What `task setup` does:

- creates the shared Docker network
- copies `.env.example` to `.env` if needed
- copies `docker-compose.dev.example.yml` to `docker-compose.dev.yml` if needed
- configures local Git hooks from `.githooks/`

Base Compose files keep services internal to the Docker network. Host port publishing lives in `docker-compose.dev.yml`, which is why `task setup` is required before the first full `docker compose up`.

After setup, review `.env` and replace placeholder values before running the full stack.

## Main Commands

```bash
task up            # start the full stack
task down          # stop the stack
task ps            # show container status
task logs SERVICE=gateway
task restart SERVICE=auth

task infra:up      # infrastructure only
task infra:down
task obs:up        # infrastructure + observability
task obs:down
task sidecar:up
task sidecar:down

task proto:lint
task proto
task generate
task run SERVICE=auth
task test
task lint

task migrate:up
task migrate:up SERVICE=auth
task migrate:down SERVICE=auth
task migrate:create SERVICE=auth NAME=create_accounts
task migrate:status
```

## Repository Layout

```text
.
├── .github/                   # CI workflows
├── docs/                      # architecture notes, ADRs, Git workflow
├── infra/                     # Traefik, Postgres, NATS, Valkey, observability configs
├── proto/                     # protobuf contracts
├── services/                  # Go services and shared module
├── Taskfile.yml               # single entrypoint for developer tasks
├── docker-compose*.yml        # modular local orchestration
├── buf.yaml                   # proto lint/breaking rules
├── buf.gen.yaml               # proto generation targets
└── go.work                    # Go workspace for the monorepo
```

Important service groups:

- `services/gateway` - public HTTP entrypoint
- `services/auth`, `services/user`, `services/room`, `services/message` - core domain services
- `services/websocket` - realtime delivery
- `services/sidecar` - scaffolded policy enforcement proxy reserved for the sidecar module
- `services/shared` - reusable packages shared across services

Proto contracts live in `proto/` and generate shared Go bindings into `services/shared/gen`.

## Development Modules

- Module 1 - MVP: auth, users, rooms, messages, gateway, websocket delivery, base tests
- Module 2 - sidecar + observability + websocket scaling: PEP sidecar, OTel stack, Valkey Pub/Sub for horizontal websocket delivery
- Module 3 - feature expansion: presence, media, notification, and search services

The execution plan lives in `.instruct.md`.

## Git Workflow

Branch names and commit messages follow the repository conventions documented in `docs/git-workflow.md`.

In short:

- branch format: `<type>/<short-description>`
- commit format: `<type>(<scope>): <description>`
- one branch and one commit should map to one logical change

## Current Foundation Files

- `docker-compose.yml` is the Compose entrypoint
- `docker-compose.network.yml` defines the shared network
- `docker-compose.infra.yml` defines infrastructure services
- `docker-compose.services.yml` defines application services
- `docker-compose.sidecar.yml` defines sidecar containers
- `docker-compose.observability.yml` defines the observability stack
- `docker-compose.dev.example.yml` documents the local development override

## ADRs

Key decisions already documented:

- `docs/adr/001-monorepo.md`
- `docs/adr/002-sidecar-pep.md`
- `docs/adr/003-modular-compose.md`
- `docs/adr/004-go-workspace.md`
- `docs/adr/005-git-hooks-without-husky.md`
- `docs/adr/006-ci-pipeline.md`
