# ADR-013: WebSocket Connector

## Status

Accepted

## Context

Real-time messaging requires pushing events to connected clients. The existing services (message, room) publish events to NATS JetStream, but clients need a mechanism to receive these events in real-time without polling.

## Decision

Implement a WebSocket Connector service that:

1. Maintains persistent WebSocket connections with authenticated clients
2. Subscribes to NATS JetStream events for messages, room membership changes, and room updates
3. Broadcasts relevant events to connected clients based on their room subscriptions

### Authentication Flow

We use a **one-time ticket + forward auth** pattern (NOT JWT in URL):

1. Client authenticates via existing auth service, receives JWT in `access_token` cookie.
2. Client calls `POST /auth/ws-ticket` on Gateway. Gateway validates the JWT cookie via the `JWTAuth` middleware and extracts the `user_id` from the token claims.
3. Gateway generates a one-time UUID ticket, stores it in Valkey with a configurable TTL (default 30s):
   ```
   Key: gateway:ws:ticket:{ticket}
   Value: {user_id}
   TTL: configured via WS_TICKET_TTL
   ```
4. Client connects to WebSocket Connector via nginx: `wss://ws.localhost/ws?ticket={ticket}&room_id={room_id}`
5. nginx `auth_request` directive issues an internal subrequest to `GET /auth/validate` on the Gateway, forwarding the ticket via the `X-Ws-Ticket` request header.
6. Gateway handler (`/auth/validate`) performs `GETDEL` on Valkey to atomically consume the ticket, then sets `X-User-Id` in the response header.
7. If the ticket is valid, nginx extracts `X-User-Id` from the subrequest response and forwards it to the WebSocket Connector as a request header. The WebSocket connection is established. Otherwise, nginx returns 401 and the connection is rejected.

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

The Hub is a single-goroutine actor. All state mutations (register, unregister, disconnect, broadcast) are sent through buffered channels and processed exclusively by `Hub.Run`. No mutexes are used. This eliminates data races by design — there is only one goroutine reading and writing the internal maps at any time.

`sendToRoom` is called from inside `Run` and iterates over connections directly without any locking.

Slow clients are handled without stalling the hub: `Conn.Send` is non-blocking (buffered channel). If the buffer is full, the connection is closed asynchronously in a separate goroutine.

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

- Additional network hop (nginx → Gateway → Valkey) for auth validation
- Ticket TTL must be long enough for network latency but short enough to prevent reuse

## References

- [ADR-011: Room Membership Events](011-room-membership-events.md) - Event schemas
- [nhooyr.io/websocket](https://nhooyr.io/websocket) - WebSocket library
- [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream) - Persistence layer
