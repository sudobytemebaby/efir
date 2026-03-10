## ADR-004: Go Workspace

## Status

Accepted

## Context

Go 1.18+ introduced workspaces as the official way to work with multi-module repositories. Previously, the `replace` directive in `go.mod` was used to manage local dependencies.

## Decision

Use `go.work` to define the workspace with all service modules.

## Rationale

- **Official tool**: First-class support in Go toolchain
- **No replace directives**: Clean go.mod files per service
- **Easy management**: `go work use ./services/auth` to add/remove modules
- **Consistent**: All modules available without path issues
- **IDE support**: Works with VSCode, GoLand, etc.

## Example

```go
// go.work
go 1.23

use (
    ./services/auth
    ./services/user
    ./services/room
    ./services/message
    ./services/websocket
    ./services/gateway
    ./services/shared
)
```

## Alternatives Considered

- **Replace directives in go.mod**:
  - Pros: Works with older Go versions
  - Cons: Harder to maintain, messy go.mod files, each service needs its own replace

- **Single mega-module**:
  - Pros: Simple, no workspace needed
  - Cons: Not suitable for microservices with independent deployments

## Consequences

- Each service has its own go.mod (independent versioning)
- Build each service independently: `go build ./services/auth`
- Import shared code: `import "efir.sh/services/shared"`
- CI builds each service separately
- Requires Go 1.18+
