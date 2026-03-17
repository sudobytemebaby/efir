# ADR-011: Room Membership Event Publishing Strategy

## Status

Accepted

## Context

The Room Service publishes `room.membership.changed` events to NATS JetStream
when members are added or removed. The WebSocket Service subscribes to these
events to maintain an up-to-date view of room membership and route real-time
messages to the correct connections.

The question is: what should happen when the NATS publish fails?

Two failure modes need to be handled:

1. **NATS is temporarily unavailable** — the membership change succeeds in
   Postgres but the event is lost. The WebSocket Service has a stale membership
   view until the affected users reconnect.

2. **The caller retries after a partial success** — the member was added to
   Postgres on the first attempt, but the caller received an error (due to NATS
   failure) and retries. The second attempt hits a conflict in the repo layer,
   which previously propagated as an internal error, making a legitimate retry
   look like a failure.

This is different from the Auth Service's `PublishUserRegistered` event, where
losing the event is recoverable — the user can still log in, and profile
creation can be reconciled later. A lost `room.membership.changed` event causes
a visible, real-time bug: newly added members don't receive messages until they
reconnect and the WebSocket Service refreshes its state.

## Decision

### 1. Make `AddMember` idempotent at the service layer

If the repository returns `ErrMemberAlreadyExists`, the service treats it as a
successful no-op rather than an error. This mirrors the pattern already used in
`CreateRoom` and fixes the retry problem — a client that retries after a partial
success gets a clean response instead of an internal error.

### 2. Swallow NATS publish failures with structured logging

If `PublishMembershipChanged` fails, the service logs the error at ERROR level
with full context (room_id, user_id, action) but does not fail the request.
The membership change is committed to Postgres; the event delivery is
best-effort.

```go
if err := s.publisher.PublishMembershipChanged(...); err != nil {
    slog.Error("failed to publish membership changed event, event may be lost",
        "room_id", roomID,
        "user_id", userID,
        "action", action,
        "error", err,
    )
}
```

This is consistent with how the Auth Service handles `PublishUserRegistered`
failures today.

### 3. Document the known limitation

The WebSocket Service must not treat NATS events as the sole source of truth
for room membership. On client reconnect, it must re-fetch current membership
from the Room Service via gRPC rather than replying solely on the event stream.
This makes the system resilient to lost events without requiring infrastructure
changes.

## Alternatives Considered

### Return the error (previous behavior)

Fail the `AddMember` request if NATS publish fails.

- **Pro**: The caller knows something went wrong.
- **Con**: The membership change is already committed to Postgres. The caller
  has no way to distinguish "member not added" from "member added but event
  lost". Retrying creates an `ErrMemberAlreadyExists` that bubbles up as an
  internal error. The surface-level correctness hides a deeper inconsistency.

Rejected because it combines two problems — partial success and event loss —
into a single opaque error, making client behavior unpredictable.

### Transactional outbox pattern

Write the event to a Postgres `outbox` table in the same transaction as the
membership change. A background worker polls the table and publishes to NATS,
guaranteeing at-least-once delivery.

- **Pro**: Events are never lost. NATS downtime is fully tolerated.
- **Con**: Requires a new table, a background worker, and dead-letter handling.
  Significant complexity for MVP.

Rejected for MVP. This is the correct long-term solution and should be
implemented in Module 2 alongside the Sidecar PEP work, when the system moves
toward production-grade reliability guarantees.

### Return the existing user on `ErrMemberAlreadyExists` without publishing

Silently succeed on conflict without re-publishing the event.

- **Pro**: Simple, no duplicate events.
- **Con**: If the first attempt failed after Postgres commit but before NATS
  publish, the event is permanently lost with no indication. The retry silently
  swallows the missing event rather than attempting recovery.

Rejected. Acceptable as behavior on idempotent retries where the event was
already published, but not as a general strategy for handling publish failures.

## Consequences

- `AddMember` and `RemoveMember` are idempotent — retries are safe.
- Membership events may be lost if NATS is unavailable at the moment of the
  operation. This is a known, documented limitation of the MVP.
- The WebSocket Service must treat room membership events as hints, not as
  authoritative state. Client reconnect logic must re-fetch membership from the
  Room Service.
- The transactional outbox pattern is deferred to Module 2 as a tracked
  follow-up, at which point this ADR should be superseded.

## Follow-up

- `TODO(outbox)`: Replace best-effort NATS publish in `AddMember` and
  `RemoveMember` with a transactional outbox to guarantee at-least-once
  delivery. Track as a Module 2 task.
- WebSocket Service reconnect handler must call `RoomService.GetRoomMembers`
  on connection establishment to sync state, not rely on replaying missed
  events.
