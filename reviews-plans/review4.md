# Efir Codebase Review — Fourth Pass

> Generated: 2026-03-31 | Branch: main | ~7.2k LOC source + ~5.1k LOC tests across 8 services + shared
>
> Follow-up to the third review (2026-03-28). This review was conducted post-MVP completion after
> manual end-to-end testing. All service endpoints reported working. This pass focuses on the full
> codebase with fresh eyes — bugs, security, concurrency, design, and operational hardening.

---

## Previous Review: Resolution Status

The third review contained 9 items (1 critical, 2 bugs, 3 should-fix, 3 nice-to-have).

| #   | Issue                                                 | Status                                                                                                                                         |
| --- | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | CreateTicket reads user ID from raw header, not JWT   | Needs verification — wsauth/handler.go still reads from header. If moved behind JWT middleware since R3, mark fixed. Otherwise still critical. |
| 2   | SendMessage handler msgType shadowing                 | Needs verification — check if := was changed to = on the media branch.                                                                         |
| 3   | Missing ErrNotMember checks in 3 room handler methods | Needs verification — check UpdateRoom, DeleteRoom, RemoveMember handlers.                                                                      |
| 4   | mapper.Enum panics on unknown values                  | Open — mapper.go:13-18 still panics. Caught by RecoveryInterceptor.                                                                            |
| 5   | Gateway getMessages no limit bounds                   | Mitigated — proto validation now clamps limit to [1, 100]. Gateway still doesn't clamp at its layer, but proto catches it downstream.          |
| 6   | WebSocket logs lack user_id                           | Fixed — readPump and pingPump now receive and log userID.                                                                                      |
| 7   | ErrCannotRemoveSelf dead code                         | Open — still defined at room/internal/service/room.go:20.                                                                                      |
| 8   | Use shared healthcheck in websocket                   | Open — websocket still uses standalone okHandler.                                                                                              |
| 9   | Document DeleteMessage membership behavior            | Open — no comment added.                                                                                                                       |

---

## New Findings

---

### Phase 0 — Critical / Security

---

#### 1. SECURITY — Refresh token replay window allows token theft

**Severity:** High  
**File:** `services/auth/internal/service/auth.go` (RefreshToken method)

The refresh flow performs three steps non-atomically:

1. Look up old token in Valkey → get userID
2. Generate new token pair → save new refresh token to Valkey
3. Delete old refresh token from Valkey

Between (a) and (c), both the old and new refresh tokens are valid simultaneously. A concurrent
request can reuse the old token to obtain a second valid token pair. This is a classic refresh
token replay vulnerability.

Additionally, if step (c) fails, the method returns an error but the new token pair is already
persisted in Valkey — an orphaned token that can never be cleaned up.

**Fix:** Use a Valkey Lua script or GETDEL to atomically read-and-delete the old token. Only then
generate and save the new pair.

---

#### 2. SECURITY — Empty JWT secret accepted at startup

**Severity:** High  
**File:** `services/auth/internal/config/config.go` (Auth.Secret field)

If the JWT_SECRET environment variable is unset or empty, the auth service starts and signs all
JWTs with `[]byte("")`. Any client can forge valid tokens by signing with an empty key. There is no
startup validation that Secret is non-empty or meets a minimum length.

Same concern applies to AccessTTL/RefreshTTL defaulting to 0 (tokens expire instantly) and
RateLimit.Requests defaulting to 0 (blocks all traffic).

**Fix:** Add a `Validate()` method on Config that checks Secret length >= 32, TTLs > 0, and
RateLimit.Requests > 0. Call it in main() before starting the server.

---

#### 3. SECURITY — Rate limiter IP extraction trusts RemoteAddr behind proxy

**Severity:** High (deployment-dependent)  
**File:** `services/gateway/internal/middleware/ratelimit.go:22-25`

The IP rate limiter extracts the client IP from `r.RemoteAddr`:

```go
ip, _, err := net.SplitHostPort(r.RemoteAddr)
```

Behind Traefik (which this project uses), RemoteAddr is always Traefik's IP. All clients share
one rate limit bucket. A single user can exhaust the limit for everyone, or alternatively the
rate limiter is effectively disabled (the proxy makes thousands of requests from one IP).

**Fix:** Use chi's `middleware.RealIP` or parse `X-Forwarded-For` / `X-Real-IP` with a trusted proxy
allowlist. Traefik already sets these headers.

