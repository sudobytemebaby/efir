# Message Service

Handles message sending, retrieval, and deletion within chat rooms.

## Quick Start

```bash
# Run the service
task message:run

# Run tests
task message:test
```

## Architecture

```
services/message/
├── cmd/main.go              # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── handler/             # gRPC handlers
│   ├── service/             # Business logic
│   ├── repository/          # Database access
│   ├── client/              # Room service gRPC client
│   └── nats/                # NATS publisher
└── migrations/              # Database migrations
```

## Technology Stack

- **Language**: Go 1.25
- **Database**: PostgreSQL
- **Message Broker**: NATS JetStream
- **Protocol**: gRPC
- **Remote Calls**: gRPC to Room Service

## gRPC API

See [docs/api.md](docs/api.md) for detailed API documentation.

### Services

| RPC              | Description                          |
| ---------------- | ------------------------------------ |
| `SendMessage`    | Send a message to a room             |
| `GetMessages`    | List messages with cursor pagination |
| `GetMessageById` | Fetch a single message by ID         |
| `DeleteMessage`  | Soft-delete a message (sender only)  |

## Configuration

See [docs/config.md](docs/config.md) for configuration reference.

## Events

See [docs/events.md](docs/events.md) for NATS event documentation.

## Database

See [docs/database.md](docs/database.md) for schema documentation.
