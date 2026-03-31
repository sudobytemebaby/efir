# Efir Codebase Review — Polishing Roadmap

> Generated: 2026-03-21 | Branch: `feat/gateway` | ~18k LOC across 6 services
>
> This document is an ordered, actionable list of every issue found during a full
> codebase review. Items are grouped into phases so they can be tackled
> incrementally. Each item states the **problem**, the **affected files**, and
> a concrete **task** to resolve it.

---

## Phase 0 — Critical Bugs (data loss / incorrect behaviour)

These must be fixed before any other work — they produce wrong results or can
lock users out.

### 0.1 WebSocket duplicate registration

**Problem**: Every connection that provides `room_id` is registered **twice** in
the same room, causing duplicate message delivery. Connections without `room_id`
are registered into an empty-string room.

**Files**: `services/websocket/internal/handler/ws.go:56-60`

```go
h.hub.Register(wsConn, userID, initialRoomID)   // unconditional
if initialRoomID != "" {
    h.hub.Register(wsConn, userID, initialRoomID) // again
}
```

**Task**:

- Remove line 56 (the unconditional register).
- Keep only the conditional block that registers when `initialRoomID != ""`.

---

### 0.2 RefreshToken deletes old token before saving new one

**Problem**: In `RefreshToken`, the old refresh token is deleted (line 167)
before the new token pair is generated and saved (line 171). If
`generateTokenPair` or the underlying `SaveRefreshToken` fails, the user has no
valid refresh token left — they are permanently locked out until they log in
again with credentials.

**Files**: `services/auth/internal/service/auth.go:158-176`

**Task**:

- Reorder: generate and persist the new token pair **first**, then delete the
  old token.
- Alternatively, wrap both operations in a helper that rolls back (re-saves the old token) on failure.

---

### 0.3 Rate limiter INCR / EXPIRE race condition (auth service)

**Problem**: `Allow()` uses two separate Valkey commands — `INCR` then
`EXPIRE` (only when `count == 1`). If the process crashes between them, the key
has no TTL and persists forever, permanently rate-limiting that email. Concurrent
first-requests can also both see `count == 1`.

**Files**: `services/auth/internal/ratelimit/ratelimit.go:50-72`

**Task**:

- Replace the two commands with the same Lua-script approach already used in the
  gateway (`services/gateway/internal/middleware/ratelimit.go:13-19`).
- Alternatively, extract the Lua script into `shared/pkg/valkey` so both the
  gateway and the auth service share one atomic implementation.

---

### 0.4 gRPC connection leak in all client packages

**Problem**: Every `NewXxxClient()` function calls `grpc.NewClient()` but never
stores the returned `*grpc.ClientConn`. The connection can never be closed,
leaking goroutines and file descriptors over the lifetime of the process.

**Files**:

- `services/gateway/internal/client/auth.go:27`
- `services/gateway/internal/client/user.go:20`
- `services/gateway/internal/client/room.go:26`
- `services/gateway/internal/client/message.go:33`
- `services/message/internal/client/room.go:21`

**Task**:

- Store `conn *grpc.ClientConn` in each client struct.
- Add a `Close() error` method that calls `conn.Close()`.
- Wire `Close()` into the graceful shutdown sequence in each service's
  `cmd/main.go`.

---

## Phase 1 — Security & Error Handling

### 1.1 Gateway forwards raw gRPC error messages to HTTP clients

**Problem**: Every gateway handler does
`http.Error(w, err.Error(), code.ToHTTPCode())`. This exposes internal gRPC
status messages (potentially DB errors, stack context) directly to end users.
Successful responses are JSON; errors are `text/plain` — inconsistent contract.

**Files**: All `services/gateway/internal/handler/*.go` — every error branch.

**Task**:

- Create a shared `writeErrorJSON(w, httpCode, safeMsg)` helper that writes a
  consistent `{"error":"...","code":"..."}` JSON body.
- Map known gRPC status codes to safe, user-facing messages.
- Never forward `err.Error()` verbatim. Log the full error server-side.

