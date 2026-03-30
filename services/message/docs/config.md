# Message Service Configuration

## Environment Variables

| Variable                   | Type      | Default           | Description                                |
| -------------------------- | --------- | ----------------- | ------------------------------------------ |
| `ENV`                      | string    | -                 | `development` or `production`              |
| `LOG_LEVEL`                | string    | `info`            | Log level: debug, info, warn, error        |
| `GRPC_PORT`                | string    | `50054`           | gRPC server port                           |
| `HEALTH_LISTEN_ADDR`       | string    | `:8080`           | Health check server address                |
| `SHUTDOWN_TIMEOUT`         | duration  | `30s`             | Graceful shutdown timeout                  |
| `READ_HEADER_TIMEOUT`      | duration  | `5s`              | Read header timeout                        |
| `GRPC_GRACEFUL_TIMEOUT`    | duration  | `5s`              | gRPC graceful shutdown timeout             |
| `NATS_URL`                 | string    | `localhost:4222`  | NATS server URL                            |
| `NATS_USER`                | string    | -                 | NATS username                              |
| `NATS_PASSWORD`            | string    | -                 | NATS password                              |
| `NATS_RECONNECT_WAIT`      | duration  | `2s`              | Reconnect wait time                        |
| `NATS_MAX_RECONNECTS`      | int       | `-1`              | Max reconnection attempts (-1 = unlimited) |
| `NATS_ACK_WAIT`            | duration  | `30s`             | Consumer ack wait time                     |
| `NATS_MAX_DELIVER`         | int       | `5`               | Max message redeliveries                   |
| `NATS_CONSUMER_RETRY_WAIT` | duration  | `2s`              | Consumer retry interval                    |
| `POSTGRES_DSN`             | string    | -                 | PostgreSQL connection string               |
| `ROOM_SERVICE_ADDR`        | string    | `localhost:50053` | Room service gRPC address                  |
| `ROOM_CALL_TIMEOUT`        | duration  | `5s`              | Room service call timeout                  |
| `RETRY_DELAYS`             | durations | `1s,2s,5s`        | Comma-separated retry delays               |

## Example Configuration

```yaml
env: development
log_level: debug

server:
  grpc_port: "50054"
  health_listen_addr: ":8080"

database:
  dsn: postgres://user:pass@localhost:5432/message?sslmode=disable

nats:
  url: nats://localhost:4222
  user: ""
  pass: ""
  reconnect_wait: 2s
  max_reconnects: -1
  ack_wait: 30s
  max_deliver: 5
  consumer_retry_wait: 2s

room:
  service_addr: "localhost:50053"
  call_timeout: 5s

timeouts:
  shutdown: 30s
  read_header: 5s
  grpc_graceful: 5s
  retry_delays: 1s,2s,5s
```

## Configuration Files

Default config file path: `config.yaml` (service directory)
