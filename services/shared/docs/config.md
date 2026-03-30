# config

Shared configuration structs for services.

## Environment

```go
type Environment string
const (
    EnvDevelopment Environment = "development"
    EnvProduction  Environment = "production"
)
```

Services should call `Validate()` on startup to reject unknown environments.

## ServerConfig

```go
type ServerConfig struct {
    GRPCPort          string `yaml:"grpc_port"           env:"GRPC_PORT"`
    HealthListenAddr  string `yaml:"health_listen_addr"  env:"HEALTH_LISTEN_ADDR"`
}
```

- `GRPC_PORT` - port for the gRPC server (e.g. `":50051"`)
- `HEALTH_LISTEN_ADDR` - address for HTTP health endpoints (e.g. `":8081"`)

## TimeoutsConfig

```go
type TimeoutsConfig struct {
    Shutdown      time.Duration `yaml:"shutdown"        env:"SHUTDOWN_TIMEOUT"`
    ReadHeader    time.Duration `yaml:"read_header"    env:"READ_HEADER_TIMEOUT"`
    GRPCGraceful  time.Duration `yaml:"grpc_graceful"  env:"GRPC_GRACEFUL_TIMEOUT"`
}
```

Default values vary per service. Services typically use a `Shutdown` of 15-30s, `ReadHeader` of 5s, and `GRPCGraceful` of 10s.

## NATSConfig

```go
type NATSConfig struct {
    URL                string        `yaml:"url"                  env:"NATS_URL"`
    User               string        `yaml:"user"                 env:"NATS_USER"`
    Pass               string        `yaml:"-"                    env:"NATS_PASSWORD"`
    ReconnectWait      time.Duration `yaml:"reconnect_wait"      env:"NATS_RECONNECT_WAIT"`
    MaxReconnects      int           `yaml:"max_reconnects"      env:"NATS_MAX_RECONNECTS"`
    AckWait            time.Duration `yaml:"ack_wait"            env:"NATS_ACK_WAIT"`
    MaxDeliver         int           `yaml:"max_deliver"         env:"NATS_MAX_DELIVER"`
    ConsumerRetryWait  time.Duration `yaml:"consumer_retry_wait" env:"NATS_CONSUMER_RETRY_WAIT"`
}
```

- `NATS_URL` - NATS server URL (e.g. `nats://localhost:4222`)
- `NATS_USER` / `NATS_PASSWORD` - credentials for NATS authentication
- `ReconnectWait` - time between reconnection attempts
- `MaxReconnects` - maximum number of reconnection attempts
- `AckWait` - time to wait for consumer acknowledgment before redelivering
- `MaxDeliver` - maximum delivery attempts per message
- `ConsumerRetryWait` - interval between retrying consumer creation when streams are not yet available

## Usage

Services define their own config structs embedding these shared types:

```go
type Config struct {
    Environment   config.Environment
    Server        config.ServerConfig
    Timeouts      config.TimeoutsConfig
    NATS          config.NATSConfig
    Database      DatabaseConfig // service-specific
}
```

Services then use viper or a similar library to bind environment variables and YAML fields.