---

### 1.2 `errors.Code.Wrap()` destroys the error chain

**Problem**: `Wrap()` calls `status.Error(code, err.Error())`, which converts
the error to a string. The original error chain (`errors.Is` /
`errors.As`) is lost. Debugging in production becomes string-grepping logs.

**Files**: `services/shared/pkg/errors/errors.go:53-58`

**Task**:

- Log the full original error (with `%+v` or structured fields) **before**
  converting to a gRPC status at the handler boundary.
- Consider using `status.FromError()` for errors that are already gRPC statuses,
  to preserve the code without re-wrapping.

---

### 1.3 `errors.FromError()` returns empty `Code` for nil; `ToHTTPCode()` / `ToGRPCCode()` return zero for unknown codes

**Problem**: `FromError(nil)` returns `""`. `Code("").ToHTTPCode()` returns `0`.
If a nil error ever leaks into the error-mapping path, the client gets HTTP
status 0 and `codes.OK` — silently masking the issue. Same applies to any
unmapped `Code` value.

**Files**: `services/shared/pkg/errors/errors.go:41-47, 60-62`

**Task**:

- Add a default/fallback in `ToGRPCCode()` and `ToHTTPCode()` that returns
  `codes.Internal` / `500` for unknown keys.
- Either return a sentinel `CodeOK` for nil or change `FromError` to return
  `(Code, bool)`.

---

### 1.4 Rate limit error leaks email addresses

**Problem**: `ErrRateLimitExceeded.Error()` includes the email in the message
string: `"rate limit exceeded for login on user@example.com"`. If this ever
surfaces in logs aggregated with third-party tools or error responses, it's
information disclosure.

**Files**: `services/auth/internal/ratelimit/ratelimit.go:23-25`

**Task**:

- Remove email from the error message string.
- Keep email only as structured log fields at the call site that handles the
  error.

---

### 1.5 Event publishing failures silently swallowed (all services)

**Problem**: Auth, room, and message services all log-and-ignore when NATS
publish fails. The event is lost. Other services (user service, websocket) never
learn about the change. No metric, no dead-letter queue, no retry.

**Files**:

- `services/auth/internal/service/auth.go:110-116`
- `services/room/internal/service/room.go:130-135, 193-201, 238-246`
- `services/message/internal/service/message.go:88-102`

**Task** (incremental):

1. **Immediate**: Add a counter metric (or at minimum a distinct log field like
   `"event_lost": true`) so lost events are alertable.
2. **Next iteration**: Implement transactional outbox pattern (you already have
   the TODO in auth). Write the event to a DB table in the same transaction as
   the domain write; a background worker publishes from the outbox.

---

## Phase 2 — Unidiomatic Patterns

### 2.1 Config stores durations as strings, parsed lazily

**Problem**: `AccessTTL`, `RefreshTTL`, `RateLimitWindow` etc. are `string`
fields parsed by separate `ParseXxxTTL()` methods called later in `main()`.
Invalid values aren't caught at config load time; they blow up after resources
are already initialized.

**Files**: All `internal/config/config.go` across services.

**Task**:

- Parse all durations eagerly inside `Load()`.
- Store as `time.Duration` directly in the `Config` struct.
- Remove the `ParseXxx()` methods and the corresponding error handling in
  `main.go`.
- Apply the same to any future config that needs parsing (ports, addresses,
  etc.).

---

### 2.2 `Config.Env.Validate()` is defined but never called (all services)

**Problem**: Every service defines `func (e Environment) Validate() error` but
none calls it. Invalid `ENV` values (e.g. `"staging"`) are silently accepted.

**Files**: All `internal/config/config.go` across services.

**Task**:

- Call `cfg.Env.Validate()` inside `Load()`, fail early with a clear error.

---

### 2.3 `main()` is a flat script — no `run()` function

**Problem**: Every service's `main()` is a ~200-line flat function using
`os.Exit(1)` for every error. This makes the lifecycle untestable and `defer`
behaviour fragile.

