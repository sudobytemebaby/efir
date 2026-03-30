# logger

Structured JSON logging built on Go's standard `log/slog` package.

## Key Types

### `type Options struct`

```go
type Options struct {
    Level  Level  // minimum log level
    Output io.Writer  // defaults to os.Stdout
}
```

### `type Level = slog.Level`

Log levels: `LevelDebug`, `LevelInfo`, `LevelWarn`, `LevelError`.

## Key Functions

### `New(opts Options) *slog.Logger`

Creates a JSON-formatted slog logger.

```go
logger := logger.New(logger.Options{
    Level:  logger.LevelInfo,
    Output: os.Stdout,
})
```

### `ParseLevel(s string) (Level, error)`

Parses a string to slog.Level. Accepts `debug`, `info`, `warn`, `error` (case-insensitive).

```go
lvl, err := logger.ParseLevel("debug")
```

### `WithContext(ctx context.Context, logger *slog.Logger) context.Context`

Attaches a logger to a context.

```go
ctx = logger.WithContext(ctx, myLogger)
```

### `FromContext(ctx context.Context) *slog.Logger`

Retrieves the logger from context. Returns `slog.Default()` if none is set.

```go
log := logger.FromContext(ctx)
log.Info("hello", "key", "value")
```

## Usage

Services create a logger once at startup and pass it through context. All log output is JSON-formatted for machine parsing.

```go
cfg := loadConfig()
log := logger.New(logger.Options{
    Level:  *cfg.LogLevel,
    Output: os.Stdout,
})

// Attach to context for use in handlers
ctx := logger.WithContext(context.Background(), log)
```
