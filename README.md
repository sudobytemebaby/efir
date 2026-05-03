# Efir 🛰️

A self-hostable messenger built with Go microservices. One job: let people communicate via text and voice, reliably.

Efir follows the Unix philosophy -- do one thing and do it well. No built-in store, no crypto wallet, no social feed. Just messaging and calls. In future.

## Table of Contents

- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Services](#services)
- [Getting Started](#getting-started)
- [Development](#development)
- [API Reference](#api-reference)
- [WebSocket Protocol](#websocket-protocol)
- [Event-Driven Communication](#event-driven-communication)
- [Database Schemas](#database-schemas)
- [Environment Variables](#environment-variables)
- [CI/CD](#cicd)
- [Project Structure](#project-structure)
- [ADRs](#architecture-decision-records)
- [Roadmap](#roadmap)

## Architecture

```
                          +-----------+
                          |   nginx   |  :80
                          |  (reverse |
                          |   proxy)  |
                          +-----+-----+
                                |
               +----------------+----------------+
               |                                 |
        api.localhost                      ws.localhost
               |                                 |
        +------+------+                  +-------+-------+
        |   Gateway   | :8080            |   WebSocket   | :8081
        |   (HTTP)    |                  |  (Connector)  |
        +------+------+                  +-------+-------+
               |                                 |
    +----------+----------+                      |
    |     |      |        |                      |
+---+--+--+--+--+---+----+-----+          +------+------+
| Auth | User | Room | Message |          | NATS        |
|:50051|:50052|:50053| :50054  |--------> | JetStream   |
+------+-----+------+----------+          +-------------+
    |     |      |        |
    v     v      v        v
  +-------------------------+            +----------+
  |      PostgreSQL         |            |  Valkey  |
  | (one DB per service)    |            | (cache)  |
  +-------------------------+            +----------+
```

**Request flow:** Client -> nginx -> Gateway (HTTP/JSON) -> Service (gRPC) -> PostgreSQL

**Real-time flow:** Service -> NATS JetStream -> WebSocket Connector -> Client (WebSocket)

Each service follows clean architecture: `handler -> service -> repository`.

## Tech Stack

| Component     | Technology           | Purpose                               |
| ------------- | -------------------- | ------------------------------------- |
| Language      | Go 1.25              | All backend services                  |
| Messaging     | NATS JetStream       | Async events between services         |
| Cache         | Valkey 9             | Session management, rate limiting     |
| Database      | PostgreSQL 18        | Persistent storage (DB per service)   |
| Internal RPC  | gRPC + Protobuf      | Synchronous service-to-service calls  |
| API Gateway   | Chi (Go)             | HTTP routing, JWT auth, rate limiting |
| Reverse Proxy | nginx 1.27           | Edge routing, CORS, forward auth      |
| Validation    | protovalidate (buf)  | Request validation at proto level     |
| Migrations    | goose                | Database schema migrations            |
| Mocks         | mockery              | Test mock generation                  |
| Linter        | golangci-lint v2     | Static analysis                       |
| Task Runner   | Task (taskfile.dev)  | Build automation                      |
| Observability | Grafana, Loki, Tempo | Logs, traces, dashboards              |

## Services

| Service       | Port  | Transport | Description                                    |
| ------------- | ----- | --------- | ---------------------------------------------- |
| **gateway**   | 8080  | HTTP      | Public API, JWT cookie auth, rate limiting     |
| **auth**      | 50051 | gRPC      | Registration, login, JWT tokens, refresh flow  |
| **user**      | 50052 | gRPC      | User profiles (username, display name, avatar) |
| **room**      | 50053 | gRPC      | Chat rooms (direct/group), membership          |
| **message**   | 50054 | gRPC      | Message storage, history, cursor pagination    |
| **websocket** | 8081  | WS        | Real-time delivery via WebSocket               |
| **sidecar**   | --    | --        | PEP (Policy Enforcement Point) for traffic     |
| **shared**    | --    | --        | Common libraries (logger, errors, middleware)  |

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [Task](https://taskfile.dev/) (`go install github.com/go-task/task/v3/cmd/task@latest` or `brew install go-task`)
- [Go 1.25+](https://go.dev/dl/) (for local development)
- [buf](https://buf.build/docs/installation) (for proto generation)

### Quick Start

```bash
# 1. Clone and initialize
git clone https://github.com/sudobytemebaby/efir.git
cd efir
task setup    # creates Docker network, copies .env.example -> .env, configures git hooks, installs tools

# 2. Configure environment
#    Edit .env and fill in secrets (JWT_SECRET, database passwords, etc.)

# 3. Start everything
task docker:up

# 4. Apply database migrations
task migrate:up
```

The API will be available at `http://api.localhost` and WebSocket at `ws://ws.localhost/ws`.

### Local Development

```bash
# Start only infrastructure (Postgres, NATS, Valkey, nginx)
task docker:infra:up

# Run a specific service locally
task go:run SERVICE=auth

# Run all tests
task go:test

# Run linter
task go:lint
```

See [docs/taskfile.md](docs/taskfile.md) for the full task reference.

## API Reference

All endpoints are served through the **Gateway** service. Requests and responses use JSON. Authentication uses HttpOnly cookies (`access_token` / `refresh_token`) set automatically on login/register.

### Authentication

| Method | Path                    | Auth   | Description            |
| ------ | ----------------------- | ------ | ---------------------- |
| POST   | `/auth/register`        | Public | Create new account     |
| POST   | `/auth/login`           | Public | Sign in                |
| POST   | `/auth/session/refresh` | Cookie | Refresh access token   |
| POST   | `/auth/session/logout`  | Cookie | Invalidate session     |
| GET    | `/auth/me`              | JWT    | Get current user ID    |
| POST   | `/auth/ws-ticket`       | JWT    | Get one-time WS ticket |

#### POST /auth/register

```json
// Request
{ "email": "user@example.com", "password": "securepass" }

// Response 200
{ "user_id": "550e8400-e29b-41d4-a716-446655440000" }

// Set-Cookie: access_token=<jwt>; HttpOnly; Path=/
// Set-Cookie: refresh_token=<token>; HttpOnly; Path=/auth/session
```

#### POST /auth/login

```json
// Request
{ "email": "user@example.com", "password": "securepass" }

// Response 200
{ "user_id": "550e8400-e29b-41d4-a716-446655440000" }
// + Set-Cookie headers (same as register)
```

#### POST /auth/session/refresh

No request body. Uses `refresh_token` cookie. Returns `204 No Content` with new cookies.

#### POST /auth/session/logout

No request body. Uses `refresh_token` cookie. Returns `204 No Content`. Clears cookies.

#### GET /auth/me

Returns `{ "user_id": "..." }` from JWT claims.

#### POST /auth/ws-ticket

Returns `{ "ticket": "..." }` -- a one-time ticket (UUID) for WebSocket authentication. TTL is configurable (default 30s). The ticket is stored in Valkey and consumed on use (GETDEL).

### Users

All endpoints require JWT authentication.

| Method | Path          | Description              |
| ------ | ------------- | ------------------------ |
| GET    | `/users/me`   | Get current user profile |
| GET    | `/users/{id}` | Get user by ID           |
| PATCH  | `/users/me`   | Update current user      |

#### GET /users/me, GET /users/{id}

```json
// Response 200
{
  "user_id": "...",
  "username": "bright-falcon-42",
  "display_name": "John",
  "avatar_url": "https://...",
  "bio": "Hello!",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

Usernames are auto-generated on registration (adjective-noun-number pattern).

#### PATCH /users/me

```json
// Request (all fields optional)
{
  "display_name": "New Name",
  "avatar_url": "https://...",
  "bio": "Updated bio"
}

// Response 200 -- full user object
```

### Rooms

All endpoints require JWT authentication.

| Method | Path                           | Description       |
| ------ | ------------------------------ | ----------------- |
| GET    | `/rooms`                       | List user's rooms |
| POST   | `/rooms`                       | Create room       |
| GET    | `/rooms/{id}`                  | Get room details  |
| PATCH  | `/rooms/{id}`                  | Update room       |
| DELETE | `/rooms/{id}`                  | Delete room       |
| POST   | `/rooms/{id}/members`          | Add member        |
| DELETE | `/rooms/{id}/members/{userId}` | Remove member     |

#### POST /rooms

```json
// Request
{
  "name": "Project Chat",
  "type": "ROOM_TYPE_GROUP",       // or "ROOM_TYPE_DIRECT"
  "participant_id": "..."          // required for direct rooms
}

// Response 201
{
  "room_id": "...",
  "name": "Project Chat",
  "type": "ROOM_TYPE_GROUP",
  "created_by": "...",
  "created_at": "...",
  "updated_at": "..."
}
```

#### POST /rooms/{id}/members

```json
// Request
{ "user_id": "target-user-uuid" }

// Response 204 No Content
```

### Messages

All endpoints require JWT authentication.

| Method | Path                   | Description         |
| ------ | ---------------------- | ------------------- |
| POST   | `/rooms/{id}/messages` | Send message        |
| GET    | `/rooms/{id}/messages` | Get message history |

#### POST /rooms/{id}/messages

```json
// Request (text message example)
{
  "type": "MESSAGE_TYPE_TEXT",
  "text": { "text": "Hello, world!" },
  "reply_to_id": "optional-message-uuid"
}

// Response 201
{
  "message_id": "...",
  "room_id": "...",
  "sender_id": "...",
  "type": "MESSAGE_TYPE_TEXT",
  "is_deleted": false,
  "created_at": "...",
  "updated_at": "...",
  "text": { "text": "Hello, world!" }
}
```

Supported message types: `TEXT`, `IMAGE`, `VIDEO`, `VIDEO_NOTE`, `VOICE`, `AUDIO`, `FILE`, `STICKER`, `VIDEO_STICKER`, `EVENT`. Each type has a corresponding content field (see [proto definitions](proto/efir/message/message.proto)).

#### GET /rooms/{id}/messages

Query parameters:

- `cursor` (optional) -- opaque pagination cursor
- `limit` (optional, default: 50, max: 100)

```json
// Response 200
{
  "messages": [ ... ],
  "next_cursor": "base64-encoded-cursor"
}
```

### Health Checks

| Method | Path      | Auth   | Description     |
| ------ | --------- | ------ | --------------- |
| GET    | `/health` | Public | Liveness probe  |
| GET    | `/ready`  | Public | Readiness probe |

### Error Format

All errors follow a consistent format:

```json
{
  "error": "human-readable message",
  "code": "NOT_FOUND"
}
```

Error codes: `NOT_FOUND`, `ALREADY_EXISTS`, `PERMISSION_DENIED`, `UNAUTHENTICATED`, `INVALID_ARGUMENT`, `UNAVAILABLE`, `INTERNAL`, `RATE_LIMITED`.

### Rate Limiting

Authenticated endpoints are rate-limited per user. Default: **100 requests per 60 seconds**. Configurable via `RATE_LIMIT_REQUESTS` and `RATE_LIMIT_WINDOW`. Implemented using Valkey with a Lua script for atomic increment-with-expiry.

## WebSocket Protocol

Connect to `ws://ws.localhost/ws?ticket=<ticket>&room_id=<optional-room-uuid>`.

### Authentication Flow

1. Client calls `POST /auth/ws-ticket` (requires JWT cookie) to get a one-time ticket
2. Client connects to WebSocket with `?ticket=<ticket>` query parameter
3. Ticket is validated and consumed (single use)

### Message Envelope

All WebSocket messages use a JSON envelope:

```json
{
  "type": "message.created",
  "payload": { ... }
}
```

### Client -> Server Messages

| Type          | Payload                | Description                  |
| ------------- | ---------------------- | ---------------------------- |
| `subscribe`   | `{ "room_id": "..." }` | Subscribe to room events     |
| `unsubscribe` | `{ "room_id": "..." }` | Unsubscribe from room events |
| `ping`        | --                     | Keep-alive ping              |

### Server -> Client Messages

| Type                      | Payload                                             | Description               |
| ------------------------- | --------------------------------------------------- | ------------------------- |
| `message.created`         | `{ "message_id", "room_id", "user_id", "content" }` | New message in a room     |
| `room.membership.changed` | `{ "room_id", "user_id", "joined" }`                | Member joined/left a room |
| `room.updated`            | `{ "room_id", "updated_by" }`                       | Room metadata changed     |
| `pong`                    | --                                                  | Ping response             |
| `error`                   | `{ "code", "message" }`                             | Error notification        |

## Event-Driven Communication

Services communicate asynchronously via NATS JetStream. Each stream uses `LimitsPolicy` retention with file storage.

### NATS Streams

| Stream    | Subjects    | Producers       |
| --------- | ----------- | --------------- |
| `AUTH`    | `auth.>`    | Auth service    |
| `ROOM`    | `room.>`    | Room service    |
| `MESSAGE` | `message.>` | Message service |

### Event Flow

```
Auth Service                    User Service
     |                               |
     |-- auth.user.registered ------>|  (creates user profile with random username)
     |                               |

Room Service                    WebSocket Connector
     |                               |
     |-- room.created -------------> |  (notifies connected clients)
     |-- room.membership.changed --> |  (member added/removed)
     |-- room.updated -------------> |  (room name changed)
     |-- room.deleted -------------> |  (room deleted)
     |                               |

Message Service                 WebSocket Connector
     |                               |
     |-- message.created ----------> |  (broadcasts to room subscribers)
     |                               |
```

### Consumer Groups

| Consumer                   | Stream  | Subject                   | Service   |
| -------------------------- | ------- | ------------------------- | --------- |
| `user-svc-auth-registered` | AUTH    | `auth.user.registered`    | User      |
| `ws-svc-message-created`   | MESSAGE | `message.created`         | WebSocket |
| `ws-svc-room-membership`   | ROOM    | `room.membership.changed` | WebSocket |
| `ws-svc-room-updated`      | ROOM    | `room.updated`            | WebSocket |

## Database Schemas

Each service owns its database. Migrations are managed with [goose](https://github.com/pressly/goose).

### Auth (`efir_auth`)

```sql
accounts (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
)
```

Refresh tokens are stored in Valkey with TTL.

### User (`efir_user`)

```sql
users (
  id           UUID PRIMARY KEY,           -- same as auth account ID
  username     VARCHAR(50) NOT NULL UNIQUE, -- auto-generated
  display_name VARCHAR(100) NOT NULL,
  avatar_url   TEXT,
  bio          TEXT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
)
```

### Room (`efir_room`)

```sql
-- Types: 'direct', 'group'
rooms (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       VARCHAR(100) NOT NULL,
  type       room_type NOT NULL,
  created_by UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)

-- Roles: 'owner', 'member'
room_members (
  room_id   UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
  user_id   UUID NOT NULL,
  role      member_role NOT NULL DEFAULT 'member',
  joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (room_id, user_id)
)
```

### Message (`efir_message`)

```sql
-- Types: 'text','image','video','video_note','voice','audio','file','sticker','video_sticker','event'
messages (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  room_id     UUID NOT NULL,
  sender_id   UUID NOT NULL,
  type        message_type NOT NULL,
  content     JSONB NOT NULL,               -- polymorphic content
  reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  deleted_at  TIMESTAMPTZ,
  edited_at   TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
)
```

## Environment Variables

Copy `.env.example` to `.env` and fill in the values:

| Variable               | Required | Description                            |
| ---------------------- | -------- | -------------------------------------- |
| `ENV`                  | Yes      | `development` or `production`          |
| `LOG_LEVEL`            | No       | `debug`, `info`, `warn`, `error`       |
| `JWT_SECRET`           | Yes      | HMAC signing key for JWTs              |
| `JWT_ACCESS_TTL`       | No       | Access token lifetime (default: 15m)   |
| `JWT_REFRESH_TTL`      | No       | Refresh token lifetime (default: 168h) |
| `POSTGRES_USER`        | Yes      | PostgreSQL superuser                   |
| `POSTGRES_PASSWORD`    | Yes      | PostgreSQL superuser password          |
| `POSTGRES_AUTH_DSN`    | Yes      | Auth service DB connection string      |
| `POSTGRES_USER_DSN`    | Yes      | User service DB connection string      |
| `POSTGRES_ROOM_DSN`    | Yes      | Room service DB connection string      |
| `POSTGRES_MESSAGE_DSN` | Yes      | Message service DB connection string   |
| `VALKEY_ADDR`          | No       | Valkey address (default: valkey:6379)  |
| `VALKEY_PASSWORD`      | Yes      | Valkey password                        |
| `NATS_URL`             | No       | NATS URL (default: nats://nats:4222)   |
| `NATS_USER`            | No       | NATS username                          |
| `NATS_PASSWORD`        | No       | NATS password                          |
| `RATE_LIMIT_REQUESTS`  | No       | Requests per window (default: 100)     |
| `RATE_LIMIT_WINDOW`    | No       | Rate limit window (default: 60s)       |

## CI/CD

GitHub Actions pipeline runs on push to `main` and on pull requests:

1. **Validate** -- commit message format (Conventional Commits), file structure
2. **Proto** -- lint protobuf definitions with `buf lint`
3. **Go** -- lint (`golangci-lint`), test (`go test`), verify generated code
4. **Build** -- Docker image build for all services

See `.github/workflows/ci.yml` for details.

## Project Structure

```
efir/
|-- services/                # All Go microservices
|   |-- auth/                # Authentication (register, login, JWT, refresh)
|   |-- user/                # User profiles (username, avatar, bio)
|   |-- room/                # Rooms and membership (direct, group)
|   |-- message/             # Messages (send, history, pagination)
|   |-- websocket/           # Real-time WebSocket connector
|   |-- gateway/             # HTTP API gateway (Chi router)
|   |-- sidecar/             # Policy enforcement proxy
|   +-- shared/              # Common packages
|       |-- gen/             #   Generated protobuf Go code
|       +-- pkg/             #   Shared libraries (logger, errors, nats, valkey, etc.)
|-- proto/                   # Protobuf service definitions
|   +-- efir/
|       |-- auth/auth.proto
|       |-- user/user.proto
|       |-- room/room.proto
|       +-- message/message.proto
|-- infra/                   # Infrastructure configuration
|   |-- postgres/            #   DB init scripts (per-service databases)
|   |-- nats/                #   NATS server config
|   |-- valkey/              #   Valkey config
|   +-- nginx/               #   nginx routing, CORS, forward auth
|-- deploy/
|   +-- compose/             # Modular Docker Compose files
|       |-- docker-compose.infra.yml
|       |-- docker-compose.services.yml
|       |-- docker-compose.sidecar.yml
|       |-- docker-compose.observability.yml
|       +-- docker-compose.dev.yml
|-- tasks/                   # Task runner scripts
|-- docs/                    # Documentation
|   |-- adr/                 #   Architecture Decision Records
|   |-- git-workflow.md      #   Branch naming, commit conventions
|   +-- taskfile.md          #   Full task reference
+-- .github/workflows/       # CI pipeline
```

### Service Internal Layout

Each service follows a consistent layout:

```
services/<name>/
|-- cmd/main.go              # Entrypoint, wiring, graceful shutdown
|-- internal/
|   |-- config/              # Config struct + env/yaml loading (cleanenv)
|   |-- handler/             # gRPC handlers (or HTTP for gateway/websocket)
|   |-- service/             # Business logic + interfaces
|   |   +-- mocks/           # Generated mocks (mockery)
|   |-- repository/          # Database access (pgx)
|   |   +-- mocks/           # Generated mocks
|   +-- nats/                # NATS streams, publishers, consumers
|-- migrations/              # SQL migrations (goose)
+-- Dockerfile
```

## Architecture Decision Records

Design decisions are documented in `docs/adr/`:

| ADR                                               | Title                                  |
| ------------------------------------------------- | -------------------------------------- |
| [001](docs/adr/001-monorepo.md)                   | Monorepo Structure                     |
| [002](docs/adr/002-sidecar-pep.md)                | Sidecar PEP (Policy Enforcement Point) |
| [003](docs/adr/003-modular-compose.md)            | Modular Docker Compose                 |
| [004](docs/adr/004-go-workspace.md)               | Go Workspace                           |
| [005](docs/adr/005-git-hooks-without-husky.md)    | Git Hooks Without Husky                |
| [006](docs/adr/006-ci-pipeline.md)                | GitHub Actions CI/CD Pipeline          |
| [007](docs/adr/007-decoupled-config.md)           | Decoupled Service Configuration        |
| [008](docs/adr/008-rate-limiting.md)              | Rate Limiting Strategy                 |
| [009](docs/adr/009-proto-validation.md)           | Protobuf Validation with protovalidate |
| [010](docs/adr/010-user-service.md)               | User Service Implementation            |
| [011](docs/adr/011-room-membership-events.md)     | Room Membership Event Publishing       |
| [012](docs/adr/012-message-schema.md)             | Message Schema and Service Design      |
| [013](docs/adr/013-websocket-connector.md)        | WebSocket Connector                    |
| [014](docs/adr/014-gateway-service.md)            | Gateway Service Implementation         |
| [015](docs/adr/015-random-username-generation.md) | Random Username Generation             |
| [016](docs/adr/016-nginx-reverse-proxy.md)        | nginx as Reverse Proxy                 |

## Roadmap

- [x] **Module 0: Foundation** -- Architecture, CI/CD, infrastructure
- [x] **Module 1: MVP** -- Core messaging functionality
  - [x] Auth Service (registration, login, JWT, refresh tokens)
  - [x] User Service (profiles, auto-generated usernames)
  - [x] Room Service (direct/group rooms, membership)
  - [x] Message Service (send, history, cursor pagination)
  - [x] WebSocket Connector (real-time delivery)
  - [x] Gateway (HTTP API, JWT cookie auth, rate limiting)
- [ ] **Module 2: Scale & Security** -- Sidecar PEP, horizontal scaling
- [ ] **Module 3: Features** -- Presence, media, notifications, search

## License

This project is not yet licensed. All rights reserved.
