# WebSocket Service

Handles real-time WebSocket connections for delivering live events to clients.

## Quick Start

```bash
# Run the service
task websocket:run

# Run tests
task websocket:test
```

## Architecture

```
services/websocket/
├── cmd/main.go                  # Entry point
├── internal/
│   ├── config/                  # Configuration
│   ├── handler/                 # WebSocket handlers
│   │   └── ws.go                # Connection handling, pumps
│   ├── hub/                     # Room/connection management
│   │   └── hub.go               # Central hub
│   ├── subscriber/              # NATS event handlers
│   │   └── events.go            # Event processing
│   └── nats/                    # NATS consumer setup
│       └── consumers.go         # Consumer definitions
└── config.yaml                  # Default config
```

## Technology Stack

- **Language**: Go 1.25
- **WebSocket**: nhooyr.io/websocket
- **Message Broker**: NATS JetStream
- **Cache**: Valkey (ticket validation)
- **Protocol**: WebSocket

## Connection Flow

```
Client → WebSocket Service
         │
         ├─ Validate ticket via Valkey (GETDEL gateway:ws:ticket:{ticket})
         │
         ├─ Start readPump goroutine (read messages, route commands)
         │
         ├─ Start pingPump goroutine (send periodic pings)
         │
         └─ Register with Hub (room management)
              │
              └─ Hub.Run() (handle room subscriptions, broadcasts)
```

## WebSocket Endpoint

See [docs/api.md](docs/api.md) for connection and message documentation.

## Events

See [docs/events.md](docs/events.md) for NATS event subscriptions.

## Configuration

See [docs/config.md](docs/config.md) for configuration reference.