**Files**: All `cmd/main.go` across services.

**Task**:

- Extract to `func run(ctx context.Context) error`.
- `main()` becomes: create signal context → call `run()` → `os.Exit(1)` on
  error.
- This enables integration tests that spin up the full service in-process.

---

### 2.4 Hub mixes channel serialization with mutex — redundant concurrency model

**Problem**: The hub uses a channel-based `Run()` loop (single-goroutine
ownership) **and** a `sync.RWMutex`. Since all state mutations happen in the
`Run()` goroutine, the mutex is redundant and adds complexity.

**Files**: `services/websocket/internal/hub/hub.go`

**Task**:

- Remove the `sync.RWMutex`.
- All mutations already run inside `Run()` via channel dispatch — they are
  naturally serialized.
- For `sendToRoom`, snapshot the target list inside the `Run` goroutine (no lock
  needed since no other goroutine writes), then write outside the loop.
- If `GetRoomUserCount` needs to be called from other goroutines, either route it
  through the channel too, or keep a minimal `sync.RWMutex` around **only** that
  read method.

---

### 2.5 Repository types leak into service interfaces

**Problem**: Service interfaces return/accept repository types directly
(`repository.Room`, `repository.User`, `repository.MessageType`, etc.). The
handler layer transitively imports the repository package.

**Files**:

- `services/room/internal/service/room.go` — `RoomService` interface
- `services/user/internal/service/user.go` — `UserService` interface
- `services/message/internal/service/message.go` — `MessageService` interface

**Task**:

- Define domain/model types in the service package (or a dedicated `model`
  package).
- Have the repository convert between domain types and persistence types.
- Update handler to use service-level types.

> Note: this is a larger refactor. Acceptable to defer past MVP, but track it.

---

### 2.6 Ownership checks use `CreatedBy` instead of member role

**Problem**: Authorization checks compare `room.CreatedBy != requesterID`
instead of checking the member's role from `room_members`. The
`MemberRoleOwner` stored in the DB is unused at runtime. Ownership can never be transferred, and adding moderator/admin roles requires reworking every check.

**Files**: `services/room/internal/service/room.go:104, 149, 216`

**Task**:

- Fetch the requester's role from `room_members` and check role-based
  permissions instead of `room.CreatedBy`.
- This also fixes the issue where the owner can be removed from a room
  (leaving an orphan room with no owner).

---

### 2.7 Publisher nil-check inconsistency (optional vs. required)

**Problem**: In the room service, `publisher` is nil-guarded in `AddMember` and
`RemoveMember` but called directly in `UpdateRoom`. Unclear whether it's
optional or required.

**Files**: `services/room/internal/service/room.go:130, 192, 237`

**Task**:

- If required: validate non-nil in `NewRoomService`, remove all nil checks.
- If optional: use a `noopPublisher` (null object pattern) that satisfies the
  interface silently. Then no nil checks are needed anywhere.

---

### 2.8 Handler directly references repository errors (layer violation)

**Problem**: The room handler imports `repository.ErrMemberNotFound` and checks
it directly, bypassing the service layer's error abstraction.

**Files**: `services/room/internal/handler/room.go:228`

**Task**:

- Define `ErrMemberNotFound` in the service package.
- Have the service wrap/return it from `RemoveMember`.
- Handler checks only service-level errors.

---

### 2.9 Dual JWT validation — gateway validates locally; auth has unused `ValidateToken` RPC

**Problem**: The gateway parses JWTs locally with the shared secret. The auth
service exposes `ValidateToken` via gRPC but nothing calls it in the runtime
flow. Two validation paths for the same token is confusing.

**Files**:

- `services/gateway/internal/middleware/auth.go`
- `services/auth/internal/handler/auth.go:112`

**Task**:

- Decide on one strategy:
  - **Local validation** (current gateway path): remove `ValidateToken` RPC from auth, or repurpose it for the websocket ticket flow.
  - **Remote validation** (every request hits auth): remove local JWT parsing
    from gateway. Slower but enables token revocation.
