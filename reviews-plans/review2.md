# Efir Codebase Review — Second Pass

> Generated: 2026-03-25 | Branch: `main` | ~20k LOC across 7 services + shared
>
> This is a follow-up review after all items from the first review (2026-03-21) were
> addressed. The codebase has improved significantly. This document covers what was
> fixed, what remains, and new findings.

---

## Previous Review: Resolution Status

The first review contained **46 items** across 6 phases. Here is the final status:

| Phase                               | Items  | Fully Fixed | Partially Fixed | Open  |
| ----------------------------------- | ------ | ----------- | --------------- | ----- |
| Phase 0 — Critical Bugs             | 4      | 4           | 0               | 0     |
| Phase 1 — Security & Error Handling | 5      | 4           | 1               | 0     |
| Phase 2 — Unidiomatic Patterns      | 10     | 9           | 1               | 0     |
| Phase 3 — Operational Hardening     | 6      | 5           | 1               | 0     |
| Phase 4 — Code Quality              | 9      | 8           | 1               | 0     |
| Phase 5 — Minor Cleanup             | 12     | 10          | 2               | 0     |
| **Total**                           | **46** | **40**      | **6**           | **0** |

**Key fixes confirmed:**

- WebSocket duplicate registration eliminated
- RefreshToken now generates new pair before deleting old
- Rate limiter uses atomic Lua script in both gateway and auth
- gRPC connections stored and closeable (gateway defers close, message RoomClient has Close())
- Gateway errors are JSON with safe messages, no raw gRPC leakage
- Rate limit errors no longer leak email addresses
- Config durations parsed eagerly as `time.Duration`
- `Env.Validate()` called in every `Load()`
- All `main()` functions extracted to `run(ctx context.Context) error`
- Hub uses pure channel dispatch, no redundant mutex
- Ownership checks use `GetMemberRole()` from `room_members` table
- Reply validation properly distinguishes `ErrMessageNotFound` from DB errors
- gRPC graceful shutdown has 5-second timeout + force stop
- CGO_ENABLED=0 in all Dockerfiles
- HEALTHCHECK in all Dockerfiles
- Version/commit info baked into binaries via `-ldflags`
- Request ID propagation via `x-request-id` through middleware chain
- `sendToRoom` uses goroutines with write timeout (no blocking on slow clients)
- Logging interceptor conditionally includes error attribute
- `memberUserIDs()` helper extracted, slices pre-allocated

---

## New Findings

### 1. Bug — Hub `sendToRoom` captures loop variable in timeout branch

**Severity**: Bug (wrong connection closed on write timeout)
**File**: `services/websocket/internal/hub/hub.go:260`

The goroutine correctly captures `conn` as parameter `c` for the write path, but the timeout
branch references the outer `conn` loop variable instead of `c`:

```go
go func(c Conn) {
    defer wg.Done()
    done := make(chan error, 1)
    go func() {
        done <- c.WriteJSON(envelope)  // ✓ uses 'c'
    }()
    select {
    case err := <-done:
        if err != nil {
            // ...
            c.Close(StatusAbnormalClosure, "write error")  // ✓ uses 'c'
        }
    case <-time.After(writeTimeout):
        conn.Close(StatusAbnormalClosure, "write timeout")  // ✗ uses 'conn' (outer)
    }
}(conn)
```

On timeout, this closes whatever connection `conn` currently points to in the outer
loop — likely the last connection iterated, not the one that timed out.

**Task**: Change line 260 from `conn.Close(...)` to `c.Close(...)`.

---

### 2. Bug — WebSocket sends HTTP error after WebSocket upgrade

**Severity**: Bug (error never reaches client)
**File**: `services/websocket/internal/handler/ws.go:62-66`

After calling `websocket.Accept()` (line 53), the handler validates `room_id`. If validation
fails, it calls `http.Error()` — but the HTTP connection has already been upgraded to WebSocket.
The error is silently discarded; the client sees nothing.

```go
conn, err := websocket.Accept(w, r, nil)  // line 53: upgrade done
// ...
if initialRoomID != "" {
    if _, err := uuid.Parse(initialRoomID); err != nil {
        http.Error(w, "invalid room_id format", http.StatusBadRequest)  // never arrives
        return
    }
}
```

**Task**: Either validate `room_id` before `websocket.Accept()`, or send the error via the WebSocket connection (using `sendError`) and then close it.

