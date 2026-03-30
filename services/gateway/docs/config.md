# Gateway Service Configuration

## Environment Variables

| Variable               | Type     | Default       | Description                         |
| ---------------------- | -------- | ------------- | ----------------------------------- |
| `CONFIG_PATH`          | string   | `config.yaml` | Path to config file                 |
| `ENV`                  | string   | -             | Environment (dev/staging/prod)      |
| `LOG_LEVEL`            | string   | `info`        | Log level: debug, info, warn, error |
| `GATEWAY_PORT`         | string   | `8080`        | HTTP server port                    |
| `MAX_BODY_SIZE`        | int      | `1048576`     | Max request body size (1MB)         |
| `SHUTDOWN_TIMEOUT`     | duration | `30s`         | Graceful shutdown timeout           |
| `READ_HEADER_TIMEOUT`  | duration | `10s`         | Read header timeout                 |
| `GRPC_TIMEOUT`         | duration | `10s`         | gRPC call timeout                   |
| `VALKEY_ADDR`          | string   | -             | Valkey server address               |
| `VALKEY_PASSWORD`      | string   | -             | Valkey password                     |
| `JWT_SECRET`           | string   | -             | JWT signing secret                  |
| `WS_TICKET_TTL`        | duration | `5m`          | WebSocket ticket TTL                |
| `RATE_LIMIT_REQUESTS`  | int      | `100`         | Max requests per window             |
| `RATE_LIMIT_WINDOW`    | duration | `60s`         | Rate limit window                   |
| `AUTH_SERVICE_ADDR`    | string   | -             | Auth gRPC service address           |
| `USER_SERVICE_ADDR`    | string   | -             | User gRPC service address           |
| `ROOM_SERVICE_ADDR`    | string   | -             | Room gRPC service address           |
| `MESSAGE_SERVICE_ADDR` | string   | -             | Message gRPC service address        |

## Example Configuration

```yaml
env: development
log_level: debug

server:
  port: "8080"
  max_body_size: 1048576

timeouts:
  shutdown: 30s
  read_header: 10s
  grpc: 10s

cache:
  addr: localhost:6379
  pass: ""

auth:
  secret: your-jwt-secret-key
  ws_ticket_ttl: 5m

rate_limit:
  requests: 100
  window: 60s

services:
  auth: localhost:50051
  user: localhost:50052
  room: localhost:50053
  message: localhost:50054
```

## Configuration Files

Default config file path: `config.yaml` (service directory)
