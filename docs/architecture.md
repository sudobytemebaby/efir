# Architecture

This document describes the architecture of Efir: how services interact, how data flows, and the design principles behind the system.

## Design Principles

1. **One service, one database** -- each service owns its data and schema
2. **Clean architecture** -- `handler -> service -> repository` layers in every service
3. **Async by default** -- services communicate via NATS JetStream events; gRPC is used only when a synchronous response is required
4. **Self-hostable** -- the entire stack runs with `docker compose up`
5. **Config via environment** -- each service reads YAML config overlaid with environment variables (via cleanenv)

## System Overview

```
Client (browser / mobile)
         |
    nginx :80
    (reverse proxy, CORS, rate limiting, forward auth)
         |
    +----+----+
    |         |
    v         v
 Gateway    WebSocket
  :8080      :8081
  (HTTP)     (WS)
    |         |
    |    +----+----+
    |    |         |
    v    v         v
  gRPC services   NATS JetStream
    |                |
    v                v
 PostgreSQL      Event consumers
  (per-svc DB)    (user, websocket)
```

## Service Interaction Map

### Synchronous (gRPC)

| Caller       | Callee       | Purpose                                  |
|--------------|--------------|------------------------------------------|
| Gateway      | Auth         | Register, login, logout, refresh tokens  |
| Gateway      | User         | Get/update user profiles                 |
| Gateway      | Room         | CRUD rooms, manage membership            |
| Gateway      | Message      | Send messages, get history               |
| Message      | Room         | Verify room membership (IsMember)        |

### Asynchronous (NATS JetStream)

| Publisher    | Event                      | Consumer(s)     | Purpose                          |
|--------------|----------------------------|-----------------|----------------------------------|
| Auth         | `auth.user.registered`     | User            | Create user profile              |
| Room         | `room.created`             | WebSocket       | Notify connected clients         |
| Room         | `room.membership.changed`  | WebSocket       | Notify room about join/leave     |
| Room         | `room.updated`             | WebSocket       | Notify room about metadata change|
| Room         | `room.deleted`             | WebSocket       | Notify room about deletion       |
| Message      | `message.created`          | WebSocket       | Deliver new message in real-time |

## Request Lifecycle

### HTTP Request (e.g., send a message)

```
1. Client sends POST /rooms/{id}/messages to api.localhost
2. nginx routes to Gateway service (:8080)
3. Gateway middleware:
   a. Recoverer (panic recovery)
   b. JWTAuth -- extracts user_id from access_token cookie
   c. UserRateLimiter -- checks Valkey counter for rate limit
4. Gateway handler:
   a. Reads JSON body, unmarshals to protobuf
   b. Injects room_id (from URL) and sender_id (from JWT)
   c. Calls Message service via gRPC
5. Message service:
   a. Handler validates request with protovalidate
   b. Service layer calls Room service (gRPC) to verify membership
   c. Repository stores message in PostgreSQL
   d. Publisher sends message.created to NATS JetStream
6. Gateway marshals gRPC response to JSON, returns to client
```

### WebSocket Connection

```
1. Client calls POST /auth/ws-ticket (authenticated)
   -> Gateway creates UUID ticket in Valkey (TTL 30s)
   -> Returns { "ticket": "..." }

2. Client connects to ws://ws.localhost/ws?ticket=<ticket>&room_id=<uuid>
   -> nginx routes to WebSocket service (:8081) via forward auth (auth_request)
   -> WebSocket handler validates ticket via Valkey (GETDEL, single-use)
   -> WebSocket connection is established
   -> If room_id provided, client is auto-subscribed to that room

3. Client sends subscribe/unsubscribe messages to manage room subscriptions
4. NATS events (message.created, room.updated, etc.) are delivered to the Hub
5. Hub broadcasts to all connections subscribed to the affected room
```

### Event-Driven Flow (registration)

```
1. Client registers via POST /auth/register
2. Auth service creates account in PostgreSQL
3. Auth service publishes auth.user.registered to NATS (AUTH stream)
4. User service's consumer picks up the event
5. User service creates a profile with auto-generated username
   (adjective-noun-number pattern, e.g., "bright-falcon-42")
```

## NATS JetStream Configuration

### Streams

Each producing service provisions its own stream on startup.

| Stream    | Subjects    | Retention      | Storage | Replicas |
|-----------|-------------|----------------|---------|----------|
| `AUTH`    | `auth.>`    | LimitsPolicy   | File    | 1        |
| `ROOM`    | `room.>`    | LimitsPolicy   | File    | 1        |
| `MESSAGE` | `message.>` | LimitsPolicy   | File    | 1        |

### Consumers

Consumers are provisioned with retry logic on startup. Each consumer is durable (survives restarts).

| Consumer                     | Stream  | Filter Subject             | Provisioned by |
|------------------------------|---------|----------------------------|----------------|
| `user-svc-auth-registered`   | AUTH    | `auth.user.registered`     | User service   |
| `ws-svc-message-created`     | MESSAGE | `message.created`          | WebSocket      |
| `ws-svc-room-membership`     | ROOM    | `room.membership.changed`  | WebSocket      |
| `ws-svc-room-updated`        | ROOM    | `room.updated`             | WebSocket      |