---

### 3. Design — Auth service interface leaks repository types

**Severity**: Design smell
**File**: `services/auth/internal/service/auth.go:32-33`

The `AuthService` interface returns `*repository.Account`:

```go
Register(ctx, email, password string) (*repository.Account, *TokenPair, error)
Login(ctx, email, password string) (*repository.Account, *TokenPair, error)
```

This forces the handler to import the repository package, violating layer boundaries.
The room, message, and user services correctly define domain types in the service layer.
Auth should follow the same pattern.

**Task**: Define a service-level `Account` type (ID, Email, CreatedAt) and convert in the
service methods, like the other services do.

---

### 4. Inconsistency — Message service returns unwrapped errors from repository

**Severity**: Minor
**Files**: `services/message/internal/service/message.go:54, 84, 113, 121`

Most services consistently wrap errors with context (`fmt.Errorf("operation: %w", err)`).
The message service does this for some operations but returns raw errors in others:

```go
// Line 54: unwrapped
return nil, err  // from roomClient.IsMember

// Line 84: unwrapped
return nil, err  // from repo.CreateMessage

// Line 121: unwrapped
return nil, nil, err  // from repo.GetMessagesByRoomID
```

Compared to `GetMessageByID` which properly wraps: `fmt.Errorf("get message: %w", err)`.

**Task**: Add `fmt.Errorf` wrapping to the four unwrapped return sites for consistent
error context in logs and stack traces.

---

### 5. Design — Gateway wsauth endpoint has no rate limiting

**Severity**: Minor security concern
**File**: `services/gateway/cmd/main.go:125`

The wsauth handler is registered outside any middleware group:

```go
r.Group(func(r chi.Router) {
    r.Use(ipRateLimiter)
    authHandler.Register(r)
})

r.Group(func(r chi.Router) {
    r.Use(jwtMiddleware)
    r.Use(userRateLimiter)
    // ... protected routes
})

wsAuthHandler.Register(r)  // no rate limiter, no JWT middleware
```

An attacker could spam the ticket creation or validation endpoints without any rate limiting.

**Task**: Wrap `wsAuthHandler.Register(r)` in a group with `jwtMiddleware` and a rate limiter
(ticket creation requires auth; validation may not, but should still be rate-limited).

---

### 6. Smell — `ErrCannotRemoveSelf` defined but never used

**Severity**: Dead code
**File**: `services/room/internal/service/room.go:20`

The error `ErrCannotRemoveSelf` is defined alongside the TODO about self-removal support.
It's never returned anywhere.

**Task**: Either implement the self-leave feature or remove the dead error constant.
If deferring, add a comment linking it to the TODO at line 221.

---

### 7. Smell — `ErrRoomAlreadyExists` defined but never used (repository)

**Severity**: Dead code
**File**: `services/room/internal/repository/room.go:16`

Defined but never referenced. `CreateRoom` doesn't check for unique constraint violations because room IDs are UUIDs (no natural key conflicts).

**Task**: Remove the unused sentinel error.

---

### 8. Robustness — No request body size limit in gateway

**Severity**: Minor
**File**: `services/gateway/internal/handler/handler.go:36`

`ReadProto` calls `io.ReadAll(r.Body)` with no size limit. A malicious client could send
an arbitrarily large body to exhaust server memory.

```go
func ReadProto(r *http.Request, msg proto.Message) error {
    body, err := io.ReadAll(r.Body)  // unbounded
    // ...
}
```

**Task**: Use `io.LimitReader(r.Body, maxBodySize)` with a reasonable limit (e.g. 1 MB),
or set `http.Server.MaxHeaderBytes` and use `http.MaxBytesReader` for body.

---

### 9. Smell — `generateUsernameFromEmail` can produce empty string

**Severity**: Edge case
**File**: `services/user/internal/service/user.go:88-94`

If passed an empty string, `strings.Split("", "@")` returns `[""]`, so the function
returns `""`. The database `NOT NULL` constraint on `username` would reject this, producing
a generic DB error instead of a clear validation error.

**Task**: Add a guard: if the result is empty after split, fall back to a generated
username (e.g. `"user-" + uuid.New().String()[:8]`).

---

### 10. Consistency — Health check listener binding differs from gRPC listener

**Severity**: Cosmetic / operational awareness
**Files**: All `cmd/main.go` — health server binds to `127.0.0.1:8080`

