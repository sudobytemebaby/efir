# nats

NATS JetStream connection helpers, stream provisioning, and consumer creation utilities.

## Connection

### `Connect(url, user, password string, opts ConnectOptions) (*nats.Conn, error)`

Connects to NATS with credentials and reconnection settings.

```go
nc, err := nats.Connect("nats://localhost:4222", "myuser", "mypass",
    nats.ConnectOptions{
        ReconnectWait: 5 * time.Second,
        MaxReconnects: 10,
    })
```

### `New(nc *nats.Conn) (jetstream.JetStream, error)`

Creates a JetStream context from a NATS connection.

```go
js, err := nats.New(nc)
```

## Stream Provisioning

### `ProvisionStreams(ctx context.Context, js jetstream.JetStream, streams []StreamConfig) error`

Creates or updates one or more streams. Uses `CreateOrUpdateStream`, so it is idempotent.

```go
err := nats.ProvisionStreams(ctx, js, []nats.StreamConfig{
    {
        Name:     "MESSAGES",
        Subjects: []string{"message.>"},
        Storage:  jetstream.FileStorage,
    },
})
```

### `StreamConfig` / `ConsumerConfig`

Type aliases for `jetstream.StreamConfig` and `jetstream.ConsumerConfig`. Use these when provisioning streams and consumers.

## Consumer Provisioning

### `ProvisionConsumer(ctx context.Context, js jetstream.JetStream, stream string, cfg ConsumerConfig) (jetstream.Consumer, error)`

Creates or updates a durable consumer on a stream. Returns an error if the stream does not exist.

### `ProvisionConsumerWithRetry(ctx context.Context, js jetstream.JetStream, stream string, cfg ConsumerConfig, retryInterval time.Duration) (jetstream.Consumer, error)`

Creates a consumer with retry logic. Retries every `retryInterval` if the stream does not exist yet (`ErrStreamNotFound`). Exits if the context is canceled.

Use this in consumer main functions where the stream may not exist when the service starts before the publisher.

### `DefaultConsumerConfig(durable, filterSubject string, maxDeliver int, ackWait time.Duration) ConsumerConfig`

Returns a `ConsumerConfig` with explicit acknowledgment policy, suitable for most event consumers:

```go
cfg := nats.DefaultConsumerConfig(
    "my-consumer",
    "message.created",
    3,           // maxDeliver
    30*time.Second,
)
```

## Usage

```go
// Connect
nc, err := nats.Connect(cfg.URL, cfg.User, cfg.Pass,
    nats.ConnectOptions{
        ReconnectWait:  cfg.ReconnectWait,
        MaxReconnects: cfg.MaxReconnects,
    })
defer nc.Close()

js, err := nats.New(nc)

// Provision streams
streams := []nats.StreamConfig{{Name: "ROOM", Subjects: []string{"room.>"}}}
nats.ProvisionStreams(ctx, js, streams)

// Create consumer with retry
consumer, err := nats.ProvisionConsumerWithRetry(ctx, js, "ROOM",
    nats.DefaultConsumerConfig("room-updater", "room.updated", 3, 30*time.Second),
    5*time.Second)
```