Consumer settings (configurable per service):
- `max_deliver`: maximum delivery attempts before the message is discarded
- `ack_wait`: time to wait for acknowledgment before redelivery

## WebSocket Hub Architecture

The WebSocket service uses a single-goroutine Hub pattern for thread-safe connection management:

```
Hub (single goroutine via select loop)
  |
  |-- rooms: map[roomID] -> map[userID] -> []Conn
  |-- register channel   <- subscribe requests
  |-- unregister channel <- unsubscribe requests
  |-- disconnect channel <- full disconnects
  |-- broadcast channel  <- messages from NATS
  |-- roomCount channel  <- count queries
```

This eliminates the need for mutexes. All state mutations happen in the Hub's `Run()` loop.

**Write pump / Read pump / Ping pump:** Each WebSocket connection spawns three goroutines:
- **readPump** -- reads client messages, dispatches subscribe/unsubscribe/ping
- **writePump** -- drains the outbound channel, writes to the WebSocket
- **pingPump** -- sends periodic WebSocket pings to detect dead connections

When any pump encounters an error, it cancels the shared context, triggering cleanup in the others.

## Authentication Architecture

### Cookie-Based JWT

```
Register/Login
  |
  v
Gateway sets HttpOnly cookies:
  - access_token  (JWT, 15min TTL, Path=/)
  - refresh_token (opaque, 30-day TTL, Path=/auth/session)
  |
  v
Protected requests:
  - JWTAuth middleware reads access_token cookie
  - Parses JWT, validates HMAC signature with JWT_SECRET
  - Extracts user_id from "sub" claim
  - Injects into request context
  |
  v
Token refresh:
  - POST /auth/session/refresh (uses refresh_token cookie)
  - Returns new pair of tokens as cookies
```

### WebSocket Ticket Auth

WebSocket connections cannot send cookies in the initial handshake across all browsers reliably. Efir uses a ticket-based approach:

1. Authenticated client requests a one-time ticket via REST
2. Ticket (UUID) is stored in Valkey with short TTL
3. Client passes ticket as query parameter in WebSocket URL
4. WebSocket service validates and consumes ticket via GETDEL (atomic get+delete)

## Shared Packages

The `services/shared/` module provides common functionality:

| Package            | Description                                         |
|--------------------|-----------------------------------------------------|
| `gen/`             | Generated protobuf + gRPC Go code                   |
| `pkg/config`       | Environment enum (`development`/`production`)        |
| `pkg/errors`       | Domain error codes with gRPC/HTTP status mapping     |
| `pkg/healthcheck`  | Liveness/readiness probe HTTP handlers               |
| `pkg/logger`       | Structured slog logger with level parsing            |
| `pkg/mapper`       | Generic mapping utilities                            |
| `pkg/middleware`    | gRPC interceptors (validation via protovalidate)     |
| `pkg/nats`         | NATS JetStream helpers (connect, stream/consumer provisioning) |
| `pkg/testutil`     | Test helpers (Postgres containers, NATS, Valkey, fixtures) |
| `pkg/valkey`       | Valkey key builders and Lua scripts                  |
| `pkg/username.go`  | Random username generator (adjective-noun-number)    |

## Infrastructure

### nginx

nginx serves as the edge router with the following configuration:

- **Entrypoint:** `:80` (HTTP), `:443` (HTTPS, production)
- **Routing:** Host-based (`api.localhost` → Gateway, `ws.localhost` → WebSocket)
- **CORS:** Configured via `map` directive, allows `http://localhost:5173` in development
- **Rate limiting:** `limit_req_zone` per IP, 100 req/min with burst of 50
- **Forward auth:** `auth_request` directive validates WebSocket tickets via Gateway before proxying
- **DNS resolution:** Docker's internal resolver (`127.0.0.11`) for dynamic upstream resolution

### Docker Compose (Modular)

The compose setup is split into modular files for flexibility:

| File                              | Contains                         |
|-----------------------------------|----------------------------------|
| `docker-compose.network.yml`     | External network definition      |
| `docker-compose.infra.yml`       | Postgres, NATS, Valkey, nginx    |
| `docker-compose.services.yml`    | All application services         |
| `docker-compose.sidecar.yml`     | Sidecar PEP containers           |
| `docker-compose.observability.yml`| Grafana, Loki, Tempo            |
| `docker-compose.dev.yml`         | Local dev overrides (ports, etc.)|

The root `docker-compose.yml` includes all of these. Use `task docker:infra:up` to start only infrastructure for local development.

### PostgreSQL

A single Postgres instance with per-service databases, created by init scripts in `infra/postgres/init/`:

- `efir_auth` -- accounts table
- `efir_user` -- users table
- `efir_room` -- rooms and room_members tables
- `efir_message` -- messages table

Each service has its own database user with access only to its own database.

## Testing Strategy

- **Unit tests** -- service and handler layers, using mockery-generated mocks
- **Repository tests** -- run against real PostgreSQL via testcontainers
- **NATS tests** -- run against embedded or containerized NATS
- **Handler tests** -- test HTTP/gRPC handlers with mocked service layer
- **Test utilities** -- `shared/pkg/testutil` provides helpers for spinning up Postgres, NATS, Valkey containers and generating test fixtures

Run all tests:
```bash
task go:test
```