---

#### 4. SECURITY — ReadProto body size check is dead code — oversized bodies silently truncated

**Severity:** Medium  
**File:** `services/gateway/internal/handler/handler.go:86-98`

The body-too-large detection branch can never execute:

```go
body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
if err != nil {
    if stderrors.Is(err, io.EOF) && int64(len(body)) >= maxBodySize {
        return errors.CodeInvalidArgument.Error("request body too large")
    }
    return err
}
```

`io.LimitReader` returns a reader that yields at most `maxBodySize` bytes. When the limit is reached,
the underlying LimitedReader returns `io.EOF`, which `io.ReadAll` treats as successful completion —
err is nil, not io.EOF. The oversized body is silently truncated and passed to protojson.Unmarshal,
producing either a confusing parse error or a silently incomplete message.

**Fix:** Read maxBodySize+1 bytes, then check if `len(body) > maxBodySize`.

---

#### 5. SECURITY — PII (email addresses) logged in plaintext

**Severity:** Medium  
**Files:** `services/auth/internal/service/auth.go` (rate limit warnings), `services/user/internal/nats/subscriber.go:85`

User email addresses appear in log output via rate-limit warnings and user-creation events. Under
GDPR or similar regulation, log aggregation systems become PII processors.

**Fix:** Log a hash/prefix of the email for correlation, not the full address.

---

### Phase 1 — Bugs

---

#### 6. BUG — Hub sendToRoom blocks the event loop; potential deadlock under load

**Severity:** High  
**File:** `services/websocket/internal/hub/hub.go:229-250`

`sendToRoom` spawns goroutines for each connection, then calls `wg.Wait()`. During the wait, the hub's
`Run()` loop is blocked — no registers, unregisters, disconnects, or other broadcasts can proceed.

