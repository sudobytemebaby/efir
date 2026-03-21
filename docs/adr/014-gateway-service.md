# ADR-014: Gateway Service Implementation

## Status

Accepted

## Context

The Gateway Service is the single HTTP entry point for all client requests. It
must validate JWT tokens, enforce rate limiting, proxy requests to downstream
gRPC services, and manage WebSocket ticket issuance. Several architectural
decisions were made during the design phase that deviate from or extend the
original epic plan.

## Decisions

### 1. JWT Validation — Local Only

The Gateway validates JWT access tokens locally by verifying the signature
against `JWT_SECRET`. No gRPC call to the Auth Service (`ValidateToken`) is
made per request.

**Rationale:** There is no token blacklist in the current implementation.
Refresh tokens are invalidated in Valkey on logout, but access tokens are
short-lived (15 minutes) and cannot be revoked. A gRPC round trip to Auth on
every request would add latency with no security benefit given the absence of a
blacklist.

**Known limitation:** A logged-out user can continue using a valid access token
until it expires naturally (up to 15 minutes). This is an accepted trade-off
for MVP. A blacklist or token introspection endpoint can be introduced in
Module 2.

---

### 2. Rate Limiting — Gateway Only

Rate limiting is enforced exclusively at the Gateway layer. Downstream services
(Auth, User, Room, Message, WebSocket) do not implement their own rate
limiters.

**Rationale:** Downstream services are only reachable from within the Docker
network and accept connections exclusively from the Gateway. Rate limiting
inside downstream services would protect against a compromised or buggy Gateway
— a scenario considered out of scope for MVP. Traefik provides an additional
global rate limiting layer at the network edge.

The Auth Service already has a rate limiter implemented (per ADR-008). It is
left in place and will not be removed, but no new downstream services will
implement rate limiting.

---

### 3. Rate Limiting — Fixed Window with Lua Script

The Gateway rate limiter uses a fixed window counter stored in Valkey,
implemented via a Lua script executed with `EVAL`:

```lua
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
```

**Rationale:** The Lua script executes atomically on the Valkey server,
eliminating the race condition between `INCR` and `EXPIRE` documented in
ADR-008. A sliding window was considered but rejected as over-engineering given
that Traefik already provides a rate limiting layer. The `SET NX` + `INCR`
pattern was also considered — it is more readable but requires two round trips
and has an edge case where a crash between the two commands leaves the counter
stuck at zero until TTL expiry.

Two rate limiters are applied:

- **By IP** — key pattern `gateway:ratelimit:ip:{ip}:{window}`, applied to
  public auth endpoints (`/auth/*`) to protect against brute force
- **By userId** — key pattern `gateway:ratelimit:user:{userId}:{window}`,
  applied to authenticated endpoints to protect against spam

Window is expressed in seconds and included in the key to ensure uniqueness per
time window.

---

### 4. Rate Limiter Not Extracted to Shared Package

The Gateway rate limiter is implemented locally in
`services/gateway/internal/middleware/ratelimit.go` and is not extracted to
`services/shared`.

**Rationale:** The Auth Service rate limiter has a different call signature
(`Allow(ctx, action, email)`) and different key semantics. Extracting a shared
implementation would require refactoring Auth, which is out of scope for this
module. If a third service requires rate limiting in the future, extraction to
`services/shared/pkg/ratelimit` with a generic `Allow(ctx, key, limit, ttl)`
interface is the correct approach at that point.

---

### 5. No Outgoing gRPC Metadata Interceptor

The Gateway does not use a shared outgoing interceptor to inject `x-user-id`
into gRPC metadata. Instead, `sender_id` and `requester_id` are passed
explicitly in each downstream request body/proto message.

**Rationale:** All downstream proto messages already include explicit identity
fields (`sender_id`, `requester_id`, `user_id`). A metadata interceptor would
add indirection without simplifying the call sites — handlers would still need
to extract userId from context and the interceptor would need to know which
methods require it. The explicit approach is more readable and consistent with
the existing service implementations.

---

### 6. No Retry Logic in gRPC Clients

Gateway gRPC clients do not implement retry logic. Failed downstream calls
return an error to the client immediately.

**Rationale:** The Message Service client already implements retry with
exponential backoff for `IsMember` calls because the Room Service may be
temporarily unavailable at startup. The Gateway is a different case — it is
the client-facing entry point, and retrying internally would mask real failures
and increase response latency unpredictably. Clients are expected to retry at
the application level if needed.

---

### 7. Rate Limiter Testing Deferred — Testcontainers

Unit tests for the rate limiter middleware are deferred until Testcontainers is
introduced into the project test suite.

**Rationale:** The rate limiter has a hard dependency on a real Valkey instance
— mocking the Valkey client at the interface level does not meaningfully test
the Lua script execution or the actual windowing behavior. Testing this
correctly requires a real Valkey container. When Testcontainers is introduced
(planned for Module 2 alongside the Sidecar PEP work), integration tests for
the rate limiter should be added covering: window expiry, limit enforcement,
atomicity under concurrent requests, and correct key namespacing for IP vs
userId limiters.

## Consequences

- Access tokens cannot be revoked before expiry — accepted for MVP
- Rate limiting state lives in Valkey and survives Gateway restarts within the
  TTL window
- The Auth Service rate limiter remains as-is and is not unified with the
  Gateway implementation
- All downstream gRPC calls include explicit identity fields — no shared
  metadata injection
- Rate limiter tests are tracked as a follow-up item for Module 2

## Follow-up

- `TODO(blacklist)`: Introduce access token blacklist in Auth Service and update
  Gateway JWT middleware to call `ValidateToken` gRPC for revocation checks.
  Track as a Module 2 task.
- `TODO(ratelimit-shared)`: Extract rate limiter to `services/shared/pkg/ratelimit`
  with a generic interface when a third service requires rate limiting.
- `TODO(ratelimit-tests)`: Add Testcontainers-based integration tests for the
  Gateway rate limiter covering window expiry, limit enforcement, atomicity, and
  key namespacing.