- Document the choice in an ADR.

---

### 2.10 Message service: reply validation conflates DB errors with "not found"

**Problem**: When validating `ReplyToID`, any error from `GetMessageByID` is
treated as `ErrInvalidReplyTarget` — including real database failures.

**Files**: `services/message/internal/service/message.go:64-68`

```go
original, err := s.repo.GetMessageByID(ctx, *input.ReplyToID)
if err != nil {
    return nil, ErrInvalidReplyTarget // could be a DB error
}
```

**Task**:

- Check specifically for `repository.ErrMessageNotFound` → return
  `ErrInvalidReplyTarget`.
- For any other error, propagate the real error upward.

---

## Phase 3 — Operational Hardening

### 3.1 Health check marked ready before servers are actually serving

**Problem**: All services call `healthHandler.SetReady(true)` before the gRPC /
HTTP server goroutines have bound their ports. A load balancer could route
traffic during the gap.

**Files**: All `cmd/main.go` across services.

**Task**:

- Create the `net.Listener` **before** starting the goroutine.
- Set ready **after** confirming the listener is bound.
- Or use `AwaitReady` with a check that pings the listening port.

---

### 3.2 No read/write deadlines on WebSocket connections

**Problem**: `readPump` calls `conn.Read(context.Background())` — blocks
forever. `WriteJSON` uses `context.Background()` too. Config defines
`ReadDeadline()` and `PingInterval()` but they're never used. Dead clients are
never evicted.

**Files**: `services/websocket/internal/handler/ws.go:84, 172`

**Task**:

- Use `context.WithTimeout` for reads, derived from config's `ReadDeadline`.
- Accept a context in `WriteJSON` (or set a write deadline on the underlying
  conn).
- Implement server-initiated ping/pong using `PingInterval()` from config to
  detect dead connections.

---

### 3.3 `sendToRoom` blocks on slow clients

**Problem**: `hub.sendToRoom` iterates connections and calls `WriteJSON`
sequentially. One slow/stuck client blocks the entire hub from processing other broadcasts for that room.

**Files**: `services/websocket/internal/hub/hub.go:236-243`

**Task**:

- Add a per-connection write timeout (via context or deadline).
- Consider per-connection outbound buffered channel with a goroutine that drains
  it — if the buffer is full, drop the message and close the connection.

---

### 3.4 gRPC graceful shutdown has no timeout

**Problem**: `grpcServer.GracefulStop()` blocks indefinitely if connections
don't close. The HTTP health server gets a proper shutdown timeout, but gRPC
doesn't.

**Files**: All `cmd/main.go` in gRPC services.

**Task**:

- Run `GracefulStop()` in a goroutine.
- If a timeout expires, call `grpcServer.Stop()` (hard stop).
- Pattern:
  ```go
  done := make(chan struct{})
  go func() { grpcServer.GracefulStop(); close(done) }()
  select {
  case <-done:
  case <-time.After(5 * time.Second):
      grpcServer.Stop()
  }
  ```

---

### 3.5 Missing `CGO_ENABLED=0` in Dockerfiles

**Problem**: All Dockerfiles build without `CGO_ENABLED=0`. If any dependency
uses CGO, the binary won't run in the Alpine final image.

**Files**: All `services/*/Dockerfile`

**Task**:

- Add `CGO_ENABLED=0` to the `go build` line in every Dockerfile.

---

### 3.6 No request tracing / correlation IDs

**Problem**: No service propagates request IDs or trace IDs through gRPC
metadata or structured logs. In a 6-service architecture, correlating a request
across services is currently impossible.

**Files**: All services.

**Task** (incremental):

1. Add a middleware that extracts or generates a request/trace ID from gRPC
   metadata / HTTP headers.
2. Inject it into the context and include it in all log calls.
3. Pass it downstream in gRPC metadata when making inter-service calls.

---

## Phase 4 — Code Quality & Consistency

