# testutil

Shared test infrastructure for integration tests using testcontainers.

## PostgresContainer

Manages a testcontainers Postgres instance with per-test schema isolation for parallel-safe tests.

### `NewPostgresContainer(ctx context.Context, migrationsDir string) *PostgresContainer`

Starts a Postgres 17 Alpine container. `migrationsDir` is the path to goose SQL migration files (e.g. `"../../migrations"`). Panics on error.

### `Pool(t *testing.T) *pgxpool.Pool`

Creates a connection pool to a fresh schema with all migrations applied. Each call gets a unique schema. The schema is dropped automatically when `t` finishes.

```go
pool := pgContainer.Pool(t)
```

**Important:** Tests using `Pool` are safe to run in parallel — each test gets its own schema.

### `Terminate(ctx context.Context) error`

Stops the container. Call from `TestMain`'s defer.

## ValkeyContainer

Manages a testcontainers Valkey instance. All keys are flushed (`FLUSHDB`) after each test.

### `NewValkeyContainer(ctx context.Context) *ValkeyContainer`

Starts a Valkey 8 Alpine container. Panics on error.

### `Client(t *testing.T) vk.Client`

Returns a valkey-go client. All keys are flushed when `t` finishes.

**Important:** Do not use `t.Parallel()` with a shared `ValkeyContainer` unless tests use disjoint key spaces, since cleanup uses `FLUSHDB`.

### `Terminate(ctx context.Context) error`

Stops the container. Call from `TestMain`'s defer.

## NATSServer

An embedded in-process NATS server with JetStream enabled. Starts in <100ms, no Docker required.

### `NewNATSServer(t *testing.T) *NATSServer`

Starts an in-process NATS server with 128MB in-memory JetStream storage. The server is shut down when `t` finishes.

### `URL() string`

Returns the client URL.

### `JetStream(t *testing.T) jetstream.JetStream`

Returns a JetStream context. The connection is closed when `t` finishes.

```go
ns := testutil.NewNATSServer(t)
js := ns.JetStream(t)
```

## Fixtures

### `RandomUUID() uuid.UUID`

Returns a new random UUID.

### `RandomEmail() string`

Returns a unique test email (`user-<prefix>@test.example`).

### `RandomUsername() string`

Returns a unique username (`user-<prefix>`).

### `RandomRoomName() string`

Returns a unique room name (`room-<prefix>`).

### `HashedPassword(t *testing.T, password string) string`

Returns a bcrypt hash of the password using `MinCost` for fast test execution.

## Complete Test Setup Pattern

```go
var (
    pgContainer    *testutil.PostgresContainer
    valkeyContainer *testutil.ValkeyContainer
)

func TestMain(m *testing.M) {
    ctx := context.Background()
    pgContainer = testutil.NewPostgresContainer(ctx, "../../migrations")
    valkeyContainer = testutil.NewValkeyContainer(ctx)
    code := m.Run()
    pgContainer.Terminate(ctx)
    valkeyContainer.Terminate(ctx)
    os.Exit(code)
}

func TestCreateUser(t *testing.T) {
    pool := pgContainer.Pool(t)
    client := valkeyContainer.Client(t)
    ns := testutil.NewNATSServer(t)
    js := ns.JetStream(t)

    email := testutil.RandomEmail()
    // ... test code
}
```