The health server binds to `127.0.0.1:8080` (loopback only) while gRPC binds to
`0.0.0.0:<port>`. This is intentional and correct for Docker health checks via localhost.
However, if running outside Docker (e.g. local dev, bare-metal), external health probes
won't reach it.

**Task**: Make the health listen address configurable, defaulting to `127.0.0.1:8080`.

---

### 11. Minor — `Wrap()` still destroys the error chain

**Severity**: Known limitation (partially addressed in first review)
**File**: `services/shared/pkg/errors/errors.go:59-67`

`Wrap()` still calls `status.Error(c.ToGRPCCode(), err.Error())`, converting the error to
a string. This is inherent to the gRPC status model — gRPC errors are wire-serializable and
don't support Go error chains. The current code correctly avoids re-wrapping errors that
already have the right gRPC code (line 63-65).

The real mitigation is that errors are logged with full context at the handler boundary
before being converted to gRPC status. This is working correctly across all services.

**Status**: Accepted limitation, no action needed.

---

### 12. Minor — `context.Background()` used in WebSocket error logs

**Severity**: Cosmetic
**File**: `services/websocket/internal/handler/ws.go:88, 101, 103`

The `readPump` and `pingPump` functions use `context.Background()` in slog calls. These
goroutines outlive the HTTP request context, so `r.Context()` isn't available. However,
the user ID and connection info could be logged as structured fields for traceability.

**Task**: Add `"user_id"` field to error logs in `readPump` and `pingPump`.

---

### 13. Test Coverage — Summary of gaps

Testing has improved significantly. Here's what exists and what's missing:

| Service   | Handler Tests  | Service Tests | Repository Tests |                  Other                   |
| --------- | :------------: | :-----------: | :--------------: | :--------------------------------------: |
| Auth      |    20 cases    | 3+ functions  |        —         |                    —                     |
| Gateway   | 8 tests (auth) |       —       |        —         |        Config: 5, Middleware: 11         |
| Room      |    7+ cases    |   10+ cases   |        —         |                    —                     |
| Message   |      Yes       |      Yes      |  Yes (postgres)  |                    —                     |
| User      |    10 cases    |  7 functions  |        —         |                    —                     |
| WebSocket |       —        |       —       |        —         |              Hub: 10 tests               |
| Shared    |       —        |       —       |        —         | Middleware: 6, Healthcheck: 7, Logger: 5 |

**Notable gaps:**

- No tests for gateway handlers: user, room, message, wsauth, ratelimit middleware
- No repository tests for auth, room, or user services
- No WebSocket handler or subscriber tests
- No integration tests anywhere
- Auth service: `RefreshToken` method has no unit test
- Shared errors package has no tests

---

## Service Scores

Scores are on a 1-10 scale across five dimensions.

### Auth Service

| Dimension     |  Score  | Notes                                                                                                        |
| ------------- | :-----: | ------------------------------------------------------------------------------------------------------------ |
| Correctness   |    8    | Token refresh ordering fixed; rate limiting atomic. Minor: repository.Account leaks through interface.       |
| Code Quality  |    8    | Clean layers, consistent error handling. Small gap: `GetAccountByID` exists in repo interface but unused.    |
| Idiomatic Go  |    8    | run() pattern, proper error wrapping, context propagation. Duration config is clean.                         |
| Security      |    8    | Bcrypt hashing, rate limiting, no PII in errors. JWT lacks iss/aud claims (acceptable for internal service). |
| Test Coverage |    6    | Handler tests solid (20 cases). Service tests exist but `RefreshToken` untested. No repository tests.        |
| **Average**   | **7.6** |                                                                                                              |

### Gateway Service

| Dimension     |  Score  | Notes                                                                                                                      |
| ------------- | :-----: | -------------------------------------------------------------------------------------------------------------------------- |
| Correctness   |    8    | Clean HTTP-to-gRPC transcoding via protojson. Safe error responses. wsauth lacks rate limiting.                            |
| Code Quality  |    8    | Centralized WriteError/WriteProto helpers. Clean middleware chain. Request ID propagation works.                           |
| Idiomatic Go  |    9    | chi router, proper middleware groups, clean handler pattern. Timeout interceptor is elegant.                               |
| Security      |    7    | JWT validation solid. No request body size limit. wsauth not rate-limited.                                                 |
| Test Coverage |    6    | Auth handler (8), config (5), JWT middleware (11). No tests for user/room/message/wsauth handlers or ratelimit middleware. |
| **Average**   | **7.6** |                                                                                                                            |

