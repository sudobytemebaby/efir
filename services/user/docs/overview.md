# User Service Overview

## Responsibilities

The User Service is responsible for:

- User profile management
- Profile creation (triggered by auth events)
- Profile updates (display name, avatar, bio)
- Profile retrieval

## Architecture

### Clean Architecture Layers

```
handler/     -> gRPC request handling
service/     -> Business logic, profile management
repository/  -> PostgreSQL user storage
nats/        -> Event subscription from auth service
```

### Request Flow

```
Client -> Gateway (HTTP) -> User Service (gRPC)
                                  |
                                  +-> Repository (PostgreSQL)
```

### Event-Driven Profile Creation

When a user registers via Auth Service:

1. Auth Service publishes `auth.user.registered` event
2. User Service subscribes and creates user profile
3. Profile is created with default display_name from email

### Error Handling

All errors map to gRPC status codes:

| Code               | Condition         |
| ------------------ | ----------------- |
| `NOT_FOUND`        | User not found    |
| `INVALID_ARGUMENT` | Validation failed |
| `INTERNAL`         | Database errors   |

### Health Checks

- `/health` - Liveness probe (always returns ok)
- `/ready` - Readiness probe (checks PostgreSQL)

## Dependencies

| Dependency | Purpose                      |
| ---------- | ---------------------------- |
| PostgreSQL | Profile storage              |
| NATS       | Event subscription from auth |
