# User Service

Manages user profiles including display names, avatars, and bios.

## Quick Start

```bash
# Run the service
task user:run

# Run tests
task user:test
```

## Architecture

```
services/user/
├── cmd/main.go              # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── handler/             # gRPC handlers
│   ├── service/             # Business logic
│   ├── repository/          # Database access
│   └── nats/                # NATS subscriber
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

| RPC             | Description               |
| --------------- | ------------------------- |
| `GetUser`       | Get user profile by ID    |
| `GetUsersByIds` | Get multiple users by IDs |
| `UpdateUser`    | Update user profile       |

## Configuration

See [docs/config.md](docs/config.md) for configuration reference.

## Events

See [docs/events.md](docs/events.md) for NATS event documentation.

## Database

See [docs/database.md](docs/database.md) for schema documentation.
