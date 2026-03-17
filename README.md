# 🛰 Efir

**What is Efir?**
It's a simple, honest messenger. I'm building this because I'm tired of "everything apps" that try to be a store, a crypto wallet, and a social network all at once. Efir is just about talking to people without extra noise and stuff they usually don't expect to appear in a communiation tool.

**The Philosophy**
I am a firm believer in the **Unix philosophy**: a tool should do one thing and do it well. For me, Efir's only job is to let people communicate — via text or voice — as reliably as possible.

I also believe in **ownership**. Efir follows a self-hosting model: you can use my instance or spin up your own on your hardware with a single command. In a world of digital blocks and censorship, having your own communication node isn't just a feature — it's a right.

---

### Tech Stack

- **Go 1.25**: Backend services.
- **NATS JetStream**: Messaging and events.
- **Valkey**: Caching and rate limiting.
- **PostgreSQL**: Persistent storage (one database per service).
- **gRPC**: Internal communication.
- **Protobuf**: Contract definition and validation.
- **Traefik**: API gateway and routing.
- **Grafana, Loki & Tempo**: Observability.

---

### Architecture

Efir follows a **clean architecture** pattern with clear separation:

```
handler → service → repository
```

Each service has:

- `cmd/main.go` — Entry point
- `internal/config/` — Configuration via environment variables
- `internal/handler/` — gRPC handlers
- `internal/service/` — Business logic
- `internal/repository/` — Database access
- `migrations/` — Database migrations (goose)

Services communicate via:

- **gRPC** — Synchronous calls between services
- **NATS JetStream** — Asynchronous events (e.g., user registration → creates profile)
- **Valkey** — Caching, session management, rate limiting

---

### Services

| Service   | Port  | Description                                           |
| --------- | ----- | ----------------------------------------------------- |
| gateway   | 8080  | HTTP API gateway, JWT auth, rate limiting             |
| auth      | 50051 | Authentication, JWT tokens, refresh tokens            |
| user      | 50052 | User profiles, username, avatar, bio                  |
| room      | 50053 | Chat rooms, membership management                     |
| message   | 50054 | Messages, history, pagination                         |
| websocket | 8081  | Real-time message delivery via WebSocket              |
| sidecar   | 50052 | PEP (Policy Enforcement Point) for traffic validation |

---

### Roadmap

- [x] **Module 0: Foundation** — Architecture, CI/CD, and infra are ready.
- [x] **Module 1: MVP** — Auth, user profiles, and basic realtime chat.
- [x] Auth Service (registration, login, JWT tokens)
- [x] User Service (profiles, username, avatar, bio)
- [ ] Room Service (chat rooms, membership)
- [ ] Message Service (send/receive messages)
- [ ] WebSocket Connector (real-time delivery)
- [ ] Gateway (HTTP API, routing, auth)
- [ ] **Module 2: Scale & Security** — Sidecar PEP for traffic validation and horizontal scaling.
- [ ] **Module 3: Features** — Presence status, media handling, notifications, and global search.

---

### Quick Start

If you have **Docker** and **Task** installed:

```bash
# Prepare the network and environment
task setup

# Spin up all services and infrastructure
task up

# Or just start infrastructure for local development
task docker:infra:up
```

---

### Development

```bash
# Run a specific service locally
task go:run SERVICE=auth

# Run tests
task go:test

# Run linter
task go:lint

# Generate mocks
task go:generate

# Work with migrations
task migrate:create SERVICE=user NAME=init
task migrate:up SERVICE=user
task migrate:status SERVICE=user
```

---

### Project Structure

```
efir/
├── services/          # All Go services
│   ├── auth/          # Authentication service
│   ├── user/          # User profile service
│   ├── room/          # Room management service
│   ├── message/       # Message storage service
│   ├── websocket/     # WebSocket connector
│   ├── gateway/       # API gateway
│   ├── sidecar/       # Policy enforcement proxy
│   └── shared/        # Common code (logger, middleware, errors)
├── proto/             # Protocol Buffer definitions
│   └── efir/
├── infra/             # Infrastructure configuration
│   ├── postgres/      # Database init scripts
│   ├── nats/          # NATS configuration
│   ├── valkey/        # Valkey configuration
│   └── traefik/       # Router configuration
├── deploy/            # Docker Compose files
├── tasks/             # Task automation scripts
└── docs/              # Documentation and ADRs
```
