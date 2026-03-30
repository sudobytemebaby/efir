# Room Service Overview

## Responsibilities

The Room Service is responsible for:

- Chat room creation and management
- Room type handling (direct/group)
- Membership management
- Room event publishing

## Architecture

### Clean Architecture Layers

```
handler/     -> gRPC request handling
service/     -> Business logic, room management
repository/  -> PostgreSQL room/member storage
nats/        -> Event publishing to NATS
```

### Room Types

| Type     | Description                       |
| -------- | --------------------------------- |
| `DIRECT` | 1-on-1 conversations              |
| `GROUP`  | Group chats with multiple members |

### Request Flow

```
Client -> Gateway (HTTP) -> Room Service (gRPC)
                                   |
                                   +-> Repository (PostgreSQL)
                                   +-> NATS (events)
```

### Event Publishing

Room service publishes events when:

- Member joins/leaves room
- Room is updated

### Error Handling

All errors map to gRPC status codes:

| Code                | Condition             |
| ------------------- | --------------------- |
| `NOT_FOUND`         | Room not found        |
| `PERMISSION_DENIED` | User lacks permission |
| `INVALID_ARGUMENT`  | Validation failed     |
| `INTERNAL`          | Database errors       |

### Health Checks

- `/health` - Liveness probe
- `/ready` - Readiness probe (checks PostgreSQL)

## Dependencies

| Dependency | Purpose                     |
| ---------- | --------------------------- |
| PostgreSQL | Room and membership storage |
| NATS       | Event publishing            |
