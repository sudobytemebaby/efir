# Room Service

Manages chat rooms (direct and group) and room memberships.

## Quick Start

```bash
# Run the service
task room:run

# Run tests
task room:test
```

## Architecture

```
services/room/
├── cmd/main.go              # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── handler/             # gRPC handlers
│   ├── service/             # Business logic
│   ├── repository/          # Database access
│   └── nats/                # NATS publisher
└── migrations/              # Database migrations
```

## Technology Stack

- **Language**: Go 1.25
- **Database**: PostgreSQL
- **Message Broker**: NATS JetStream
- **Protocol**: gRPC

## gRPC API

See [docs/api.md](docs/api.md) for detailed API documentation.

### Services

| RPC              | Description             |
| ---------------- | ----------------------- |
| `CreateRoom`     | Create a new room       |
| `GetRoom`        | Get room details        |
| `UpdateRoom`     | Update room name        |
| `DeleteRoom`     | Delete a room           |
| `AddMember`      | Add user to room        |
| `RemoveMember`   | Remove user from room   |
| `GetRoomMembers` | List room members       |
| `IsMember`       | Check if user is member |

## Configuration

See [docs/config.md](docs/config.md) for configuration reference.

## Events

See [docs/events.md](docs/events.md) for NATS event documentation.

## Database

See [docs/database.md](docs/database.md) for schema documentation.
