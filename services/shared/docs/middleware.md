# middleware

gRPC interceptors for logging, panic recovery, request ID generation, and user ID extraction.

## Constants

```go
MetadataKeyUserID    = "x-user-id"
MetadataKeyRequestID = "x-request-id"
```

These metadata keys are used to pass user and request IDs through the gRPC pipeline.

## Interceptors

### `RequestIDInterceptor() grpc.UnaryServerInterceptor`

Extracts `x-request-id` from incoming metadata, or generates a UUID if absent. Stores the request ID in context for retrieval via `GetRequestID(ctx)`.

### `LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor`

Logs every gRPC request at start and completion. Records method name, request ID, duration in milliseconds, and error details (including error code from `StatusError`).

### `RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor`

Catches panics in handlers, logs the panic with method and request ID, and returns an `Internal` gRPC error. Prevents unhandled panics from crashing the server.

### `UserIDInterceptor() grpc.UnaryServerInterceptor`

Extracts `x-user-id` from incoming metadata and stores it in context for retrieval via `GetUserID(ctx)`. Passes through if not present.

## Context Helpers

### `GetUserID(ctx context.Context) (string, bool)`

Returns the user ID from context and whether it was present.

```go
userID, ok := middleware.GetUserID(ctx)
```

### `GetRequestID(ctx context.Context) (string, bool)`

Returns the request ID from context and whether it was present.

```go
reqID, ok := middleware.GetRequestID(ctx)
```

## Usage

Apply interceptors in order when creating the gRPC server:

```go
var logger *slog.Logger // from shared/pkg/logger

grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        middleware.RecoveryInterceptor(logger),
        middleware.RequestIDInterceptor(),
        middleware.UserIDInterceptor(),
        middleware.LoggingInterceptor(logger),
    ),
)
```

Order matters: `RecoveryInterceptor` should be outermost to catch panics from inner interceptors.