### 4.1 Gateway is a manual gRPC-JSON transcoder (~800 lines of boilerplate)

**Problem**: The gateway hand-writes JSON request/response structs that mirror
proto definitions, manually converts between them, and manually maps errors.
Every proto field change requires updating three places. `message.go` alone is
450 lines of pure mapping.

**Files**: `services/gateway/internal/handler/{message,room,user,auth}.go`

**Task**:

- Evaluate **grpc-gateway** or **connect-go** to auto-generate the HTTP layer
  from proto annotations.
- If keeping the manual approach, extract the type-switch mapping into generated or table-driven code to reduce per-type boilerplate.

> Note: this is a significant refactor. Consider after MVP if the manual
> approach is causing too many bugs.

---

### 4.2 Logging interceptor logs `"error", nil` on every successful request

**Problem**: `LoggingInterceptor` always includes `"error", err` in the log
attributes, even when `err == nil`. This produces noisy `error=<nil>` entries.

**Files**: `services/shared/pkg/middleware/middleware.go:35-39`

**Task**:

- Conditionally include the error attribute only when `err != nil`.

---

### 4.3 Missing test coverage for shared middleware

**Problem**: `RecoveryInterceptor` and `LoggingInterceptor` have zero test
coverage. Only `UserIDInterceptor` and `GetUserID` are tested.

**Files**: `services/shared/pkg/middleware/middleware_test.go`

**Task**:

- Add tests for `RecoveryInterceptor`: verify it catches panics and returns
  `codes.Internal`.
- Add tests for `LoggingInterceptor`: verify log output for success and error
  cases, verify duration is recorded.

---

### 4.4 Slice pre-allocation missing in hot paths

**Problem**: Multiple handlers build slices with `var s []T` + `append` instead
of pre-allocating with `make([]T, 0, len(input))`.

**Files**:

- `services/user/internal/handler/user.go` — `GetUsersByIds`
- `services/room/internal/handler/room.go` — member ID collection
- `services/room/internal/service/room.go:187-190, 232-235`

**Task**:

- Replace `var s []T` + loop-append with `make([]T, 0, len(source))` or
  `make([]T, len(source))` with index assignment.

---

### 4.5 Repeated recipientID-building code in room service

**Problem**: The same 3-line loop to extract `[]uuid.UUID` from
`[]RoomMember` appears three times in the room service.

**Files**: `services/room/internal/service/room.go:125-128, 187-190, 232-235`

**Task**:

- Extract a helper: `func memberUserIDs(members []repository.RoomMember) []uuid.UUID`.

---

### 4.6 Dual rate limiting with different atomicity guarantees

**Problem**: The gateway uses an atomic Lua script for rate limiting. The auth
service uses separate INCR + EXPIRE (non-atomic, racy). Two independent rate
limit systems with different configs and key patterns.

**Files**:

- `services/gateway/internal/middleware/ratelimit.go` (Lua — correct)
- `services/auth/internal/ratelimit/ratelimit.go` (INCR + EXPIRE — racy)

**Task**:

- Extract the Lua-based rate limiter into `shared/pkg/ratelimit` so both
  gateway and auth service use the same atomic implementation.