The `wsConnWrapper.WriteJSON` has a write timeout (so it won't block forever), but for a room with
many connections, the hub is stalled for the duration of ALL concurrent writes. If a write fails
and `c.Close()` triggers `Disconnect()` from `readPump`, and the disconnect channel buffer is full, the
`readPump` goroutine blocks waiting to send on `h.disconnect`, while the hub is blocked on `wg.Wait()` —
a deadlock.

**Fix:** Move writes off the hub goroutine entirely. Have sendToRoom fan out without waiting, or use
a per-connection outbound channel that the hub dispatches to non-blockingly.

---

#### 7. BUG — Duplicate room subscriptions cause duplicate message delivery

**Severity:** Medium  
**File:** `services/websocket/internal/hub/hub.go:158-172`

`addConn` unconditionally appends the connection to the room's user list:

```go
h.rooms[roomID][userID] = append(userConns, conn)
```

If a client sends multiple "subscribe" messages for the same room, the same Conn is appended
repeatedly. Every broadcast writes to that connection N times, causing duplicate messages on
the client.

**Fix:** Check if conn is already in userConns before appending.

---

#### 8. BUG — ReadPump timeout kills idle-but-alive WebSocket connections

**Severity:** Medium  
**File:** `services/websocket/internal/handler/ws.go:95-108`

The read loop uses a fixed timeout per read:

```go
readCtx, cancel := context.WithTimeout(context.Background(), readTimeout)
_, msg, err := conn.Read(readCtx)
```

If the client is idle (normal in a chat app — no messages to send) but alive (responding to
pings), the read times out and the connection is terminated. The pingPump correctly detects dead
connections, so the readPump should use a much longer timeout or no timeout at all.

**Fix:** Use `context.Background()` for reads (let pingPump handle liveness), or reset the timeout
on every pong receipt.

---

#### 9. BUG — WebSocket ReadLimit checked after full message is buffered in memory

**Severity:** Medium  
**File:** `services/websocket/internal/handler/ws.go:111`

```go
if len(msg) > int(h.cfg.WebSocket.ReadLimit) {
```

The message is fully read into memory before the size check. A malicious client can send an
arbitrarily large frame that gets fully buffered. The nhooyr/websocket library supports
`conn.SetReadLimit()` which rejects oversized messages during the read itself.

**Fix:** Call `ws.SetReadLimit(readLimit)` on the raw `*websocket.Conn` after Accept.

---

#### 10. BUG — Username collision causes permanent NATS retry loop

**Severity:** Medium  
**File:** `services/user/internal/service/user.go:31-48`, `services/user/internal/repository/user.go`

`generateUsernameFromEmail` extracts the local part. Two users with alice@gmail.com and
alice@yahoo.com produce the same username "alice". The repository's `ON CONFLICT (id) DO NOTHING`
only handles ID conflicts, not username conflicts. The resulting unique constraint violation
surfaces as an unhandled error, causing the NATS subscriber to Nak and retry forever until
MaxDeliver is exhausted.

**Fix:** Handle the username unique constraint violation — either append a random suffix on conflict,
or add ON CONFLICT handling for the username column.

---

#### 11. BUG — TOCTOU race on Register allows duplicate 500 errors

**Severity:** Medium  
**File:** `services/auth/internal/service/auth.go:92-98`

Register checks `GetAccountByEmail` then calls `CreateAccount` in two separate queries. Two concurrent
registrations with the same email both pass the check. The second INSERT hits the unique constraint,
but the error is not caught as `ErrAccountAlreadyExists` — it surfaces as a 500 Internal.

**Fix:** Handle the Postgres unique violation error (code 23505) from `CreateAccount` and map it to
`ErrAccountAlreadyExists`. Or drop the pre-check entirely and rely on the constraint.

---

#### 12. BUG — Direct room creation has no constraint preventing duplicates

**Severity:** Medium  
**File:** `services/room/internal/service/room.go:55-82`

`CreateRoom` checks `GetDirectRoomByUsers` then creates the room in separate queries with no
transaction. Two concurrent calls for the same user pair can both pass the existence check.
There is no database-level unique constraint on direct rooms.

**Fix:** Add a partial unique index on the rooms table for direct rooms, or wrap the check-and-create
in a serializable transaction.

---

#### 13. BUG — NATS Nak on unmarshal errors causes poison message retry loops

**Severity:** Low-Medium  
**Files:** `services/user/internal/nats/subscriber.go:59-74`, `services/websocket/internal/subscriber/events.go:101-106, 137-142, 172-177`

When a NATS message has permanently malformed JSON, `Nak()` causes redelivery. Since the data is
inherently unparseable, every redelivery fails identically. This repeats MaxDeliver times per
poison message, generating log noise and consuming redelivery budget.

**Fix:** Use `msg.Term()` for permanent/non-retryable errors (bad JSON, invalid UUID). Reserve
`msg.Nak()` for transient errors (DB timeout, service unavailable).

---

#### 14. BUG — AddMember returns 500 for duplicate members

**Severity:** Low  
**File:** `services/room/internal/service/room.go:195-198`

The repository returns `ErrMemberAlreadyExists` for duplicate additions (`ON CONFLICT DO NOTHING`),
but the service layer doesn't handle it — it wraps it as a generic error that becomes a 500.
Should either return success (idempotent) or return `CodeAlreadyExists`.

---

#### 15. BUG — Auth handler endpoints skip request-ID propagation to gRPC

**Severity:** Low  
**File:** `services/gateway/internal/handler/auth/handler.go:32, 46, 60, 73`

All auth handler methods call the gRPC client with `r.Context()` directly, while every other
handler uses `middleware.InjectRequestIDToOutgoingContext()`. Distributed tracing is broken for
register, login, logout, and refresh endpoints.

---

### Phase 2 — Design & Hardening

---

#### 16. DESIGN — Rate limiter fails closed on Valkey errors

**Severity:** Medium  
**File:** `services/gateway/internal/middleware/ratelimit.go:31-33`

If Valkey is temporarily unavailable, every request gets a 500. This means a Valkey blip takes
down the entire gateway. Rate limiters should typically fail open — if you can't check the limit,
allow the request through with a warning log.

**Fix:** On Valkey error, log a warning and call `next.ServeHTTP(w, r)`.

---

#### 17. DESIGN — No authentication on gRPC services; trust self-asserted identity

**Severity:** Medium (architecture-level)  
**Files:** All gRPC handlers (room, message, user) accept requester_id/sender_id from the request

All gRPC services trust the caller-supplied user ID from the request body. This is correct IF
the gateway is the only caller and it always injects the authenticated user ID. But:

- There is no mTLS or authentication between services
- If any service is accidentally exposed, full impersonation is possible
- The shared middleware's `UserIDInterceptor` reads `x-user-id` from metadata without validation

This is an accepted architecture decision for now but should be documented as a Module 2 concern
(sidecar PEP).

---

#### 18. DESIGN — Direct rooms have no invariant enforcement

**Severity:** Medium  
**File:** `services/room/internal/service/room.go`

Multiple invariant violations are possible for direct rooms:

- A user can create a direct room with themselves (`createdBy == participantID` not checked)
- Members can be added to a direct room via `AddMember` (no room-type check), breaking the 2-member invariant
- Members can be removed from a direct room via `RemoveMember`, leaving a 1-member "direct" room
- CreateRoom allows creating a direct room without a participant (`participantID == uuid.Nil`)

**Fix:** Add room-type guards in AddMember and RemoveMember. Validate `createdBy != participantID`
and `participantID != Nil` for direct rooms in CreateRoom.

---

#### 19. DESIGN — Event publishing inconsistency in room service

**Severity:** Low-Medium  
**File:** `services/room/internal/service/room.go`

Inconsistent behavior across mutations:

- UpdateRoom: logs event failure and returns success
- AddMember/RemoveMember: event failure causes the entire operation to return error (even though the DB mutation already succeeded)
- CreateRoom: publishes no events at all
- DeleteRoom: publishes no events at all

Downstream consumers tracking room state will miss creates and deletes entirely, and may receive
spurious errors for membership changes.

**Fix:** Standardize on log-and-continue for all event publishing failures (match UpdateRoom's
pattern). Add events for CreateRoom and DeleteRoom.

---

#### 20. DESIGN — RecoveryInterceptor does not capture stack traces

**Severity:** Low-Medium  
**File:** `services/shared/pkg/middleware/middleware.go:86-101`

When a panic is recovered, only the panic value is logged, not the stack trace. Production panics
become very difficult to diagnose.

**Fix:** Add `"stack", string(debug.Stack())` to the log call.

---

#### 21. DESIGN — User service cannot clear optional fields to NULL

**Severity:** Low  
**File:** `services/user/internal/repository/user.go:124-129`

The COALESCE pattern means passing NULL = "keep old value". There is no way for a user to remove
their avatar_url or bio once set. The proto uses optional string which can distinguish present-
empty from absent, but the repository treats both as "don't change".

**Fix:** Use a separate "clear" sentinel or check proto field presence to distinguish "set to empty"
from "don't change".

---

#### 22. DESIGN — MustGetUserID silently returns empty string

**Severity:** Low  
**File:** `services/gateway/internal/middleware/auth.go:85-88`

In Go, `Must*` functions conventionally panic on failure. This one returns `""` on missing context
value. If a handler is accidentally mounted outside the JWT middleware, empty user IDs flow into
gRPC requests silently. The wsauth handler defensively checks for this, but user/room/message
handlers do not.

**Fix:** Either make it panic (true "must" semantics) or rename to `GetUserIDOrEmpty` and add
defensive checks at call sites.

---

#### 23. DESIGN — Sub-second rate limit window produces TTL=0, causing permanent lockout

**Severity:** Low (config-dependent)  
**Files:** `services/gateway/internal/middleware/ratelimit.go:27`, `services/auth/internal/ratelimit/ratelimit.go:53`

`int(window.Seconds())` truncates sub-second durations to 0. A TTL of 0 means the rate limit key
never expires, and the counter increments forever — eventually blocking all traffic permanently.

**Fix:** Use `int(math.Ceil(window.Seconds()))` or validate `window >= 1s` at config time.

---

#### 24. DESIGN — Zero-value config timeouts create operational risks

**Severity:** Low  
**File:** `services/shared/pkg/config/environment.go:29-33`

TimeoutsConfig fields default to 0 if unconfigured:

- `Shutdown = 0` → `context.WithTimeout` expires immediately, no graceful drain
- `ReadHeaderTimeout = 0` → slowloris DoS vector (Go's gosec G112)
- `GRPCGraceful = 0` → GracefulStop gets no time

**Fix:** Apply sensible defaults (e.g., Shutdown=15s, ReadHeader=5s, GRPCGraceful=10s) when zero.

---

### Phase 3 — Code Quality

---

#### 25. QUALITY — video_sticker DB enum has no Go code mapping

**Severity:** Low-Medium  
**File:** `services/message/migrations/000001_create_messages_table.sql`, `services/message/internal/repository/message.go`

The migration defines a 'video_sticker' enum value, but no `MessageTypeVideoSticker` constant
exists in the Go code. If such a row appears in the database, `unmarshalContent` returns "unknown
message type" — silently corrupting the response.

---

#### 26. QUALITY — EventContent type missing from handler proto mapper

**Severity:** Low-Medium  
**File:** `services/message/internal/handler/mapper.go`

`messageTypeToProto` and `protoToMessageTypeMap` have no entry for `service.MessageTypeEvent` / "event".
Event-type messages are returned to clients as `MESSAGE_TYPE_UNSPECIFIED` — silently wrong data.

---

#### 27. QUALITY — Dead wsauth.Register() method

**Severity:** Low  
**File:** `services/gateway/internal/handler/wsauth/handler.go:29-31`

`Register()` is defined but never called from main.go. It also only registers ValidateTicket,
missing CreateTicket. If someone refactors to use it, they silently lose an endpoint.

---

#### 28. QUALITY — Redundant username index in user migration

**Severity:** Negligible  
**File:** `services/user/migrations/20260317142256_init.sql:13`

`CREATE INDEX idx_users_username` on a column that already has a UNIQUE constraint (which creates
an implicit unique index). Wastes disk and slows writes slightly.

---

#### 29. QUALITY — Duplicate reply-preview construction in message repository

**Severity:** Low  
**File:** `services/message/internal/repository/postgres.go:208-242 and 315-349`

~35 lines of reply-preview logic copy-pasted between `GetMessagesByRoomID` and `GetMessageByID`.
Extract a `buildReplyPreview` helper.

---

#### 30. QUALITY — NATS ProvisionConsumerWithRetry can spin-loop on zero interval

**Severity:** Low  
**File:** `services/shared/pkg/nats/nats.go:60-78`

If retryInterval is zero, `time.After(0)` fires immediately, creating a tight CPU loop. Add a
minimum floor.

---

#### 31. QUALITY — Cursor pagination breaks if cursor message is hard-deleted

**Severity:** Low  
**File:** `services/message/internal/repository/postgres.go:150-154`

The cursor subquery returns NULL if the cursor message was deleted, causing the row comparison
to return zero results. Pagination silently breaks.

---

### Phase 4 — Positive Observations

These patterns are done well and worth preserving:

- Proto validation with buf.validate is thorough — email format, password length, limit ranges,
  required fields, enum defined_only. This catches a whole class of invalid-input bugs.
- wsConnWrapper with mutex for concurrent writes is correct and clean.
- Ticket-based WebSocket auth with GETDEL is a solid one-time-use token pattern.
- run() function pattern with signal.NotifyContext is idiomatic and testable.
- Error model with StatusError + Unwrap() preserves Go error chains while hiding internals from
  clients. Good security/debuggability tradeoff.
- Keyset pagination in the message service is the right choice for real-time chat.
- Event-driven user creation via NATS decouples auth from user profile management cleanly.
- Configurable health listen addresses, body size limits, and structured logging throughout.
- 14 ADRs documenting architectural decisions — excellent for a project of this size.

---

## Test Coverage Assessment

No significant change from R3. Current state:

| Service   | Handler Tests | Service Tests | Repository Tests | Config Tests | Other                                                |
| --------- | ------------- | ------------- | ---------------- | ------------ | ---------------------------------------------------- |
| Auth      | 20+ cases     | 3+ functions  | —                | Yes          | —                                                    |
| Gateway   | Auth: 8       | —             | —                | Yes          | JWT middleware: 11                                   |
| Room      | 7+ cases      | 10+ cases     | —                | Yes          | —                                                    |
| Message   | Yes           | Yes           | Yes (postgres)   | Yes          | —                                                    |
| User      | 10 cases      | 7 functions   | —                | Yes          | —                                                    |
| WebSocket | —             | —             | —                | Yes          | Hub: 10                                              |
| Shared    | —             | —             | —                | —            | Middleware: 6, Healthcheck: 7, Logger: 5, Errors: 20 |

**Key gaps (unchanged from R3):**

- No gateway handler tests for user, room, message, wsauth endpoints
- No gateway ratelimit middleware tests
- No repository tests for auth, room, or user services
- No WebSocket handler or subscriber tests
- No integration tests
- Auth RefreshToken method has no unit test
- Shared mapper and NATS packages have no tests

---

## Service Scores

---

### Auth Service

| Dimension     | Score   | Delta    | Notes                                                                                                                |
| ------------- | ------- | -------- | -------------------------------------------------------------------------------------------------------------------- |
| Correctness   | 7.5     | -1.5     | Refresh token replay window. TOCTOU race on Register. Previously unflagged issues now surfaced with deeper analysis. |
| Code Quality  | 8.5     | —        | Clean 3-layer architecture. Consistent error wrapping.                                                               |
| Idiomatic Go  | 9       | —        | run() pattern, proper DI, context propagation.                                                                       |
| Security      | 6.5     | -1.5     | Empty JWT secret accepted. PII in logs. Refresh replay. Rate limit key collision via email.                          |
| Test Coverage | 6.5     | —        | Handler (20+), service (3+), config. RefreshToken untested. No repo tests.                                           |
| **Average**   | **7.6** | **-0.6** |                                                                                                                      |

---

### Gateway Service

| Dimension     | Score   | Delta | Notes                                                                                                                                 |
| ------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Correctness   | 7       | —     | ReadProto body size check is dead code. R3 items need verification.                                                                   |
| Code Quality  | 8.5     | —     | Centralized helpers. Request ID propagation (except auth handlers).                                                                   |
| Idiomatic Go  | 9       | —     | chi router, clean middleware groups, protojson transcoding.                                                                           |
| Security      | 5.5     | —     | IP rate limiter useless behind proxy. Fail-closed rate limiting. MustGetUserID footgun. CreateTicket status still needs verification. |
| Test Coverage | 6       | —     | Auth handler (8), config, JWT middleware (11). Most handlers untested.                                                                |
| **Average**   | **7.2** | —     |                                                                                                                                       |

---

### Room Service

| Dimension     | Score   | Delta    | Notes                                                                                                   |
| ------------- | ------- | -------- | ------------------------------------------------------------------------------------------------------- |
| Correctness   | 7.5     | -0.5     | Direct room race condition. No direct-room invariant enforcement. AddMember returns 500 for duplicates. |
| Code Quality  | 8.5     | —        | Clean layering. Domain types. Dead ErrCannotRemoveSelf still present.                                   |
| Idiomatic Go  | 9       | —        | Consistent error wrapping, interface DI, proper context propagation.                                    |
| Security      | 8       | -0.5     | Permission model sound but TOCTOU gaps in all mutating methods.                                         |
| Test Coverage | 7       | —        | Handler and service tests. No repository tests.                                                         |
| **Average**   | **8.0** | **-0.2** |                                                                                                         |

---

### Message Service

| Dimension     | Score   | Delta | Notes                                                                                                            |
| ------------- | ------- | ----- | ---------------------------------------------------------------------------------------------------------------- |
| Correctness   | 7.5     | +0.5  | video_sticker/event type mapping gaps. Cursor pagination edge case. Assuming msgType shadowing from R3 is fixed. |
| Code Quality  | 8       | -0.5  | Duplicated reply-preview logic. marshalContent switch is redundant.                                              |
| Idiomatic Go  | 8       | —     | Room client retry is clean. Keyset pagination correct.                                                           |
| Security      | 8       | —     | Membership checks on reads. Sender-only deletion.                                                                |
| Test Coverage | 7.5     | —     | Handler, service, and repository tests all present. Best coverage.                                               |
| **Average**   | **7.8** | —     |                                                                                                                  |

---

### User Service

| Dimension     | Score   | Delta    | Notes                                                                                                 |
| ------------- | ------- | -------- | ----------------------------------------------------------------------------------------------------- |
| Correctness   | 7.5     | -1.5     | Username collision causes permanent retry loop. Cannot clear optional fields. Nak on poison messages. |
| Code Quality  | 8.5     | —        | Simple, focused. Clean NATS subscriber pattern.                                                       |
| Idiomatic Go  | 8.5     | —        | Proper layering, context propagation, error mapping.                                                  |
| Security      | 8.5     | —        | No direct mutations from external input. PII logged (email).                                          |
| Test Coverage | 6.5     | —        | Handler (10), service (7), config. No subscriber or repo tests.                                       |
| **Average**   | **7.9** | **-0.3** |                                                                                                       |

---

### WebSocket Service

| Dimension     | Score   | Delta    | Notes                                                                                                                                |
| ------------- | ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Correctness   | 6.5     | -2.0     | Hub blocks event loop on broadcast. Duplicate subscriptions. Read timeout kills idle connections. ReadLimit checked after buffering. |
| Code Quality  | 7.5     | —        | Hub architecture is clean conceptually. Poison message handling wrong.                                                               |
| Idiomatic Go  | 7.5     | —        | Channel-based dispatch. wsConnWrapper with mutex correct.                                                                            |
| Security      | 7.5     | —        | Ticket-based auth with GETDEL. Memory exhaustion via large frames.                                                                   |
| Test Coverage | 5.5     | —        | Only hub tests (10) and config. No handler, subscriber tests.                                                                        |
| **Average**   | **6.9** | **-0.4** |                                                                                                                                      |

---

### Shared Packages

| Dimension     | Score   | Delta    | Notes                                                                                            |
| ------------- | ------- | -------- | ------------------------------------------------------------------------------------------------ |
| Correctness   | 8.5     | —        | Error mapping correct. StatusError preserves chain. Fallback behavior sound.                     |
| Code Quality  | 8       | -0.5     | mapper.Enum still panics. ProvisionConsumerWithRetry spin-loop risk. Zero-value config timeouts. |
| Idiomatic Go  | 9       | —        | Empty-struct context keys, atomic.Bool, slog, generics in mapper.                                |
| Security      | 8       | -0.5     | Recovery interceptor lacks stack traces. No config validation.                                   |
| Test Coverage | 7       | —        | Errors (20), logger (5), middleware (6), healthcheck (7). Mapper, NATS, valkey untested.         |
| **Average**   | **8.1** | **-0.2** |                                                                                                  |

---

## Overall Score

| Dimension             | Score | Delta from R3 | Rationale                                                                                                                              |
| --------------------- | ----- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Architecture          | 8.5   | —             | Clean microservice boundaries. Event-driven NATS. Gateway pattern. Proper layer separation. 14 ADRs.                                   |
| Correctness           | 7     | -0.5          | Hub event-loop blocking. Refresh token replay. Username collision. Duplicate direct rooms. ReadProto dead code. Multiple TOCTOU races. |
| Code Quality          | 8.5   | —             | Consistent patterns. Domain types. Proto validation. Minor dead code and duplication.                                                  |
| Idiomatic Go          | 8.5   | —             | run() pattern, signal.NotifyContext, channel hub, clean config, DI.                                                                    |
| Security              | 6     | -0.5          | Empty JWT secret. Refresh replay. Rate limiter useless behind proxy. PII in logs. No service-to-service auth. ReadProto truncation.    |
| Test Coverage         | 6.5   | —             | Unchanged from R3. Testing plan exists but not yet executed.                                                                           |
| Operational Readiness | 8.5   | —             | Graceful shutdown. Configurable health. Request IDs. event_lost logging. CGO_ENABLED=0. HEALTHCHECK. Version info.                     |
| Infrastructure        | 8.5   | —             | Docker Compose. Traefik. OTEL + Prometheus + Loki + Tempo + Grafana. Task runner. CI pipeline. Migration tooling.                      |

**Overall: 7.8 / 10 (down from 7.9)**

The slight decrease reflects deeper analysis uncovering issues that were present before but not
previously caught (refresh token replay, hub blocking, username collision, empty JWT secret). The
architecture, code quality, and idiomatic Go remain strong. The main areas pulling the score down
are security hardening and test coverage.

Fixing items #1-4 (security) and #6-9 (hub/WS bugs) would push the score to ~8.3. Adding config
validation (#2, #24) and rate limiter proxy awareness (#3) would push security to ~7.5+.

---

## Prioritized Action Items

---

### Must Fix (security)

| #   | Issue                                                                     | File                                             | Effort                           |
| --- | ------------------------------------------------------------------------- | ------------------------------------------------ | -------------------------------- |
| 1   | Refresh token replay — atomic read-and-delete needed                      | `auth/internal/service/auth.go` (RefreshToken)   | ~15 lines (Lua script or GETDEL) |
| 2   | Empty JWT secret accepted — add config validation                         | `auth/internal/config/config.go` + `cmd/main.go` | ~20 lines                        |
| 3   | Rate limiter ignores X-Forwarded-For — useless behind Traefik             | `gateway/internal/middleware/ratelimit.go`       | ~10 lines                        |
| 4   | ReadProto body size check dead code — oversized bodies silently truncated | `gateway/internal/handler/handler.go:86-98`      | ~5 lines                         |

---

### Must Fix (bugs)

| #   | Issue                                                          | File                                                   | Effort               |
| --- | -------------------------------------------------------------- | ------------------------------------------------------ | -------------------- |
| 6   | Hub sendToRoom blocks event loop — deadlock under load         | `websocket/internal/hub/hub.go:229-250`                | ~30 lines (redesign) |
| 9   | WS ReadLimit checked after full buffer — memory exhaustion DoS | `websocket/internal/handler/ws.go:111`                 | ~2 lines             |
| 10  | Username collision → permanent retry loop                      | `user/internal/service/user.go` + `repository/user.go` | ~10 lines            |
| 11  | Register TOCTOU → 500 on concurrent duplicate emails           | `auth/internal/service/auth.go` + `repository/auth.go` | ~10 lines            |

---

### Should Fix

| #   | Issue                                                         | File                                              | Effort    |
| --- | ------------------------------------------------------------- | ------------------------------------------------- | --------- |
| 7   | Duplicate room subscriptions cause duplicate message delivery | `websocket/internal/hub/hub.go:158-172`           | ~5 lines  |
| 8   | Read timeout kills idle WS connections                        | `websocket/internal/handler/ws.go:95-108`         | ~5 lines  |
| 12  | Direct room race condition — no unique constraint             | `room/internal/service/room.go:55-82` + migration | ~15 lines |
| 13  | NATS Nak on poison messages — use Term()                      | `user subscriber` + `websocket subscriber`        | ~6 lines  |
| 16  | Rate limiter fails closed — Valkey outage kills gateway       | `gateway/internal/middleware/ratelimit.go:31-33`  | ~5 lines  |
| 18  | Direct room invariants not enforced                           | `room/internal/service/room.go`                   | ~15 lines |
| 19  | Event publishing inconsistency + missing create/delete events | `room/internal/service/room.go`                   | ~20 lines |
| 20  | RecoveryInterceptor no stack trace                            | `shared/pkg/middleware/middleware.go:86-101`      | ~3 lines  |
| 24  | Zero-value config timeouts — add defaults                     | `shared/pkg/config/environment.go:29-33`          | ~10 lines |

---

### Nice to Have

| #   | Issue                                         | File                                               | Effort    |
| --- | --------------------------------------------- | -------------------------------------------------- | --------- |
| 5   | PII (email) in logs                           | `auth/service`, `user/subscriber`                  | ~5 lines  |
| 14  | AddMember returns 500 for duplicate members   | `room/internal/service/room.go:195-198`            | ~5 lines  |
| 15  | Auth handlers skip request-ID propagation     | `gateway/internal/handler/auth/handler.go`         | ~4 lines  |
| 21  | Cannot clear optional user fields to NULL     | `user/internal/repository/user.go:124-129`         | ~10 lines |
| 22  | MustGetUserID returns "" instead of panicking | `gateway/internal/middleware/auth.go:85-88`        | ~3 lines  |
| 25  | video_sticker DB enum has no Go mapping       | `message/migrations` + `repository/message.go`     | ~5 lines  |
| 26  | EventContent missing from handler mapper      | `message/internal/handler/mapper.go`               | ~2 lines  |
| 27  | Dead wsauth.Register() method                 | `gateway/internal/handler/wsauth/handler.go:29-31` | Delete    |
| 29  | Duplicate reply-preview logic                 | `message/internal/repository/postgres.go`          | ~15 lines |
| 30  | ProvisionConsumerWithRetry spin-loop risk     | `shared/pkg/nats/nats.go:60-78`                    | ~3 lines  |

---

## Recommended Fix Order

1. Config validation (#2, #24) — prevents silent misconfiguration, quick wins
2. Refresh token atomicity (#1) — security, contained to one method
3. ReadProto body size fix (#4) — 5-line fix, prevents silent truncation
4. Hub sendToRoom redesign (#6) + ReadLimit (#9) — WebSocket stability
5. Rate limiter proxy awareness (#3) + fail-open (#16) — gateway resilience
6. Username collision (#10) + Register TOCTOU (#11) — data integrity
7. Direct room invariants (#18) + race condition (#12) — room service correctness
8. NATS poison message handling (#13) — operational hygiene
9. Remaining should-fix and nice-to-have items
