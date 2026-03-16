# ADR-008: Rate Limiting Strategy

## Status

Accepted

## Context

Auth service exposes public-facing endpoints (`Register`, `Login`) that are vulnerable to brute-force attacks and credential stuffing. We need a mechanism to limit the number of requests per identity within a time window.

## Decision

Implement per-email fixed window rate limiting at the service layer using Valkey as the counter store.

**Algorithm:** Fixed window counter — N requests per M minutes per email per action. The window resets after TTL expires.

**Storage:** Valkey (already a project dependency for refresh token storage). Keys follow the pattern `auth:ratelimit:{action}:{email}` with TTL equal to the window duration.

**Layer:** Service layer via an injected `Limiter` interface. The handler maps `ErrRateLimitExceeded` to `codes.Unavailable`.

**Configuration:** Limits are configurable via environment variables (`RATE_LIMIT_REQUESTS`, `RATE_LIMIT_WINDOW`) with sane defaults (10 requests / 1 minute).

## Rationale

- **Valkey is already present**: no new infrastructure dependency
- **Per-email granularity**: limits apply to the target identity, not the caller IP — more relevant for credential attacks
- **Service layer**: rate limiting is a business rule ("too many login attempts"), not a transport concern
- **Interface injection**: `Limiter` is an interface, making it mockable in tests and swappable in the future

## Alternatives Considered

- **Traefik rate limiting**: already configured globally in `middleware.yml`. Suitable for general traffic shaping but operates at the IP level and cannot enforce per-email semantics.
- **Sliding window**: more accurate, prevents bursting at window boundaries, but requires a sorted set in Valkey and more complex logic. Overkill for auth rate limiting at current scale.
- **Token bucket**: good for smooth rate limiting, but harder to reason about for "N attempts per minute" semantics that users understand.

## Known Limitations

There is a race condition between `INCR` and `EXPIRE`: if the service crashes after incrementing but before setting the TTL, the key will persist indefinitely and the email will be permanently blocked.

TODO: replace `INCR` + conditional `EXPIRE` with a Lua script or `SET key 1 EX {ttl} NX` pattern to make the operation atomic.

## Consequences

- `Register` and `Login` are protected against brute-force and credential stuffing
- Rate limit state is stored in Valkey and survives service restarts within the TTL window
- Limits are tunable per deployment without code changes
- `Limiter` interface enables unit testing without a real Valkey instance
