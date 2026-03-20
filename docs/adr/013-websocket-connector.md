# ADR-013: WebSocket Connector

## Status

Proposed

## Context

Real-time messaging requires pushing events to connected clients. The existing services (message, room) publish events to NATS JetStream, but clients need a mechanism to receive these events in real-time without polling.

## Decision

Implement a WebSocket Connector service that:

1. Maintains persistent WebSocket connections with authenticated clients
2. Subscribes to NATS JetStream events for messages, room membership changes, and room updates
3. Broadcasts relevant events to connected clients based on their room subscriptions

### Authentication Flow

We use a **one-time ticket + forward auth** pattern (NOT JWT in URL):

1. Client authenticates via existing auth service, receives JWT
2. Client calls `POST /auth/ws-ticket` on Gateway with `X-User-Id` header
3. Gateway generates a one-time UUID ticket, stores it in Valkey with 30s TTL:
   ```
   Key: gateway:ws:ticket:{ticket}
   Value: {user_id}
   TTL: 30s
   ```
4. Client connects to WebSocket Connector via Traefik: `GET /ws?ticket={ticket}&room_id={room_id}`
5. Traefik forward auth middleware calls `GET /auth/validate` on Gateway
6. Gateway performs `GETDEL` on Valkey to atomically consume the ticket, returns `X-User-Id`
7. If ticket valid, WebSocket connection established; otherwise rejected

### WebSocket Envelope Format

All messages use JSON envelope:

```json
{
  "type": "message.created",
  "payload": { ... }
}
```

#### Server-to-Client Types

- `message.created` - New message in subscribed room
- `room.membership.changed` - User joined/left room
- `room.updated` - Room metadata changed
- `error` - Error response

#### Client-to-Server Types

- `subscribe` - Subscribe to room
- `unsubscribe` - Unsubscribe from room
- `ping` - Keepalive (server responds with `pong`)

### Hub Pattern

The WebSocket Connector uses a Hub pattern to manage connections:

```
rooms: map[roomID]map[userID][]Conn
userIDs: map[Conn]userID
connRooms: map[Conn]map[roomID]struct{}  // reverse index for cleanup
```

#### Thread Safety

- All hub operations use channels for communication with the hub goroutine
- `sendToRoom` acquires RLock, copies connections to a local slice, then releases lock before iterating
- This prevents data races between iteration and concurrent `removeConn`

#### Graceful Shutdown

- `hub.Run(ctx)` accepts context, exits on `ctx.Done()`
- On connection close, `Disconnect(conn)` removes from ALL subscribed rooms via reverse index

### NATS JetStream Integration

The service uses JetStream consumers (NOT core NATS subscriptions) for guaranteed delivery:

```go
msgConsumer, _ := sharednats.ProvisionConsumerWithRetry(ctx, js, StreamMessage, MessageCreatedConsumer())
msgSub, _ := msgConsumer.Consume(s.handleMessageCreated)
```

Streams are created by publishing services (message, room), not by WebSocket Connector.

### Event Payloads

#### message.created

```json
{
  "type": "message.created",
  "payload": {
    "message_id": "uuid",
    "room_id": "uuid",
    "user_id": "uuid",
    "content": { ... }
  }
}
```

#### room.membership.changed

```json
{
  "type": "room.membership.changed",
  "payload": {
    "room_id": "uuid",
    "user_id": "uuid",
    "joined": true
  }
}
```

#### room.updated

```json
{
  "type": "room.updated",
  "payload": {
    "room_id": "uuid",
    "updated_by": "uuid"
  }
}
```

## Consequences

### Positive

- One-time tickets prevent replay attacks and URL-based token leakage
- JetStream consumers ensure no events lost during service restarts
- Hub pattern cleanly separates connection management from business logic
- Forward auth keeps auth logic in Gateway, not duplicated in WebSocket Connector

### Negative

- Additional network hop (Traefik → Gateway → Valkey) for auth validation
- Ticket TTL must be long enough for network latency but short enough to prevent reuse

## References

- [ADR-011: Room Membership Events](011-room-membership-events.md) - Event schemas
- [nhooyr.io/websocket](https://nhooyr.io/websocket) - WebSocket library
- [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream) - Persistence layer
