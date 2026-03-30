# Sidecar Service

Policy Enforcement Point (PEP) for validating and proxying gRPC requests.

## Status

This service is currently a **placeholder**. Full implementation is planned for Module 2.

## Quick Start

```bash
# Run the service
task sidecar:run

# Run tests
task sidecar:test
```

## Architecture

```
services/sidecar/
├── cmd/main.go                    # Entry point
├── internal/
│   └── config/                   # Configuration
├── config/                       # Policy files (future)
└── Dockerfile
```

## Planned Architecture

The sidecar will act as a gRPC reverse proxy that validates requests before forwarding to upstream services:

```
Gateway (gRPC client)
        │
        ▼
  Sidecar (port 51051)
  ├── Validate protobuf
  ├── Enforce policy rules
  └── Proxy to upstream
        │
        ▼
  Upstream Service
```

### Per-Service Sidecars

One sidecar instance per upstream service:

| Sidecar         | Upstream        | Port  |
| --------------- | --------------- | ----- |
| auth-sidecar    | Auth Service    | 50051 |
| user-sidecar    | User Service    | 50052 |
| room-sidecar    | Room Service    | 50053 |
| message-sidecar | Message Service | 50054 |

## Technology Stack

- **Language**: Go 1.25
- **Protocol**: gRPC reverse proxy
- **Policy**: YAML-based policy files

## Current Implementation

Only basic health endpoints are implemented:

| Endpoint  | Method | Description     |
| --------- | ------ | --------------- |
| `/health` | GET    | Liveness probe  |
| `/ready`  | GET    | Readiness probe |

## Planned Features

- Protobuf message validation
- Policy-based request enforcement
- Request routing to upstream services
- OpenTelemetry metrics and traces

## Configuration

See [docs/config.md](docs/config.md) for planned configuration reference.
