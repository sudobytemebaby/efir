# Auth Service Configuration

## Environment Variables

| Variable                   | Type     | Default          | Description                         |
| -------------------------- | -------- | ---------------- | ----------------------------------- |
| `ENV`                      | string   | -                | `development` or `production`       |
| `LOG_LEVEL`                | string   | `info`           | Log level: debug, info, warn, error |
| `GRPC_PORT`                | string   | `50051`          | gRPC server port                    |
| `HEALTH_LISTEN_ADDR`       | string   | `:8080`          | Health check server address         |
| `POSTGRES_DSN`             | string   | -                | PostgreSQL connection string        |
| `VALKEY_ADDR`              | string   | `localhost:6379` | Valkey address                      |
| `VALKEY_PASSWORD`          | string   | -                | Valkey password                     |
| `JWT_SECRET`               | string   | -                | JWT signing secret                  |
| `JWT_ACCESS_TTL`           | duration | `15m`            | Access token TTL                    |
| `JWT_REFRESH_TTL`          | duration | `168h`           | Refresh token TTL (7 days)          |
| `RATE_LIMIT_REQUESTS`      | int      | `5`              | Requests per window                 |
| `RATE_LIMIT_WINDOW`        | duration | `1m`             | Rate limit window                   |
| `NATS_URL`                 | string   | `localhost:4222` | NATS server URL                     |
| `NATS_USER`                | string   | -                | NATS username                       |
| `NATS_PASSWORD`            | string   | -                | NATS password                       |
| `NATS_RECONNECT_WAIT`      | duration | `5s`             | Reconnect wait time                 |
| `NATS_MAX_RECONNECTS`      | int      | `10`             | Max reconnection attempts           |
| `NATS_ACK_WAIT`            | duration | `30s`            | Message ack wait time               |
| `NATS_MAX_DELIVER`         | int      | `5`              | Max message redeliveries            |
| `NATS_CONSUMER_RETRY_WAIT` | duration | `5s`             | Consumer retry interval             |
| `SHUTDOWN_TIMEOUT`         | duration | `30s`            | Graceful shutdown timeout           |
| `READ_HEADER_TIMEOUT`      | duration | `5s`             | Read header timeout                 |
| `GRPC_GRACEFUL_TIMEOUT`    | duration | `10s`            | gRPC graceful shutdown timeout      |

## Example Configuration

```yaml
env: development
log_level: debug

server:
  grpc_port: "50051"
  health_listen_addr: ":8080"

database:
  dsn: postgres://user:pass@localhost:5432/auth?sslmode=disable

cache:
  addr: localhost:6379
  pass: ""

auth:
  secret: your-jwt-secret-key
  access_ttl: 15m
  refresh_ttl: 168h

rate_limit:
  requests: 5
  window: 1m

nats:
  url: localhost:4222
  user: ""
  pass: ""
  reconnect_wait: 5s
  max_reconnects: 10
  ack_wait: 30s
  max_deliver: 5
  consumer_retry_wait: 5s

timeouts:
  shutdown: 30s
  read_header: 5s
  grpc_graceful: 10s
```

## Configuration Files

Default config file path: `config.yaml` (service directory)