- Or consolidate rate limiting at the gateway only and remove it from auth (if
  the gateway's per-IP + per-user limiting is sufficient).

---

### 4.7 `user.CreateUser` handles impossible `ErrUserNotFound` from create path

**Problem**: `CreateUser` checks for `repository.ErrUserNotFound` after calling
`CreateUser` on the repo. A create operation can't return "not found" — this
is likely a copy-paste artifact.

**Files**: `services/user/internal/service/user.go:40-41`

**Task**:

- Remove the `ErrUserNotFound` branch from `CreateUser`.
- Add a test that exercises the `ErrUserAlreadyExists` idempotency path
  (line 37-38) — this is correct and should be verified.

---

### 4.8 Room service allows removing the last owner

**Problem**: `RemoveMember` checks that the requester is the owner, but doesn't
prevent the owner from being the target. A room can end up with zero owners.

**Files**: `services/room/internal/service/room.go:207-225`

**Task**:

- Add a check: if `userID == room.CreatedBy` (or the user's role is owner),
  reject the removal with an error.
- Or prevent removal if it would leave the room with zero members of role
  `owner`.

---

### 4.9 Default room type silently applied for unknown enum values

**Problem**: The room handler's gRPC-to-domain mapping defaults unknown room
types to `RoomTypeGroup` without error. The gateway correctly rejects unknown
types, but the gRPC handler is lenient.

**Files**: `services/room/internal/handler/room.go:56-64`

**Task**:

- Return `CodeInvalidArgument` for unrecognized room types instead of silently
  defaulting.

---

## Phase 5 — Minor Cleanup

These are low-risk, high-polish items. Each is small and independent.

| #    | Issue                                                                                | File(s)                                            | Task                                                                                                |
| ---- | ------------------------------------------------------------------------------------ | -------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| 5.1  | `healthcheck` discards JSON encode errors                                            | `shared/pkg/healthcheck/healthcheck.go:31,39,44`   | Log the error (even if unlikely).                                                                   |
| 5.2  | No `HEALTHCHECK` in Dockerfiles                                                      | All `Dockerfile`s                                  | Add `HEALTHCHECK CMD` for orchestration.                                                            |
| 5.3  | No version/commit info baked into binaries                                           | All `Dockerfile`s                                  | Add `-ldflags "-X main.Version=..."` to `go build`.                                                 |
| 5.4  | UUID parse errors silently skipped in `GetRoomMembers` client                        | `services/message/internal/client/room.go:64`      | Log or return error for unparseable UUIDs.                                                          |
| 5.5  | Hard-coded shutdown timeout (5s), read header timeout (5s), hub channel buffer (256) | All `cmd/main.go`, `hub/hub.go:99-102`             | Make configurable via environment variables.                                                        |
| 5.6  | NATS `msg.Ack()` / `msg.Nak()` errors silently discarded                             | `services/websocket/internal/subscriber/events.go` | Log at warn level.                                                                                  |
| 5.7  | `timestampToString` returns empty string for nil instead of omitting field           | `services/gateway/internal/handler/common.go:7-12` | Return `*string` (nil = omit via `omitempty`), or leave as-is and ensure JSON tags use `omitempty`. |
| 5.8  | WebSocket handler doesn't validate `room_id` format on initial connect               | `services/websocket/internal/handler/ws.go:54`     | Validate UUID format before registering (like it does for subscribe at line 116).                   |
| 5.9  | No max length on ticket query param                                                  | `services/websocket/internal/handler/ws.go:34`     | Add length check before hitting Valkey.                                                             |
| 5.10 | Hub test uses `time.Sleep(1ms)` ignoring timeout param                               | `services/websocket/internal/hub/hub_test.go:65`   | Use channels or sync primitives for test synchronization.                                           |
| 5.11 | Protobuf `MESSAGE_TYPE_EVENT` has no send path                                       | `proto/efir/message/message.proto`                 | Either add `EventContent` to `SendMessageRequest` oneof or remove the dead enum value.              |
| 5.12 | Protobuf `participant_id` not validated for direct rooms                             | `proto/efir/room/room.proto:31`                    | Add buf/validate rule: required when `type == DIRECT`.                                              |

---

## Execution Order

Recommended order for tackling the phases:

```
Phase 0  →  Fix critical bugs first (4 items, small diffs)
Phase 1  →  Security & error handling (5 items, mostly shared/pkg + gateway)
Phase 2  →  Unidiomatic patterns (10 items, medium-sized refactors)
Phase 3  →  Operational hardening (6 items, cross-cutting)
Phase 4  →  Code quality & consistency (9 items, mix of small and large)
Phase 5  →  Minor cleanup (12 items, independent small fixes)
```

Within each phase, items are already ordered by impact. Work top-to-bottom.
