# Gateway Service

HTTP-to-gRPC proxy that handles client requests, JWT authentication, and rate limiting.

## Quick Start

```bash
# Run the service
task gateway:run

# Run tests
task gateway:test
```

## Architecture

```
services/gateway/
├── cmd/main.go                    # Entry point
├── internal/
│   ├── config/                   # Configuration
│   ├── handler/                  # HTTP handlers
│   │   ├── auth/                 # Auth endpoints
│   │   ├── user/                 # User endpoints
│   │   ├── room/                 # Room endpoints
│   │   ├── message/              # Message endpoints
│   │   └── wsauth/               # WebSocket auth endpoints
│   └── middleware/               # JWT auth, rate limiting
└── config.yaml                   # Example config
```

## Technology Stack

- **Language**: Go 1.25
- **HTTP Router**: Chi
- **Authentication**: JWT (Bearer tokens)
- **Cache**: Valkey (rate limiting, WebSocket tickets)
- **Protocol**: HTTP/REST → gRPC proxy

## Request Flow

```
Client Request
     │
     ▼
IP Rate Limiter (Valkey)
     │
     ▼
JWT Auth Middleware
     │
     ▼
User Rate Limiter (Valkey)
     │
     ▼
Handler (gRPC proxy)
     │
     ▼
Backend Services (Auth/User/Room/Message)
```

## Endpoints

See [docs/api.md](docs/api.md) for detailed endpoint documentation.

### Public Endpoints (IP Rate Limited)

| Method | Path             | Description          |
| ------ | ---------------- | -------------------- |
| `POST` | `/auth/register` | Register new user    |
| `POST` | `/auth/login`    | Login and get tokens |
| `POST` | `/auth/logout`   | Logout               |
| `POST` | `/auth/refresh`  | Refresh access token |

### Protected Endpoints (JWT + User Rate Limited)

| Method   | Path                           | Description                      |
| -------- | ------------------------------ | -------------------------------- |
| `GET`    | `/users/me`                    | Get current user profile         |
| `GET`    | `/users/{id}`                  | Get user by ID                   |
| `PATCH`  | `/users/me`                    | Update current user profile      |
| `POST`   | `/rooms`                       | Create a new room                |
| `GET`    | `/rooms/{id}`                  | Get room by ID                   |
| `PATCH`  | `/rooms/{id}`                  | Update room                      |
| `DELETE` | `/rooms/{id}`                  | Delete room                      |
| `POST`   | `/rooms/{id}/members`          | Add member to room               |
| `DELETE` | `/rooms/{id}/members/{userId}` | Remove member from room          |
| `POST`   | `/rooms/{id}/messages`         | Send a message                   |
| `GET`    | `/rooms/{id}/messages`         | Get messages (cursor pagination) |
| `POST`   | `/auth/ws-ticket`              | Create WebSocket auth ticket     |

### Health Endpoints

| Method | Path      | Description     |
| ------ | --------- | --------------- |
| `GET`  | `/health` | Liveness probe  |
| `GET`  | `/ready`  | Readiness probe |

## Configuration

See [docs/config.md](docs/config.md) for configuration reference.
