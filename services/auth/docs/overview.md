# Auth Service Overview

## Responsibilities

The Auth Service is responsible for:

- User registration with email/password
- User login authentication
- JWT access/refresh token generation
- Token refresh and invalidation
- Rate limiting for auth operations
- Publishing user registration events

## Architecture

### Clean Architecture Layers

```
handler/     -> gRPC request handling, proto validation
service/     -> Business logic, token generation
repository/  -> PostgreSQL account storage
nats/        -> Event publishing to NATS
ratelimit/   -> Valkey-based rate limiting
```

### Request Flow

```
Client -> Gateway (HTTP) -> Auth Service (gRPC)
                             |
                             +-> Repository (PostgreSQL)
                             +-> Valkey (tokens, rate limit)
                             +-> NATS (events)
```

### Rate Limiting

Rate limiting is applied per email address using Valkey:

- Key format: `auth:ratelimit:{action}:{email}`
- Default: 5 requests per minute per action/email
- Actions: `register`, `login`, `refresh`, `logout`

### Token Management

**Access Token**:

- Short-lived JWT (default: 15 minutes)
- Contains: user_id, exp, iat

**Refresh Token**:

- Long-lived opaque token
- Stored in Valkey: `auth:refresh:{token}`
- Default TTL: 7 days

### Error Handling

All errors map to gRPC status codes:

| Code                 | Condition                     |
| -------------------- | ----------------------------- |
| `INVALID_ARGUMENT`   | Invalid email/password format |
| `NOT_FOUND`          | User not found                |
| `ALREADY_EXISTS`     | Email already registered      |
| `RESOURCE_EXHAUSTED` | Rate limit exceeded           |
| `INTERNAL`           | Database/cache errors         |

### Health Checks

- `/health` - Liveness probe (always returns ok)
- `/ready` - Readiness probe (checks PostgreSQL + Valkey)

## Dependencies

| Dependency | Purpose                      |
| ---------- | ---------------------------- |
| PostgreSQL | Account storage              |
| Valkey     | Token storage, rate limiting |
| NATS       | Event publishing             |
