# Testing Plan — Comprehensive Coverage Strategy

> Generated: 2026-03-28 | Branch: `main`
>
> This plan defines the full testing strategy for the efir codebase. It covers
> unit tests, integration tests with testcontainers, and the testing
> infrastructure needed to support it all. Designed to be implemented after all
> bug fixes from the implementation plan are complete.

---

## Table of Contents

1. [Principles](#1-principles)
2. [Architecture Overview](#2-architecture-overview)
3. [Dependencies to Add](#3-dependencies-to-add)
4. [Shared Test Infrastructure](#4-shared-test-infrastructure)
5. [Service-by-Service Test Plan](#5-service-by-service-test-plan)
6. [Integration Tests](#6-integration-tests)
7. [CI Pipeline Updates](#7-ci-pipeline-updates)
8. [Coverage Targets](#8-coverage-targets)

---

## 1. Principles

- **Table-driven tests everywhere** — every test function uses `[]struct` with
  named cases and `t.Run`.
- **`require` for preconditions, `assert` for assertions** — fail fast on setup
  errors, continue collecting assertion failures.
- **No `time.Sleep` for synchronization** — use channels, `sync.WaitGroup`,
  `t.Deadline()`, or `require.Eventually`.
- **Testcontainers for repository tests** — real Postgres, real Valkey. No
  in-memory fakes for data stores.
- **Mockery-generated mocks for service boundaries** — never mock the thing
  you're testing, only its dependencies.
- **Tests document behavior** — test names read as specifications:
  `TestSendMessage_ReplyToDeletedMessage_ReturnsInvalidReplyTarget`.
- **Parallel where safe** — `t.Parallel()` on all tests that don't share
  mutable state. Repository tests use per-test schemas or transactions for
  isolation.
- **No test pollution** — each test creates its own fixtures. No shared global
  state between test functions.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                    Test Layers                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Unit Tests (mock dependencies)                     │
│  ├── Handler tests   — mock service interface       │
│  ├── Service tests   — mock repo + clients + pub    │
│  └── Shared pkg tests — standalone logic            │
│                                                     │
│  Repository Tests (testcontainers)                  │
│  ├── Postgres repos  — real DB, migrations applied  │
│  └── Valkey repos    — real Valkey instance          │
│                                                     │
│  Integration Tests (testcontainers, multi-layer)    │
│  ├── Service + Repo  — real DB, real service logic  │
│  └── NATS pub/sub    — real JetStream               │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## 3. Dependencies to Add

Add to each service's `go.mod` that needs repository testing:

```
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/testcontainers/testcontainers-go/modules/valkey
go get github.com/testcontainers/testcontainers-go/wait
```

For NATS integration tests:

```
go get github.com/nats-io/nats-server/v2/server
go get github.com/nats-io/nats-server/v2/test
```

NATS can be tested with an embedded in-process server (`server.New` +
`test.RunServer`) — no testcontainer needed since it's pure Go with no
external dependencies.

---

## 4. Shared Test Infrastructure

### 4.1 Postgres Test Helper

Create `services/shared/pkg/testutil/postgres.go`:

```go
package testutil

// PostgresContainer manages a testcontainers Postgres instance.
// Usage:
//   func TestMain(m *testing.M) {
//       pg := testutil.NewPostgresContainer(ctx, migrationsDir)
//       defer pg.Terminate(ctx)
//       code := m.Run()
//       os.Exit(code)
//   }
//
//   func TestFoo(t *testing.T) {
//       pool := pg.Pool(t)  // returns a pgxpool.Pool connected to a fresh schema
//   }

type PostgresContainer struct { ... }

func NewPostgresContainer(ctx context.Context, migrationsDir string) *PostgresContainer
func (c *PostgresContainer) Pool(t *testing.T) *pgxpool.Pool
func (c *PostgresContainer) Terminate(ctx context.Context) error
```

Key design decisions:

- **One container per `TestMain`** — containers are expensive to start (~2s).
  Reuse across tests in the same package.
- **Per-test schema isolation** — each `Pool(t)` call creates a new Postgres
  schema with a random name, runs migrations in it, and sets `search_path`.
  Tests run in parallel without interference.
- **Migrations via `golang-migrate`** — apply the service's migration files
  from `migrations/` directory automatically.
- **Cleanup via `t.Cleanup`** — schema dropped when test finishes.

```go
func (c *PostgresContainer) Pool(t *testing.T) *pgxpool.Pool {
    t.Helper()
    schema := "test_" + strings.ReplaceAll(uuid.New().String()[:8], "-", "")
    // CREATE SCHEMA, SET search_path, run migrations
    t.Cleanup(func() { /* DROP SCHEMA CASCADE */ })
    return pool
}
```

### 4.2 Valkey Test Helper

Create `services/shared/pkg/testutil/valkey.go`:

```go
type ValkeyContainer struct { ... }

func NewValkeyContainer(ctx context.Context) *ValkeyContainer
func (c *ValkeyContainer) Client(t *testing.T) vk.Client
func (c *ValkeyContainer) Terminate(ctx context.Context) error
```

- **Per-test key prefix** — each `Client(t)` wraps the client to prefix all
  keys with a test-unique namespace, or uses a separate DB number via `SELECT`.
- **`t.Cleanup` flushes keys** — FLUSHDB on the selected database.

### 4.3 NATS Test Helper

Create `services/shared/pkg/testutil/nats.go`:

```go
type NATSServer struct { ... }

func NewNATSServer(t *testing.T) *NATSServer
func (s *NATSServer) JetStream(t *testing.T) jetstream.JetStream
func (s *NATSServer) URL() string
```

- **Embedded NATS server** — uses `nats-server/v2/server` for an in-process
  server. No Docker needed, starts in <100ms.
- **Per-test cleanup** — streams and consumers purged between tests.
- **JetStream enabled** — configured with in-memory storage for speed.

### 4.4 Test Fixtures

Create `services/shared/pkg/testutil/fixtures.go`:

```go
package testutil

func RandomUUID() uuid.UUID        { return uuid.New() }
func RandomEmail() string          { return fmt.Sprintf("user-%s@test.com", uuid.New().String()[:8]) }
func RandomUsername() string       { return "user-" + uuid.New().String()[:8] }
func RandomRoomName() string       { return "room-" + uuid.New().String()[:8] }
func HashedPassword(t *testing.T, password string) string {
    t.Helper()
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost) // MinCost for speed
    require.NoError(t, err)
    return string(hash)
}
```

---

## 5. Service-by-Service Test Plan

### 5.1 Auth Service

#### 5.1.1 Repository Tests (`auth/internal/repository/`)

**File**: `auth/internal/repository/auth_test.go`

Requires: Postgres testcontainer.

| Test                    | Cases                                         |
| ----------------------- | --------------------------------------------- |
| `TestCreateAccount`     | success; duplicate email returns error        |
| `TestGetAccountByEmail` | found; not found returns `ErrAccountNotFound` |
| `TestGetAccountByID`    | found; not found returns `ErrAccountNotFound` |

**File**: `auth/internal/repository/token_test.go`

Requires: Valkey testcontainer.

| Test                          | Cases                                                    |
| ----------------------------- | -------------------------------------------------------- |
| `TestSaveRefreshToken`        | success; overwrite existing                              |
| `TestGetUserIDByRefreshToken` | found; not found returns `ErrTokenNotFound`; expired key |
| `TestDeleteRefreshToken`      | success; already deleted (idempotent)                    |

#### 5.1.2 Service Tests — Gap Fill (`auth/internal/service/`)

**Existing**: Register, Login, Logout tested.
**Missing**: `RefreshToken`.

| Test               | Cases                                                                                                                                                                      |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TestRefreshToken` | success — returns new pair, deletes old; invalid token — `ErrInvalidToken`; repo error on get — propagated; repo error on delete — propagated; generate error — propagated |

#### 5.1.3 Rate Limiter Tests (`auth/internal/ratelimit/`)

**File**: `auth/internal/ratelimit/ratelimit_test.go`

Requires: Valkey testcontainer.

| Test        | Cases                                                                                                                                                                                                             |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TestAllow` | under limit — returns nil; at limit — returns nil; over limit — returns `ErrRateLimitExceeded`; window expiry — counter resets; different actions — independent counters; different emails — independent counters |

---

### 5.2 Gateway Service

#### 5.2.1 Handler Tests

All gateway handlers are thin HTTP-to-gRPC translators. Test with `httptest`
and mock gRPC clients.

**File**: `gateway/internal/handler/room/handler_test.go`

| Test               | Cases                                                                           |
| ------------------ | ------------------------------------------------------------------------------- |
| `TestCreateRoom`   | success 201; invalid body 400; gRPC NotFound; gRPC AlreadyExists; gRPC Internal |
| `TestGetRoom`      | success 200; not found 404; internal 500                                        |
| `TestUpdateRoom`   | success 200; not found 404; permission denied 403                               |
| `TestDeleteRoom`   | success 204; not found 404; permission denied 403                               |
| `TestAddMember`    | success 204; not found 404; permission denied 403                               |
| `TestRemoveMember` | success 204; not found 404; permission denied 403; cannot remove owner 403      |

**File**: `gateway/internal/handler/user/handler_test.go`

| Test                | Cases               |
| ------------------- | ------------------- |
| `TestGetUser`       | success; not found  |
| `TestGetUsersByIds` | success; empty list |
| `TestUpdateUser`    | success; not found  |

**File**: `gateway/internal/handler/message/handler_test.go`

| Test              | Cases                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------ |
| `TestSendMessage` | success 201; invalid body; permission denied; invalid reply target                                     |
| `TestGetMessages` | success with messages; success empty; with cursor; with limit; limit clamped to max; permission denied |

**File**: `gateway/internal/handler/wsauth/handler_test.go`

| Test                 | Cases                                                                                                   |
| -------------------- | ------------------------------------------------------------------------------------------------------- |
| `TestCreateTicket`   | success 201 (with valid JWT context); missing user ID 401; invalid user ID format 400; Valkey error 500 |
| `TestValidateTicket` | success 200; missing ticket header 401; invalid/expired ticket 401; Valkey error 500                    |

**Mock strategy**: Create mock gRPC clients that implement the generated
`XxxServiceClient` interfaces. Use testify mocks or simple struct-with-func
pattern.

```go
type mockRoomClient struct {
    roomv1.RoomServiceClient
    createRoomFunc func(ctx context.Context, req *roomv1.CreateRoomRequest, ...) (*roomv1.CreateRoomResponse, error)
}
```

For wsauth tests, use a real Valkey testcontainer (since the handler
interacts directly with Valkey, not through a repository abstraction).

#### 5.2.2 Middleware Tests — Gap Fill

**File**: `gateway/internal/middleware/ratelimit_test.go`

Requires: Valkey testcontainer.

| Test                  | Cases                                                                                                              |
| --------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `TestUserRateLimiter` | under limit — passes through; over limit — 429; missing user ID — 401; different users — independent            |

---

### 5.3 Room Service

#### 5.3.1 Repository Tests (`room/internal/repository/`)

**File**: `room/internal/repository/room_test.go`

Requires: Postgres testcontainer.

| Test                       | Cases                                               |
| -------------------------- | --------------------------------------------------- |
| `TestCreateRoom`           | success group; success direct                       |
| `TestGetRoomByID`          | found; not found                                    |
| `TestUpdateRoom`           | success; not found                                  |
| `TestDeleteRoom`           | success; not found; cascades to members             |
| `TestAddMember`            | success; duplicate returns `ErrMemberAlreadyExists` |
| `TestRemoveMember`         | success; not found                                  |
| `TestGetRoomMembers`       | returns all; empty room                             |
| `TestIsMember`             | is member; is not member                            |
| `TestGetMemberRole`        | owner; member; not found                            |
| `TestGetDirectRoomByUsers` | found; not found; only matches direct type          |

#### 5.3.2 Service Tests — Gap Fill

**Existing tests cover most cases. Add:**

| Test                                       | Cases                                                                              |
| ------------------------------------------ | ---------------------------------------------------------------------------------- |
| `TestRemoveMember_SelfRemoval`             | non-owner self-removal succeeds; owner self-removal returns `ErrCannotRemoveOwner` |
| `TestRemoveMember_TargetIsOwner`           | attempting to remove another owner returns `ErrCannotRemoveOwner`                  |
| `TestUpdateRoom_NotMember`                 | non-member gets `ErrNotMember` (not `ErrNotOwner`)                                 |
| `TestDeleteRoom_NotMember`                 | non-member gets `ErrNotMember`                                                     |
| `TestCreateRoom_DirectRoom_DuplicateCheck` | second create returns `ErrDirectRoomExists`                                        |

#### 5.3.3 Handler Tests — Gap Fill

Add missing error mapping cases:

| Test                         | Cases                          |
| ---------------------------- | ------------------------------ |
| `TestUpdateRoom_NotMember`   | returns `CodePermissionDenied` |
| `TestDeleteRoom_NotMember`   | returns `CodePermissionDenied` |
| `TestRemoveMember_NotMember` | returns `CodePermissionDenied` |

---

### 5.4 Message Service

#### 5.4.1 Repository Tests — Extend

**Existing**: Content marshal/unmarshal tests.
**Add**: Postgres integration tests.

**File**: `message/internal/repository/postgres_test.go`

Requires: Postgres testcontainer.

| Test                                 | Cases                                                                                             |
| ------------------------------------ | ------------------------------------------------------------------------------------------------- |
| `TestCreateMessage`                  | text message; media message; with reply_to_id                                                     |
| `TestGetMessageByID`                 | found; not found; soft-deleted still returned                                                     |
| `TestGetMessagesByRoomID`            | returns ordered; pagination with cursor; respects limit; empty room                               |
| `TestSoftDeleteMessage`              | sets deleted_at; content preserved; not found                                                     |
| `TestGetMessagesByRoomID_Pagination` | keyset pagination correctness — next cursor points to correct boundary; final page has nil cursor |

#### 5.4.2 Service Tests — Gap Fill

| Test                             | Cases                                                                                     |
| -------------------------------- | ----------------------------------------------------------------------------------------- |
| `TestSendMessage_MediaType`      | IMAGE type stored correctly (verifies msgType shadowing fix); VIDEO type stored correctly |
| `TestDeleteMessage_AfterLeaving` | sender can delete after leaving room (documents behavior)                                 |
| `TestGetMessages_Pagination`     | cursor and limit forwarded correctly                                                      |

#### 5.4.3 Handler Tests — Gap Fill

| Test                                  | Cases                                                                                                               |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `TestSendMessage_AllContentTypes`     | text; image; video; file; voice; video_note; sticker; audio — verify correct `msgType` and content mapping for each |
| `TestSendMessage_TypeContentMismatch` | TEXT type with media content; IMAGE type with text content — 400                                                    |
| `TestGetMessages_InvalidCursor`       | malformed cursor UUID — 400                                                                                         |
| `TestDeleteMessage_NotSender`         | returns `CodePermissionDenied`                                                                                      |

#### 5.4.4 Client Tests (`message/internal/client/`)

**File**: `message/internal/client/room_test.go`

| Test                 | Cases                                                                                                                                                                         |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TestWithRetry`      | succeeds on first try; succeeds on second try after Unavailable; fails after all retries exhausted; non-retryable error returns immediately; context cancelled stops retrying |
| `TestIsMember`       | success; not member; gRPC error                                                                                                                                               |
| `TestGetRoomMembers` | success; invalid UUID in response returns error                                                                                                                               |

Use a mock gRPC server (via `google.golang.org/grpc/test/bufconn`) for
the client tests.

---

### 5.5 User Service

#### 5.5.1 Repository Tests (`user/internal/repository/`)

**File**: `user/internal/repository/user_test.go`

Requires: Postgres testcontainer.

| Test                | Cases                                                           |
| ------------------- | --------------------------------------------------------------- |
| `TestCreateUser`    | success; duplicate returns `ErrUserAlreadyExists` (ON CONFLICT) |
| `TestGetUserByID`   | found; not found                                                |
| `TestGetUsersByIDs` | all found; partial match; none found returns empty              |
| `TestUpdateUser`    | update display_name only; update all fields; not found          |

#### 5.5.2 Subscriber Tests (`user/internal/nats/`)

**File**: `user/internal/nats/subscriber_test.go`

Requires: Embedded NATS server.

| Test                       | Cases                                                                                       |
| -------------------------- | ------------------------------------------------------------------------------------------- |
| `TestHandleUserRegistered` | valid event — calls `CreateUser`; invalid JSON — naks message; service error — naks message |

Use the shared NATS test helper with embedded server. Publish a message to
the `auth.user.registered` subject and verify the mock `UserService` receives
the correct `CreateUser` call.

---

### 5.6 WebSocket Service

#### 5.6.1 Handler Tests (`websocket/internal/handler/`)

**File**: `websocket/internal/handler/ws_test.go`

WebSocket handler tests are more complex because they involve a real HTTP
server and WebSocket upgrade. Use `httptest.NewServer` with the `nhooyr.io/
websocket` client.

| Test                                | Cases                                     |
| ----------------------------------- | ----------------------------------------- |
| `TestHandleWS_MissingTicket`        | returns 401                               |
| `TestHandleWS_TicketTooLong`        | returns 400                               |
| `TestHandleWS_InvalidTicket`        | returns 401                               |
| `TestHandleWS_InvalidRoomID`        | returns 400 (before upgrade)              |
| `TestHandleWS_ValidConnection`      | upgrades successfully; registered in hub  |
| `TestHandleWS_WithInitialRoom`      | registered in the specified room          |
| `TestHandleMessage_Subscribe`       | sends subscribe, connection added to room |
| `TestHandleMessage_Unsubscribe`     | sends unsubscribe, connection removed     |
| `TestHandleMessage_Ping`            | sends ping, receives pong                 |
| `TestHandleMessage_UnknownType`     | receives error envelope                   |
| `TestHandleMessage_InvalidJSON`     | receives error envelope                   |
| `TestHandleMessage_MessageTooLarge` | receives error envelope (readLimit)       |

Mock strategy:

- Use a real hub instance (it's lightweight).
- Use a real Valkey testcontainer for ticket validation.
- Create test tickets directly in Valkey before connecting.

```go
func setupWSTest(t *testing.T) (*httptest.Server, vk.Client) {
    t.Helper()
    h := hub.NewHub(256)
    ctx, cancel := context.WithCancel(context.Background())
    t.Cleanup(cancel)
    go h.Run(ctx)

    valkeyClient := valkeyContainer.Client(t)
    cfg := &config.Config{...}
    wsHandler := handler.NewWebSocketHandler(h, "", valkeyClient, cfg)

    mux := http.NewServeMux()
    mux.HandleFunc("/ws", wsHandler.HandleWS)
    server := httptest.NewServer(mux)
    t.Cleanup(server.Close)

    return server, valkeyClient
}

func createTicket(t *testing.T, client vk.Client, userID string) string {
    t.Helper()
    ticket := uuid.New().String()
    key := valkey.GatewayWSTicketKey(ticket)
    err := client.Do(context.Background(),
        client.B().Set().Key(key).Value(userID).Ex(30*time.Second).Build(),
    ).Error()
    require.NoError(t, err)
    return ticket
}
```

#### 5.6.2 Subscriber Tests (`websocket/internal/subscriber/`)

**File**: `websocket/internal/subscriber/events_test.go`

Requires: Embedded NATS server.

| Test                              | Cases                                                 |
| --------------------------------- | ----------------------------------------------------- |
| `TestHandleMessageCreated`        | valid event — broadcasts to room; invalid JSON — naks |
| `TestHandleRoomMembershipChanged` | valid event — broadcasts; invalid JSON — naks         |
| `TestHandleRoomUpdated`           | valid event — broadcasts; invalid JSON — naks         |

Strategy: Create a real hub, register a mock connection in a room, publish
NATS events, and verify the mock connection receives the correct envelope
via `WriteJSON`.

```go
type spyConn struct {
    hub.Conn
    mu      sync.Mutex
    writes  []hub.Envelope
    writeCh chan struct{}
}

func (c *spyConn) WriteJSON(v any) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    if env, ok := v.(hub.Envelope); ok {
        c.writes = append(c.writes, env)
    }
    select {
    case c.writeCh <- struct{}{}:
    default:
    }
    return nil
}

func (c *spyConn) waitForWrite(t *testing.T, timeout time.Duration) {
    t.Helper()
    select {
    case <-c.writeCh:
    case <-time.After(timeout):
        t.Fatal("timed out waiting for write")
    }
}
```

#### 5.6.3 Hub Tests — Extend

**Existing**: 10 tests covering register, unregister, broadcast, room count.

**Add**:

| Test                                   | Cases                                                                      |
| -------------------------------------- | -------------------------------------------------------------------------- |
| `TestBroadcast_SlowConnection`         | write timeout — connection closed; other connections still receive message |
| `TestDisconnect_MultipleRooms`         | connection in 3 rooms — all cleaned up                                     |
| `TestRegister_SameUserMultipleConns`   | both connections receive broadcasts                                        |
| `TestGetRoomUserCount_AfterDisconnect` | count decrements correctly                                                 |

---

### 5.7 Shared Packages

#### 5.7.1 Mapper Tests

**File**: `shared/pkg/mapper/mapper_test.go`

| Test             | Cases                                                 |
| ---------------- | ----------------------------------------------------- |
| `TestSlice`      | maps correctly; empty slice; nil slice                |
| `TestEnum`       | known value; unknown value panics                     |
| `TestEnumWithOk` | known value returns true; unknown value returns false |

#### 5.7.2 NATS Package Tests

**File**: `shared/pkg/nats/nats_test.go`

Requires: Embedded NATS server.

| Test                             | Cases                                                         |
| -------------------------------- | ------------------------------------------------------------- |
| `TestConnect`                    | success; invalid URL; auth failure                            |
| `TestProvisionStreams`           | creates new stream; updates existing; multiple streams        |
| `TestProvisionConsumer`          | success; stream not found                                     |
| `TestProvisionConsumerWithRetry` | stream appears after retry; context cancelled — returns error |

#### 5.7.3 Valkey Package Tests

**File**: `shared/pkg/valkey/keys_test.go`

| Test               | Cases                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------- |
| `TestKeyFunctions` | each key function produces expected format; no collisions between different key types |

**File**: `shared/pkg/valkey/scripts_test.go`

Requires: Valkey testcontainer.

| Test                       | Cases                                                                                |
| -------------------------- | ------------------------------------------------------------------------------------ |
| `TestIncrWithExpiryScript` | first call sets TTL and returns 1; subsequent calls increment; key expires after TTL |

#### 5.7.4 Middleware Tests — Extend

**Existing**: RequestID, Logging, Recovery, UserID interceptors tested.

**Add**:

| Test                                      | Cases                              |
| ----------------------------------------- | ---------------------------------- |
| `TestRecoveryInterceptor_PanicWithString` | recovers string panic              |
| `TestRecoveryInterceptor_PanicWithError`  | recovers error panic               |
| `TestLoggingInterceptor_StatusError`      | logs code and unwrapped cause      |
| `TestLoggingInterceptor_Success`          | logs at INFO level, no error field |
| `TestRequestIDInterceptor_Propagates`     | existing request ID preserved      |
| `TestRequestIDInterceptor_Generates`      | missing request ID generated       |

#### 5.7.5 Healthcheck Tests — Extend

**Add**:

| Test             | Cases                                                            |
| ---------------- | ---------------------------------------------------------------- |
| `TestAwaitReady` | check succeeds — sets ready; context cancelled — stays not ready |
| `TestMiddleware` | /health and /ready intercepted; other paths pass through         |

---

## 6. Integration Tests

Integration tests combine multiple layers and use real infrastructure.
Place in `<service>/internal/integration/` directories with build tag
`//go:build integration`.

### 6.1 Auth Integration

**File**: `auth/internal/integration/auth_test.go`

Requires: Postgres + Valkey testcontainers, embedded NATS.

Tests the full flow: `handler → service → repository → DB`.

| Test                         | Flow                                                            |
| ---------------------------- | --------------------------------------------------------------- |
| `TestRegisterLoginFlow`      | Register → Login with same credentials → verify tokens work     |
| `TestRegisterDuplicate`      | Register → Register same email → AlreadyExists                  |
| `TestLoginInvalidPassword`   | Register → Login wrong password → InvalidCredentials            |
| `TestRefreshTokenFlow`       | Register → use refresh token → get new pair → old token invalid |
| `TestLogoutInvalidatesToken` | Register → Logout → refresh with old token fails                |
| `TestRateLimiting`           | Register N+1 times rapidly → last one rate limited              |

Setup:

```go
func setupAuthIntegration(t *testing.T) authv1.AuthServiceServer {
    t.Helper()
    pool := pgContainer.Pool(t)
    valkeyClient := valkeyContainer.Client(t)
    js := natsServer.JetStream(t)

    accountRepo := repository.NewAccountRepository(pool)
    tokenRepo := repository.NewTokenRepository(valkeyClient)
    publisher := nats.NewPublisher(js)
    limiter := ratelimit.NewValkeyLimiter(valkeyClient, 5, time.Minute)

    svc := service.NewAuthService(accountRepo, tokenRepo, publisher, limiter,
        "test-secret", 15*time.Minute, 24*time.Hour)

    handler, err := handler.NewAuthHandler(svc)
    require.NoError(t, err)
    return handler
}
```

### 6.2 Room Integration

**File**: `room/internal/integration/room_test.go`

Requires: Postgres testcontainer, embedded NATS.

| Test                          | Flow                                                           |
| ----------------------------- | -------------------------------------------------------------- |
| `TestCreateAndGetRoom`        | Create → Get → verify all fields                               |
| `TestRoomMembershipLifecycle` | Create → AddMember → GetMembers → RemoveMember → verify        |
| `TestOwnerPermissions`        | Create as A → Update as B (non-member) → PermissionDenied      |
| `TestDirectRoomDedup`         | Create direct A↔B → Create direct A↔B again → AlreadyExists    |
| `TestDeleteRoomCascade`       | Create → AddMembers → Delete → GetRoom returns NotFound        |
| `TestOwnerCannotSelfRemove`   | Create → RemoveMember(self) → CannotRemoveOwner                |
| `TestNonOwnerCanSelfRemove`   | Create → AddMember(B) → RemoveMember(B, requester=B) → success |

### 6.3 Message Integration

**File**: `message/internal/integration/message_test.go`

Requires: Postgres testcontainer, mock room client, embedded NATS.

| Test                     | Flow                                                                      |
| ------------------------ | ------------------------------------------------------------------------- |
| `TestSendAndGetMessages` | Send 3 messages → GetMessages → verify order and content                  |
| `TestPagination`         | Send 10 messages → GetMessages(limit=3) → verify cursor → next page       |
| `TestReplyChain`         | Send A → Reply B to A → Get B → verify reply_to preview                   |
| `TestReplyToDeleted`     | Send A → Delete A → Reply to A → InvalidReplyTarget                       |
| `TestSoftDelete`         | Send → Delete → Get → verify is_deleted=true, content preserved           |
| `TestCrossRoomReply`     | Send in room1 → Reply in room2 pointing to room1 msg → InvalidReplyTarget |

### 6.4 User Integration

**File**: `user/internal/integration/user_test.go`

Requires: Postgres testcontainer.

| Test                     | Flow                                                  |
| ------------------------ | ----------------------------------------------------- |
| `TestCreateAndGetUser`   | Create → Get → verify username derived from email     |
| `TestIdempotentCreate`   | Create → Create same ID → returns existing (no error) |
| `TestUpdateUser`         | Create → Update display_name → Get → verify updated   |
| `TestGetUsersByIDs`      | Create 3 → GetByIDs([1,3]) → returns 2                |
| `TestEmptyEmailFallback` | Create with empty email → username is `user-XXXXXXXX` |

### 6.5 Event Flow Integration

**File**: `services/shared/pkg/testutil/integration/events_test.go`

Requires: Embedded NATS.

Tests the full NATS event pipeline between services.

| Test                         | Flow                                                            |
| ---------------------------- | --------------------------------------------------------------- |
| `TestUserRegisteredEvent`    | Auth publishes → User subscriber receives and creates user      |
| `TestMessageCreatedEvent`    | Message publishes → WS subscriber receives and formats envelope |
| `TestMembershipChangedEvent` | Room publishes → WS subscriber broadcasts to room               |

These tests wire up real publishers and subscribers against an embedded NATS
server and verify the full serialization/deserialization pipeline.

---

## 7. CI Pipeline Updates

### 7.1 Test Execution

Update `.github/scripts/test-go-modules.sh`:

```bash
# Unit tests (always run, no external deps)
go test -coverprofile="$tmp" -covermode=atomic \
    -tags='!integration' \
    -timeout=5m ./...

# Integration tests (only when Docker available)
if command -v docker &>/dev/null; then
    go test -coverprofile="$tmp_int" -covermode=atomic \
        -tags=integration \
        -timeout=10m ./...
fi
```

### 7.2 Coverage Merging

Merge unit + integration coverage into a single profile for accurate
reporting. Update `check-coverage.sh` to accept both profiles.

### 7.3 GitHub Actions

Update `.github/workflows/_go.yml`:

```yaml
services:
  # No services needed — testcontainers manages its own containers
steps:
  - name: Unit Tests
    run: task go:test
  - name: Integration Tests
    run: task go:test-integration
    env:
      TESTCONTAINERS_RYUK_DISABLED: "false"
```

### 7.4 Coverage Thresholds

Update `.github/scripts/check-coverage.sh`:

```
Service packages:   70% → 80%  (after repo tests added)
Handler packages:   60% → 75%  (after gateway handler tests added)
Repository packages: 0% → 80%  (new target)
Shared packages:    60% → 80%  (after mapper/NATS tests added)
```

---

## 8. Coverage Targets

### Per-Service Targets

| Service       | Layer           | Current | Target | Notes                                    |
| ------------- | --------------- | :-----: | :----: | ---------------------------------------- |
| **Auth**      | Handler         |  ~85%   |  90%   | Add RefreshToken edge cases              |
|               | Service         |  ~60%   |  85%   | Add RefreshToken, rate limit error paths |
|               | Repository      |   0%    |  80%   | New — testcontainers Postgres + Valkey   |
|               | Rate Limiter    |   0%    |  90%   | New — testcontainers Valkey              |
| **Gateway**   | Auth Handler    |  ~80%   |  90%   | Existing tests solid                     |
|               | Room Handler    |   0%    |  85%   | New — mock gRPC client                   |
|               | User Handler    |   0%    |  85%   | New — mock gRPC client                   |
|               | Message Handler |   0%    |  85%   | New — mock gRPC client                   |
|               | WSAuth Handler  |   0%    |  90%   | New — testcontainers Valkey              |
|               | JWT Middleware  |  ~90%   |  90%   | Existing tests solid                     |
|               | Rate Limit MW   |   0%    |  85%   | New — testcontainers Valkey              |
| **Room**      | Handler         |  ~70%   |  85%   | Add ErrNotMember cases                   |
|               | Service         |  ~75%   |  90%   | Add self-removal, ownership edge cases   |
|               | Repository      |   0%    |  85%   | New — testcontainers Postgres            |
| **Message**   | Handler         |  ~65%   |  85%   | Add all content types, type mismatch     |
|               | Service         |  ~70%   |  85%   | Add media type, pagination               |
|               | Repository      |  ~40%   |  85%   | Extend with Postgres integration         |
|               | Client          |   0%    |  85%   | New — bufconn mock gRPC                  |
| **User**      | Handler         |  ~80%   |  90%   | Existing tests solid                     |
|               | Service         |  ~75%   |  85%   | Add empty email fallback                 |
|               | Repository      |   0%    |  80%   | New — testcontainers Postgres            |
|               | Subscriber      |   0%    |  80%   | New — embedded NATS                      |
| **WebSocket** | Handler         |   0%    |  75%   | New — httptest + WS client               |
|               | Hub             |  ~70%   |  85%   | Add slow conn, multi-room                |
|               | Subscriber      |   0%    |  80%   | New — embedded NATS + spy conn           |
| **Shared**    | Errors          |  ~90%   |  95%   | Existing tests solid                     |
|               | Middleware      |  ~70%   |  85%   | Add StatusError logging cases            |
|               | Healthcheck     |  ~80%   |  90%   | Add AwaitReady, Middleware               |
|               | Logger          |  ~85%   |  90%   | Existing tests solid                     |
|               | Mapper          |   0%    |  95%   | New — pure logic, easy                   |
|               | NATS            |   0%    |  80%   | New — embedded NATS                      |
|               | Valkey          |   0%    |  85%   | New — testcontainers Valkey              |

### Aggregate Targets

| Metric                   | Current | Target |
| ------------------------ | :-----: | :----: |
| Overall line coverage    |  ~55%   |  80%+  |
| Services with repo tests |   1/5   |  5/5   |
| Integration test suites  |    0    |   5    |
| Test files               |   22    |  ~45   |

---

## Implementation Order

Recommended order for implementing the test plan:

```
1. Shared test infrastructure (testutil package)     — foundation
2. Shared package tests (mapper, NATS, valkey)        — quick wins
3. Repository tests (all services)                    — biggest gap
4. Service tests gap fill                             — extend existing
5. Gateway handler tests                              — second biggest gap
6. WebSocket handler + subscriber tests               — complex but critical
7. Message client tests (bufconn)                     — isolated
8. Integration tests                                  — capstone
9. CI pipeline updates                                — finalize
```

Each step is independently shippable. Repository tests (step 3) provide
the most coverage improvement per effort.
