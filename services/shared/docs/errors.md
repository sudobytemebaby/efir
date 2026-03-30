# errors

Provides a unified error code system mapped across gRPC, HTTP, and internal representations.

## Error Codes

| Code                | gRPC Status       | HTTP | Default Message     |
| ------------------- | ----------------- | ---- | ------------------- |
| `NOT_FOUND`         | NotFound          | 404  | not found           |
| `ALREADY_EXISTS`    | AlreadyExists     | 409  | already exists      |
| `PERMISSION_DENIED` | PermissionDenied  | 403  | permission denied   |
| `UNAUTHENTICATED`   | Unauthenticated   | 401  | unauthenticated     |
| `INVALID_ARGUMENT`  | InvalidArgument   | 400  | invalid argument    |
| `UNAVAILABLE`       | Unavailable       | 503  | service unavailable |
| `INTERNAL`          | Internal          | 500  | internal error      |
| `RATE_LIMITED`      | ResourceExhausted | 429  | rate limit exceeded |

## Key Types

### `type Code string`

Error code identifiers.

### `type StatusError struct`

Wraps a code, message, and underlying error. Implements `error`, `Unwrap()`, `Code()`, and `GRPCStatus()`.

## Key Functions

### `Code.Error(msg string) error`

Creates a new `StatusError` with the given code and message.

```go
err := errors.CodeNotFound.Error("user not found")
```

### `Code.Wrap(err error) error`

Wraps an existing error with this code. Returns the error unchanged if it's already a `StatusError` with the same code.

```go
dbErr := db.QueryRow(ctx, "...").Scan(&user)
if dbErr == sql.ErrNoRows {
    return errors.CodeNotFound.Wrap(dbErr)
}
```

### `Code.ToGRPCCode() codes.Code`

Converts to the corresponding gRPC `codes.Code`.

### `Code.ToHTTPCode() int`

Converts to the corresponding HTTP status code.

### `FromError(err error) (Code, bool)`

Extracts the error code from a gRPC status error. Returns `(CodeInternal, true)` if the error is not a gRPC status.

## Usage

Services should return `*StatusError` from handlers. The gateway translates these to HTTP responses via `ToHTTPCode()`, and NATS event consumers receive gRPC status errors via `GRPCStatus()`.

```go
// Return an error
return nil, errors.CodeUnauthenticated.Error("token expired")

// Check an error
code, ok := errors.FromError(err)
if ok && code == errors.CodeNotFound {
    // handle not found
}
```
