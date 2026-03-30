# WebSocket Service Overview

## Responsibilities

The WebSocket Service is responsible for:

- Managing WebSocket connections from clients
- Authenticating clients via one-time tickets (Valkey)
- Subscribing to NATS events (messages, room updates)
- Broadcasting real-time events to connected clients
- Room subscription management
- Connection health (ping/pong)

## Architecture

### Hub Pattern

The service uses a central Hub to manage all connections and room subscriptions:

```
internal/hub/hub.go
├── rooms     map[string]map[string][]Conn  // roomID → (userID → [conns])
├── userIDs   map[Conn]string               // conn → userID
├── connRooms map[Conn]map[string]struct{} // conn → [roomIDs]
```

### Request Flow

```
NATS Event (message.created)
        │
        ▼
┌─────────────────────────┐
│   Subscriber (events)    │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│   Hub.BroadcastToRoom   │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  Concurrent write to     │
│  all conns in room      │
└─────────────────────────┘
```

### Connection Lifecycle

1. **Connect** - Client connects with ticket
2. **Validate** - Ticket validated via Valkey (GETDEL)
3. **Register** - Connection registered with Hub
4. **Subscribe** - Client joins rooms via `subscribe` message
5. **Receive** - Client receives real-time events
6. **Unsubscribe** - Client leaves rooms via `unsubscribe` message
7. **Disconnect** - Connection closed, cleaned up from all rooms

### NATS Event Flow

| Event                     | Source          | Action            |
| ------------------------- | --------------- | ----------------- |
| `message.created`         | Message Service | Broadcast to room |
| `room.membership.changed` | Room Service    | Broadcast to room |
| `room.updated`            | Room Service    | Broadcast to room |

### Error Handling

| Error Code          | Meaning              | Action   |
| ------------------- | -------------------- | -------- |
| `message_too_large` | Exceeds read limit   | Continue |
| `invalid_json`      | Cannot parse JSON    | Continue |
| `invalid_payload`   | Payload parse error  | Continue |
| `invalid_room_id`   | Invalid UUID format  | Continue |
| `unknown_type`      | Unknown message type | Continue |

## Dependencies

| Dependency      | Purpose                    |
| --------------- | -------------------------- |
| Valkey          | Ticket validation (GETDEL) |
| NATS            | Event subscriptions        |
| Gateway Service | Ticket creation endpoint   |
