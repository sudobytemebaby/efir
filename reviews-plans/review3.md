  ---
  Efir Codebase Review — Third Pass

  ▎ Generated: 2026-03-28 | Branch: main | ~7.2k LOC source + ~5.1k LOC tests across 8 services + shared

  ▎ Follow-up to the second review (2026-03-25). Significant improvements since last review.
  ▎ All 46 items from the first review are resolved. Most items from the second review are
  ▎ addressed. This document covers the current state, remaining issues, and new findings.

  ---
  Previous Review: Resolution Status

  The second review contained 13 items (2 bugs, 4 should-fix, 4 nice-to-have, 3 informational).

  ┌─────┬──────────────────────────────────────────┬─────────────────────────────────────────────────────────────────────────────┐
  │  #  │                  Issue                   │                                   Status                                    │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 1   │ Hub sendToRoom closes wrong connection   │ Fixed — sendToRoom rewritten with clean goroutine-per-connection, no        │
  │     │ on timeout                               │ timeout branch uses outer variable                                          │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 2   │ HTTP error sent after WebSocket upgrade  │ Fixed — room_id validation now happens before websocket.Accept()            │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 3   │ Auth service interface leaks repository  │ Fixed — service.Account domain type defined (line 25-29)                    │
  │     │ types                                    │                                                                             │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 4   │ Message service returns unwrapped errors │ Fixed — all error returns now use fmt.Errorf wrapping                       │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 5   │ Gateway wsauth has no rate limiting      │ Fixed — wrapped in ipRateLimiter group                                      │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 6   │ ErrCannotRemoveSelf defined but never    │ Open — still defined at room/internal/service/room.go:20, never returned    │
  │     │ used                                     │                                                                             │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 7   │ ErrRoomAlreadyExists defined but never   │ Fixed — removed                                                             │
  │     │ used                                     │                                                                             │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 8   │ No request body size limit in gateway    │ Fixed — io.LimitReader with configurable maxBodySize                        │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 9   │ generateUsernameFromEmail can produce    │ Fixed — falls back to "user-" + uuid                                        │
  │     │ empty string                             │                                                                             │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 10  │ Health listen address not configurable   │ Fixed — HealthListenAddr in ServerConfig                                    │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 11  │ Wrap() destroys error chain              │ Accepted — now uses StatusError with Unwrap(), preserving the Go error      │
  │     │                                          │ chain                                                                       │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 12  │ context.Background() in WS error logs,   │ Open — still no user_id in WS error logs                                    │
  │     │ no user_id                               │                                                                             │
  ├─────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
  │ 13  │ Test coverage gaps                       │ Partially addressed — errors package now has tests (20 cases)               │
  └─────┴──────────────────────────────────────────┴─────────────────────────────────────────────────────────────────────────────┘

  ---
  New Findings

  1. Security — CreateTicket reads user ID from raw header, not JWT

  Severity: Critical security vulnerability
  File: services/gateway/internal/handler/wsauth/handler.go:40

  CreateTicket reads X-User-Id directly from the HTTP request header:

  userID := r.Header.Get("X-User-Id")

  This endpoint is behind ipRateLimiter but NOT behind jwtMiddleware. Any client can create a WebSocket ticket for any user by
  sending:
  POST /auth/ws-ticket
  X-User-Id: <victim-uuid>

  The ticket is stored in Valkey and later used to authenticate the WebSocket connection. This allows full impersonation.

  Task: Move CreateTicket behind the jwtMiddleware group, or add JWT validation within the handler. Use
  middleware.MustGetUserID(r.Context()) instead of reading the raw header.

  ---
  2. Bug — SendMessage handler shadows msgType for media messages

  Severity: Bug (media messages stored with wrong type)
  File: services/message/internal/handler/message.go:75

  In the Media case of the SendMessage handler:

  var msgType service.MessageType  // line 63: outer variable
  // ...
  case *messagev1.SendMessageRequest_Media:
      media := c.Media
      msgType, ok := protoToMessageType(req.Type)  // line 75: := creates NEW local variable

  The := on line 75 creates a new local msgType that shadows the outer one. The outer msgType (used at line 183 to build
  SendMessageInput) remains its zero value ("") for all media messages (IMAGE, VIDEO). The message will be stored in the database
  with an empty type string.

  Task: Change := to = on line 75 and declare ok separately, or restructure to use the same assignment pattern as other branches.

  ---
  3. Bug — RemoveMember handler missing ErrNotMember check

  Severity: Bug (wrong HTTP status code)
  File: services/room/internal/handler/room.go:213-228

  The service's RemoveMember returns ErrNotMember when the requester isn't a room member (line 233 of service). The handler checks
  for ErrRoomNotFound, ErrNotOwner, ErrMemberNotFound, and ErrCannotRemoveOwner — but not ErrNotMember. The error falls through to
  CodeInternal.Wrap(err), returning HTTP 500 instead of 403.

  Same issue exists in UpdateRoom handler (line 115-123) and DeleteRoom handler (line 146-153) — both services can return
  ErrNotMember but the handlers don't check for it.

  Task: Add ErrNotMember checks to RemoveMember, UpdateRoom, and DeleteRoom handlers, mapping to CodePermissionDenied.

  ---
  4. Design — mapper.Enum panics on unknown values

  Severity: Minor risk
  File: services/shared/pkg/mapper/mapper.go:13-18

  mapper.Enum calls panic() on unknown enum values. This is used in the message handler's messageToProto mapper. If the database
  contains an unexpected message type (e.g., after a migration that adds a new type before the code is updated), the entire service
  crashes.

  func Enum[From comparable, To any](m map[From]To, from From) To {
      to, ok := m[from]
      if !ok {
          panic(fmt.Sprintf("mapper: unknown enum value: %v", from))
      }
      return to
  }

  The RecoveryInterceptor will catch this panic and return codes.Internal, so it won't take down the process. But it's still an
  ungraceful failure mode.

  Task: Consider using EnumWithOk instead in response mappers, or return a default/unknown enum value rather than panicking.

  ---
  5. Smell — ErrCannotRemoveSelf still defined but never used

  Severity: Dead code
  File: services/room/internal/service/room.go:20

  Still defined, still unreferenced. The RemoveMember method handles self-removal correctly (owner can't self-remove, non-owner
  self-removal is allowed) but never returns this error.

  Task: Remove it, or add a comment explaining it's reserved for future self-leave restrictions.

  ---
  6. Smell — WebSocket logs lack user_id context

  Severity: Cosmetic / operational
  File: services/websocket/internal/handler/ws.go:91,104,106,132

  The readPump and pingPump functions log errors with context.Background() and no user_id field. When debugging connection issues in
  production, there's no way to correlate these errors to specific users.

  Task: Pass userID to readPump and pingPump, include as structured field in all log calls.

  ---
  7. Minor — DeleteMessage doesn't check membership

  Severity: Minor authorization gap
  File: services/message/internal/service/message.go:152-166

  DeleteMessage checks that the requester is the sender (msg.SenderID != requesterID) but doesn't verify the requester is still a
  room member. A user who was removed from a room can still delete their old messages if they know the message ID.

  This may be intentional (a user should be able to delete their own messages even after leaving), but it's worth documenting the
  decision.

  Task: Document the behavior, or add a membership check if deletion should be restricted to current members.

  ---
  8. Minor — Gateway getMessages doesn't validate limit range

  Severity: Minor
  File: services/gateway/internal/handler/message/handler.go:53-56

  The gateway parses limit from query params but doesn't validate it's within a reasonable range. A client can pass limit=999999 to
  attempt a large query. The protobuf validation might catch this, but the gateway should enforce it at the boundary.

  if parsed, err := strconv.ParseInt(l, 10, 32); err == nil {
      req.Limit = int32(parsed)  // no bounds check
  }

  Task: Clamp limit to a max value (e.g., 100) at the gateway level.

  ---
  9. Minor — Duplicate okHandler function

  Severity: Cosmetic
  Files: services/websocket/cmd/main.go:127-130, services/sidecar/cmd/main.go:56-59

  Both services define identical standalone okHandler functions for health endpoints. The websocket service doesn't use the shared
  healthcheck package like other services.

  Task: Consider using the shared healthcheck package in the websocket service for consistency.

  ---
  Test Coverage Assessment

  Testing has improved with the addition of errors package tests. Current state:

  ┌───────────┬──────────────┬──────────────┬─────────────────┬─────────────┬──────────────────────────────────────────────────┐
  │  Service  │   Handler    │   Service    │   Repository    │   Config    │                      Other                       │
  │           │    Tests     │    Tests     │      Tests      │    Tests    │                                                  │
  ├───────────┼──────────────┼──────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ Auth      │  20+ cases   │ 3+ functions │        —        │     Yes     │                        —                         │
  ├───────────┼──────────────┼──────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ Gateway   │   Auth: 8    │      —       │        —        │     Yes     │                JWT middleware: 11                │
  ├───────────┼──────────────┼──────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ Room      │   7+ cases   │  10+ cases   │        —        │     Yes     │                        —                         │
  ├───────────┼──────────────┼──────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ Message   │     Yes      │     Yes      │ Yes (postgres)  │     Yes     │                        —                         │
  ├───────────┼──────────────┼──────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ User      │   10 cases   │ 7 functions  │        —        │     Yes     │                        —                         │
  ├───────────┼──────────────┼──────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ WebSocket │      —       │      —       │        —        │     Yes     │                     Hub: 10                      │
  ├───────────┼──────────────┼──────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ Shared    │      —       │      —       │        —        │      —      │    Middleware: 6, Healthcheck: 7, Logger: 5,     │
  │           │              │              │                 │             │                    Errors: 20                    │
  └───────────┴──────────────┴──────────────┴─────────────────┴─────────────┴──────────────────────────────────────────────────┘

  Key gaps remaining:
  - No gateway handler tests for user, room, message, wsauth endpoints
  - No gateway ratelimit middleware tests
  - No repository tests for auth, room, or user services
  - No WebSocket handler or subscriber tests
  - No integration tests
  - Auth RefreshToken method has no unit test
  - Shared mapper and NATS packages have no tests

  ---
  Service Scores

  Auth Service

  ┌───────────────┬───────┬──────────────────────────────────────────────────────────────────────────────────────────────┐
  │   Dimension   │ Score │                                            Notes                                             │
  ├───────────────┼───────┼──────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Correctness   │   9   │ Token refresh ordering correct. Rate limiting atomic. Domain types properly isolated.        │
  ├───────────────┼───────┼──────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality  │  8.5  │ Clean 3-layer architecture. Consistent error wrapping. Service-level Account type.           │
  ├───────────────┼───────┼──────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go  │   9   │ run() pattern, proper DI, context propagation, time.Duration configs.                        │
  ├───────────────┼───────┼──────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Security      │   8   │ Bcrypt, atomic rate limiting, no PII in errors. JWT lacks iss/aud (acceptable for internal). │
  ├───────────────┼───────┼──────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Test Coverage │  6.5  │ Handler (20+), service (3+), config tests. RefreshToken untested. No repo tests.             │
  ├───────────────┼───────┼──────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Average       │  8.2  │                                                                                              │
  └───────────────┴───────┴──────────────────────────────────────────────────────────────────────────────────────────────┘

  Gateway Service

  ┌───────────────┬───────┬───────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │   Dimension   │ Score │                                                 Notes                                                 │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Correctness   │   7   │ Critical: CreateTicket reads user ID from raw header without JWT validation — impersonation possible. │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality  │  8.5  │ Centralized WriteError/WriteProto helpers. maxBodySize configurable. Request ID propagation.          │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go  │   9   │ chi router, clean middleware groups, timeout interceptor. protojson transcoding is clean.             │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Security      │  5.5  │ JWT middleware solid. CreateTicket allows user impersonation. No limit validation on getMessages.     │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Test Coverage │   6   │ Auth handler (8), config, JWT middleware (11). No tests for user/room/message/wsauth/ratelimit.       │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Average       │  7.2  │                                                                                                       │
  └───────────────┴───────┴───────────────────────────────────────────────────────────────────────────────────────────────────────┘

  Room Service

  ┌───────────────┬───────┬────────────────────────────────────────────────────────────────────────────────────────────────┐
  │   Dimension   │ Score │                                             Notes                                              │
  ├───────────────┼───────┼────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Correctness   │   8   │ Role-based checks correct. Owner removal prevented. Handler missing ErrNotMember in 3 methods. │
  ├───────────────┼───────┼────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality  │  8.5  │ Clean layering. Domain types. memberUserIDs helper. One dead error constant.                   │
  ├───────────────┼───────┼────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go  │   9   │ Consistent error wrapping, interface DI, proper context propagation.                           │
  ├───────────────┼───────┼────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Security      │  8.5  │ Permission model sound. Event publishing with event_lost logging.                              │
  ├───────────────┼───────┼────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Test Coverage │   7   │ Handler and service tests. No repository tests.                                                │
  ├───────────────┼───────┼────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Average       │  8.2  │                                                                                                │
  └───────────────┴───────┴────────────────────────────────────────────────────────────────────────────────────────────────┘

  Message Service

  ┌──────────────┬───────┬───────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │  Dimension   │ Score │                                                 Notes                                                 │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Correctness  │   7   │ Bug: msgType variable shadowing causes empty type for media messages. Reply validation thorough.      │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality │  8.5  │ gRPC client with typed retry. Keyset pagination. Clean type conversion. Consistent error wrapping     │
  │              │       │ (fixed).                                                                                              │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go │   8   │ Consistent patterns. Room client retry is clean generic implementation.                               │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Security     │   8   │ Membership checks on reads. Sender-only deletion. DeleteMessage doesn't recheck membership            │
  │              │       │ (debatable).                                                                                          │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Test         │  7.5  │ Handler, service, and repository tests all present. Best coverage of any service.                     │
  │ Coverage     │       │                                                                                                       │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Average      │  7.8  │                                                                                                       │
  └──────────────┴───────┴───────────────────────────────────────────────────────────────────────────────────────────────────────┘

  User Service

  ┌───────────────┬───────┬───────────────────────────────────────────────────────────────────────────────┐
  │   Dimension   │ Score │                                     Notes                                     │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────┤
  │ Correctness   │   9   │ Idempotent create. Empty email fallback handled. Event-driven creation clean. │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality  │  8.5  │ Simple, focused. Clean NATS subscriber pattern.                               │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go  │  8.5  │ Proper layering, context propagation, error mapping.                          │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────┤
  │ Security      │  8.5  │ No direct mutations from external input. Writes go through auth event.        │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────┤
  │ Test Coverage │  6.5  │ Handler (10), service (7), config. No subscriber or repo tests.               │
  ├───────────────┼───────┼───────────────────────────────────────────────────────────────────────────────┤
  │ Average       │  8.2  │                                                                               │
  └───────────────┴───────┴───────────────────────────────────────────────────────────────────────────────┘

  WebSocket Service

  ┌──────────────┬───────┬───────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │  Dimension   │ Score │                                                 Notes                                                 │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Correctness  │  8.5  │ Previous bugs fixed (variable capture, HTTP error after upgrade). Room ID validated before accept.    │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality │  7.5  │ Clean hub architecture. Read/write pumps with timeouts. Ping/pong implemented. Doesn't use shared     │
  │              │       │ healthcheck.                                                                                          │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go │  7.5  │ Good channel-based dispatch. wsConnWrapper with mutex correct. No user_id in logs.                    │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Security     │  7.5  │ Ticket-based auth with GETDEL. Ticket length validation. IP rate limiting via gateway.                │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Test         │  5.5  │ Only hub tests (10) and config. No handler, subscriber, or integration tests.                         │
  │ Coverage     │       │                                                                                                       │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Average      │  7.3  │                                                                                                       │
  └──────────────┴───────┴───────────────────────────────────────────────────────────────────────────────────────────────────────┘

  Shared Packages

  ┌──────────────┬───────┬───────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │  Dimension   │ Score │                                                 Notes                                                 │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Correctness  │  8.5  │ Error mapping works correctly. StatusError preserves error chain via Unwrap(). Fallback to            │
  │              │       │ 500/codes.Internal for unknown codes.                                                                 │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality │  8.5  │ Clean abstractions. mapper.Enum panics — caught by recovery interceptor but ungraceful.               │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go │   9   │ Empty-struct context keys, atomic.Bool for health, slog integration, generics in mapper.              │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Security     │  8.5  │ Recovery interceptor catches panics. No sensitive data logged.                                        │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Test         │   7   │ Errors (20), logger (5), middleware (6), healthcheck (7). Mapper, NATS, valkey untested.              │
  │ Coverage     │       │                                                                                                       │
  ├──────────────┼───────┼───────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Average      │  8.3  │                                                                                                       │
  └──────────────┴───────┴───────────────────────────────────────────────────────────────────────────────────────────────────────┘

  ---
  Overall Score

  ┌───────────────────┬───────┬──────────┬───────────────────────────────────────────────────────────────────────────────────────┐
  │     Dimension     │ Score │  Delta   │                                       Rationale                                       │
  │                   │       │ from R2  │                                                                                       │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Architecture      │  8.5  │    —     │ Clean microservice boundaries. Event-driven via NATS JetStream. Gateway pattern with  │
  │                   │       │          │ protojson. Proper layer separation with domain types in all services.                 │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Correctness       │  7.5  │    —     │ Two bugs found: msgType shadowing in media messages, missing ErrNotMember handler     │
  │                   │       │          │ checks. Critical security issue with CreateTicket. Previous bugs all fixed.           │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Code Quality      │  8.5  │   +0.5   │ More consistent error wrapping. Domain types properly separated. Shared helpers       │
  │                   │       │          │ well-designed. Minor dead code (ErrCannotRemoveSelf).                                 │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Idiomatic Go      │  8.5  │   +0.5   │ run() pattern, signal.NotifyContext, channel-based hub, clean config, proper DI.      │
  │                   │       │          │ StatusError with Unwrap() preserves error chains.                                     │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Security          │  6.5  │   -1.0   │ Critical: CreateTicket allows user impersonation. Everything else is solid: rate      │
  │                   │       │          │ limiting, bcrypt, safe error responses, body size limits.                             │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Test Coverage     │  6.5  │   +0.5   │ Errors package now tested (20 cases). Config tests across all services. Still         │
  │                   │       │          │ missing: repo tests, gateway handler tests, integration tests, WS handler tests.      │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Operational       │  8.5  │   +0.5   │ Graceful shutdown everywhere. Configurable health listen addresses. Request ID        │
  │ Readiness         │       │          │ tracing. event_lost logging. CGO_ENABLED=0. HEALTHCHECK. Version info.                │
  ├───────────────────┼───────┼──────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Infrastructure    │  8.5  │    —     │ Excellent Docker Compose setup. Traefik. Full observability (OTEL, Prometheus, Loki,  │
  │                   │       │          │ Tempo, Grafana). Task runner. CI pipeline. 14 ADRs.                                   │
  └───────────────────┴───────┴──────────┴───────────────────────────────────────────────────────────────────────────────────────┘

  Overall: 7.9 / 10 (up from 7.8)

  The codebase has improved consistently. The architecture is sound, the error model is now proper (preserving Go error chains),
  domain types are correctly separated, and most previous issues are resolved. The main concern is the CreateTicket security
  vulnerability — fixing that alone would push the security score to ~8.0 and overall to ~8.1.

  ---
  Prioritized Action Items

  Must Fix (security + bugs)

  ┌─────┬──────────────────────────────────────────────┬────────────────────────────────────────────────────────────┬────────────┐
  │  #  │                    Issue                     │                            File                            │   Effort   │
  ├─────┼──────────────────────────────────────────────┼────────────────────────────────────────────────────────────┼────────────┤
  │ 1   │ CreateTicket allows user impersonation —     │ gateway/internal/handler/wsauth/handler.go:40 +            │ ~5 lines   │
  │     │ reads user ID from raw header without JWT    │ gateway/cmd/main.go:133-136                                │            │
  ├─────┼──────────────────────────────────────────────┼────────────────────────────────────────────────────────────┼────────────┤
  │ 2   │ msgType shadowing — media messages stored    │ message/internal/handler/message.go:75                     │ 1 line (:= │
  │     │ with empty type                              │                                                            │  → =)      │
  ├─────┼──────────────────────────────────────────────┼────────────────────────────────────────────────────────────┼────────────┤
  │ 3   │ Missing ErrNotMember checks in 3 room        │ room/internal/handler/room.go:115,146,213                  │ ~9 lines   │
  │     │ handler methods                              │                                                            │            │
  └─────┴──────────────────────────────────────────────┴────────────────────────────────────────────────────────────┴────────────┘

  Should Fix

  ┌─────┬───────────────────────────────────────────┬───────────────────────────────────────────────────┬───────────┐
  │  #  │                   Issue                   │                       File                        │  Effort   │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────┤
  │ 4   │ mapper.Enum panics on unknown values      │ shared/pkg/mapper/mapper.go:15                    │ ~3 lines  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────┤
  │ 5   │ Gateway getMessages no limit bounds check │ gateway/internal/handler/message/handler.go:53-56 │ ~3 lines  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────┤
  │ 6   │ Add user_id to WebSocket error logs       │ websocket/internal/handler/ws.go                  │ ~10 lines │
  └─────┴───────────────────────────────────────────┴───────────────────────────────────────────────────┴───────────┘

  Nice to Have

  ┌─────┬─────────────────────────────────────────────┬─────────────────────────────────────────┬───────────┐
  │  #  │                    Issue                    │                  File                   │  Effort   │
  ├─────┼─────────────────────────────────────────────┼─────────────────────────────────────────┼───────────┤
  │ 7   │ Remove unused ErrCannotRemoveSelf           │ room/internal/service/room.go:20        │ 1 line    │
  ├─────┼─────────────────────────────────────────────┼─────────────────────────────────────────┼───────────┤
  │ 8   │ Use shared healthcheck in websocket service │ websocket/cmd/main.go                   │ ~10 lines │
  ├─────┼─────────────────────────────────────────────┼─────────────────────────────────────────┼───────────┤
  │ 9   │ Document DeleteMessage membership behavior  │ message/internal/service/message.go:152 │ Comment   │
  └─────┴─────────────────────────────────────────────┴─────────────────────────────────────────┴───────────┘

  Test Debt

  - Auth: RefreshToken unit test
  - Gateway: handler tests for user, room, message, wsauth; ratelimit middleware
  - All services: repository-level tests
  - WebSocket: handler and subscriber tests
  - Shared: mapper and NATS package tests
  - Integration tests
