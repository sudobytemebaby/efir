# healthcheck

HTTP health and readiness endpoint handler.

## Handler

```go
type Handler struct {
    ready atomic.Bool
}
```

`New()` creates a handler starting in "not ready" state.

## Endpoints

| Path      | Description                              |
| --------- | ---------------------------------------- |
| `/health` | Liveness probe — always returns `200 OK` |
| `/ready`  | Readiness probe — returns `200` or `503` |

## Response Format

```json
{"status": "ok"}
{"status": "ready"}
{"status": "not ready"}
```

## Key Methods

### `SetReady(ready bool)`

Manually sets the ready state. Thread-safe via `atomic.Bool`.

### `AwaitReady(ctx context.Context, checkFn func() bool, intervalMs int)`

Polls `checkFn` every `intervalMs` milliseconds until it returns true, then sets ready. Exits early if the context is canceled.

### `Register(mux *http.ServeMux)`

Registers `/health` and `/ready` handlers on the given `http.ServeMux`.

### `Middleware(next http.Handler) http.Handler`

Returns an `http.Handler` that intercepts `/health` and `/ready` paths, forwarding all other requests to `next`.

## Usage

### Option 1: Register on a mux

```go
h := healthcheck.New()
h.Register(mux)
http.ListenAndServe(":8080", mux)
```

### Option 2: Use as middleware

```go
h := healthcheck.New()
mux := http.NewServeMux()
// ... register your routes
http.ListenAndServe(":8080", h.Middleware(mux))
```

### Await readiness before serving

```go
h := healthcheck.New()
h.AwaitReady(ctx, func() bool {
    // check DB, NATS, etc. connections
    return db.Ping(ctx) == nil && nats.Connected()
}, 100)
```