### Room Service

| Dimension     |  Score  | Notes                                                                                                                    |
| ------------- | :-----: | ------------------------------------------------------------------------------------------------------------------------ |
| Correctness   |    9    | Role-based ownership checks via `GetMemberRole`. Owner removal prevented. Direct room dedup works.                       |
| Code Quality  |    8    | Clean 3-layer architecture. Domain types in service layer. `memberUserIDs` helper extracted. Two unused error constants. |
| Idiomatic Go  |    9    | Consistent error wrapping, interface-based DI, proper context propagation.                                               |
| Security      |    8    | Permission model is sound. Event publishing gracefully degrades with `event_lost` logging.                               |
| Test Coverage |    7    | Both handler and service tests with multiple cases. No repository tests.                                                 |
| **Average**   | **8.2** |                                                                                                                          |

### Message Service

| Dimension     |  Score  | Notes                                                                                                                       |
| ------------- | :-----: | --------------------------------------------------------------------------------------------------------------------------- |
| Correctness   |    8    | Reply validation is thorough (existence, deletion, same-room checks). Soft delete is correct. Unwrapped errors in 4 places. |
| Code Quality  |    8    | gRPC client has retry with exponential backoff. Keyset pagination is well-implemented. Type conversion is clean.            |
| Idiomatic Go  |    7    | Mostly consistent. Inconsistent error wrapping (some wrapped, some raw). Room client retry is clean generic implementation. |
| Security      |    8    | Membership checks on all read/write paths. Sender-only deletion.                                                            |
| Test Coverage |    7    | Handler, service, and repository tests all present. Good coverage for a data-heavy service.                                 |
| **Average**   | **7.6** |                                                                                                                             |

### User Service

| Dimension     |  Score  | Notes                                                                                                                              |
| ------------- | :-----: | ---------------------------------------------------------------------------------------------------------------------------------- |
| Correctness   |    8    | Idempotent create handles race via ON CONFLICT. Event-driven creation from auth is clean. Edge case: empty email → empty username. |
| Code Quality  |    8    | Simple, focused service. Clean NATS subscriber pattern.                                                                            |
| Idiomatic Go  |    8    | Proper layering, context propagation, error mapping. `toUser()` could use nil guard.                                               |
| Security      |    8    | No direct mutations from external input. All writes go through auth event.                                                         |
| Test Coverage |    6    | Handler (10 cases) and service (7 functions) tests. No subscriber tests. No repository tests.                                      |
| **Average**   | **7.6** |                                                                                                                                    |

### WebSocket Service

| Dimension     |  Score  | Notes                                                                                                                                    |
| ------------- | :-----: | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Correctness   |    6    | Two bugs: variable capture in sendToRoom timeout (closes wrong conn), HTTP error after WS upgrade.                                       |
| Code Quality  |    7    | Clean hub architecture. Read/write pumps with timeouts. Ping/pong implemented. Channel-based dispatch eliminates races.                  |
| Idiomatic Go  |    7    | Good use of channels and goroutines. `wsConnWrapper` with mutex for writes is correct pattern. Hardcoded timeouts could be configurable. |
| Security      |    7    | Ticket-based auth with GETDEL (one-time use). Ticket length validation. Missing rate limiting.                                           |
| Test Coverage |    5    | Only hub tests (10). No handler, subscriber, or config tests.                                                                            |
| **Average**   | **6.4** |                                                                                                                                          |

### Shared Packages

| Dimension     |  Score  | Notes                                                                                                  |
| ------------- | :-----: | ------------------------------------------------------------------------------------------------------ |
| Correctness   |    8    | Error mapping, middleware, health checks all work correctly. `FromError` handles edge cases properly.  |
| Code Quality  |    8    | Clean abstractions. Logger, healthcheck, middleware are well-designed. NATS helpers with retry.        |
| Idiomatic Go  |    9    | Empty-struct context keys, atomic.Bool for health state, slog integration.                             |
| Security      |    8    | Recovery interceptor catches panics. No sensitive data logged.                                         |
| Test Coverage |    6    | Logger (5), middleware (6), healthcheck (7) tested. Errors, NATS, and Valkey packages have zero tests. |
| **Average**   | **7.8** |                                                                                                        |

