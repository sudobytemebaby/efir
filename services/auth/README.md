# Auth Service

Handles user authentication including registration, login, logout, and JWT token management.

## Quick Start

```bash
# Run the service
task auth:run

# Run tests
task auth:test
```

## Architecture

```
services/auth/
├── cmd/main.go              # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── handler/             # gRPC handlers
│   ├── service/             # Business logic
│   ├── repository/          # Database access
│   ├── nats/                # NATS publisher
│   └── ratelimit/           # Rate limiting
└── migrations/              # Database migrations
```

## Technology Stack

- **Language**: Go 1.25
- **Database**: PostgreSQL
- **Cache**: Valkey
- **Message Broker**: NATS JetStream
- **Protocol**: gRPC
- **Auth**: JWT

## gRPC API

See [docs/api.md](docs/api.md) for detailed API documentation.

### Services

| RPC            | Description                           |
| -------------- | ------------------------------------- |
| `Register`     | Register a new user account           |
| `Login`        | Authenticate and receive tokens       |
| `Logout`       | Invalidate refresh token              |
| `RefreshToken` | Exchange refresh token for new tokens |

## Configuration

See [docs/config.md](docs/config.md) for configuration reference.

## Events

See [docs/events.md](docs/events.md) for NATS event documentation.

## Database

See [docs/database.md](docs/database.md) for schema documentation.
