# Gateway Service Overview

## Responsibilities

The Gateway Service is responsible for:

- Accepting HTTP/REST requests from clients
- JWT token validation and user authentication
- Rate limiting (IP-based and user-based)
- Proxying requests to backend gRPC services
- WebSocket ticket-based authentication
- Request/response format translation (JSON ↔ Protobuf)

## Architecture

### HTTP-to-gRPC Proxy

The gateway translates REST-style HTTP requests into gRPC calls to backend services:

```
services/gateway/
├── handler/     -> HTTP handlers (auth, user, room, message, wsauth)
├── middleware/  -> JWT auth, rate limiting
└── config/      -> Configuration
```

### Backend Services

The gateway proxies to 4 gRPC backend services:

| Service | Port  | Purpose                                    |
| ------- | ----- | ------------------------------------------ |
| Auth    | 50051 | User registration, login, token management |
| User    | 50052 | User profile management                    |
| Room    | 50053 | Room/chat management                       |
| Message | 50054 | Message handling                           |

### Rate Limiting

Two-level rate limiting using Valkey:

**IP-based Rate Limiting:**

- Applied to: All public endpoints
- Key pattern: `gateway:ratelimit:ip:<ip>:<window>`
- Default: 100 requests per 60 seconds

**User-based Rate Limiting:**

- Applied to: All protected endpoints (after JWT validation)
- Key pattern: `gateway:ratelimit:user:<user_id>:<window>`
- Same limits as IP-based

### WebSocket Authentication

The gateway provides ticket-based authentication for WebSocket connections:

1. Authenticated user calls `POST /auth/ws-ticket` to get a one-time ticket
2. Ticket is stored in Valkey with configurable TTL
3. WebSocket service validates ticket via `GET /auth/validate`
4. On success, the ticket is consumed (one-time use)

### Error Mapping

gRPC error codes are mapped to HTTP status codes:

| gRPC Code            | HTTP Code | Error Message                     |
| -------------------- | --------- | --------------------------------- |
| `NOT_FOUND`          | 404       | "resource not found"              |
| `ALREADY_EXISTS`     | 409       | "resource already exists"         |
| `PERMISSION_DENIED`  | 403       | "permission denied"               |
| `UNAUTHENTICATED`    | 401       | "authentication required"         |
| `INVALID_ARGUMENT`   | 400       | "invalid request"                 |
| `UNAVAILABLE`        | 503       | "service temporarily unavailable" |
| `INTERNAL`           | 500       | "internal server error"           |
| `RESOURCE_EXHAUSTED` | 429       | "rate limit exceeded"             |

### Health Checks

- `/health` - Liveness probe (always returns ok)
- `/ready` - Readiness probe (checks Valkey connectivity)

## Dependencies

| Dependency      | Purpose                                    |
| --------------- | ------------------------------------------ |
| Valkey          | Rate limiting, WebSocket ticket storage    |
| Auth Service    | User registration, login, token validation |
| User Service    | User profile queries                       |
| Room Service    | Room management                            |
| Message Service | Message operations                         |