---

## Overall Score

| Dimension                 | Score | Rationale                                                                                                                                                                                     |
| ------------------------- | :---: | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Architecture**          |  8.5  | Clean microservice boundaries. Event-driven via NATS JetStream. Gateway pattern with protojson is practical. Proper layer separation in most services.                                        |
| **Correctness**           |  7.5  | Two bugs remain (hub variable capture, WS HTTP error after upgrade). All critical bugs from first review are fixed. Business logic is sound.                                                  |
| **Code Quality**          |  8.0  | Consistent patterns across services. Good use of interfaces, error wrapping, and dependency injection. Minor inconsistencies in error wrapping (message service). Small amount of dead code.  |
| **Idiomatic Go**          |  8.0  | `run()` pattern, `signal.NotifyContext`, channel-based hub, clean config loading, proper context propagation. Follows community conventions well.                                             |
| **Security**              |  7.5  | Rate limiting, JWT auth, safe error responses, bcrypt, ticket-based WS auth. Gaps: no body size limit, wsauth not rate-limited, no iss/aud in JWT.                                            |
| **Test Coverage**         |  6.0  | Handler and service tests exist for most services. Significant gaps in repository tests, integration tests, and some handler coverage.                                                        |
| **Operational Readiness** |  8.0  | Graceful shutdown with timeout across all services. Health checks with HEALTHCHECK in Dockerfiles. Request ID tracing. event_lost logging. CGO_ENABLED=0. Version info in binaries.           |
| **Infrastructure**        |  8.5  | Excellent Docker Compose setup (modular). Traefik routing. Full observability stack (OTEL, Prometheus, Loki, Tempo, Grafana). Task runner for workflow. CI pipeline. ADRs document decisions. |

### **Overall: 7.8 / 10**

This is a well-structured, thoughtfully designed codebase for its stage. The architecture
is sound, the code is mostly idiomatic, and the first review's critical issues were all
addressed. The main areas for improvement are:

1. Fix the two remaining bugs (hub variable capture, WS HTTP error after upgrade)
2. Increase test coverage, especially repository and integration tests
3. Address the minor security gaps (body size limit, wsauth rate limiting)
4. Clean up dead code and minor inconsistencies

---

## Prioritized Action Items

### Must Fix (bugs)

| #   | Issue                                               | File                                     | Effort    |
| --- | --------------------------------------------------- | ---------------------------------------- | --------- |
| 1   | Hub `sendToRoom` closes wrong connection on timeout | `websocket/internal/hub/hub.go:260`      | 1 line    |
| 2   | HTTP error sent after WebSocket upgrade             | `websocket/internal/handler/ws.go:62-66` | ~10 lines |

### Should Fix (security + correctness)

| #   | Issue                                    | File                                     | Effort   |
| --- | ---------------------------------------- | ---------------------------------------- | -------- |
| 3   | Add request body size limit to gateway   | `gateway/internal/handler/handler.go:36` | ~3 lines |
| 4   | Rate-limit wsauth endpoints              | `gateway/cmd/main.go:125`                | ~5 lines |
| 5   | Wrap unwrapped errors in message service | `message/internal/service/message.go`    | ~4 lines |
| 6   | Guard against empty username from email  | `user/internal/service/user.go:88-94`    | ~5 lines |

### Nice to Have (cleanliness)

| #   | Issue                                                       | File                                                  | Effort         |
| --- | ----------------------------------------------------------- | ----------------------------------------------------- | -------------- |
| 7   | Define domain `Account` type in auth service layer          | `auth/internal/service/auth.go`                       | ~30 lines      |
| 8   | Remove unused `ErrCannotRemoveSelf`, `ErrRoomAlreadyExists` | `room/internal/service/room.go`, `repository/room.go` | 2 lines        |
| 9   | Add user_id to WebSocket error logs                         | `websocket/internal/handler/ws.go`                    | ~5 lines       |
| 10  | Make health listen address configurable                     | All `cmd/main.go`                                     | ~10 lines each |

### Test Debt (track for next iteration)

- Auth: Add `RefreshToken` unit test
- Gateway: Add handler tests for user, room, message, wsauth
- Gateway: Add ratelimit middleware tests
- Shared: Add errors package tests
- All services: Add repository-level tests (even with testcontainers/pgx mock)
- WebSocket: Add handler and subscriber tests
