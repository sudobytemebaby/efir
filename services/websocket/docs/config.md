# WebSocket Service Configuration

## Environment Variables

| Variable                   | Type     | Default                 | Description                                |
| -------------------------- | -------- | ----------------------- | ------------------------------------------ |
| `CONFIG_PATH`              | string   | `config.yaml`           | Path to config file                        |
| `ENV`                      | string   | -                       | Environment (dev/staging/prod)             |
| `LOG_LEVEL`                | string   | `info`                  | Log level: debug, info, warn, error        |
| `WEBSOCKET_PORT`           | string   | `8081`                  | HTTP server port                           |
| `HUB_BUFFER_SIZE`          | int      | `256`                   | Hub channel buffer size                    |
| `SHUTDOWN_TIMEOUT`         | duration | `30s`                   | Graceful shutdown timeout                  |
| `READ_HEADER_TIMEOUT`      | duration | `5s`                    | HTTP header read timeout                   |
| `NATS_URL`                 | string   | `localhost:4222`        | NATS server URL                            |
| `NATS_USER`                | string   | -                       | NATS username                              |
| `NATS_PASSWORD`            | string   | -                       | NATS password                              |
| `NATS_RECONNECT_WAIT`      | duration | `2s`                    | Reconnect wait time                        |
| `NATS_MAX_RECONNECTS`      | int      | `-1`                    | Max reconnection attempts (-1 = unlimited) |
| `NATS_ACK_WAIT`            | duration | `30s`                   | Consumer ack wait time                     |
| `NATS_MAX_DELIVER`         | int      | `5`                     | Max message redeliveries                   |
| `NATS_CONSUMER_RETRY_WAIT` | duration | `2s`                    | Consumer retry interval                    |
| `VALKEY_ADDR`              | string   | `localhost:6379`        | Valkey server address                      |
| `VALKEY_PASSWORD`          | string   | -                       | Valkey password                            |
| `GATEWAY_URL`              | string   | `http://localhost:8080` | Gateway service URL                        |
| `WRITE_LIMIT`              | int64    | `4096`                  | Max write message size                     |
| `READ_LIMIT`               | int64    | `4096`                  | Max read message size                      |
| `PING_INTERVAL`            | duration | `30s`                   | Ping frequency                             |
| `READ_DEADLINE`            | duration | `35s`                   | Read timeout                               |
| `WRITE_DEADLINE`           | duration | `35s`                   | Write timeout                              |

## Example Configuration

```yaml
env: development
log_level: debug

server:
  port: "8081"
  hub_buffer_size: 256

timeouts:
  shutdown: 30s
  read_header: 5s

nats:
  url: nats://localhost:4222
  user: ""
  pass: ""
  reconnect_wait: 2s
  max_reconnects: -1
  ack_wait: 30s
  max_deliver: 5
  consumer_retry_wait: 2s

cache:
  addr: localhost:6379
  pass: ""

services:
  gateway_url: http://localhost:8080

websocket:
  write_limit: 4096
  read_limit: 4096
  ping_interval: 30s
  read_deadline: 35s
  write_deadline: 35s
```

## Configuration Files

Default config file path: `config.yaml` (service directory)
