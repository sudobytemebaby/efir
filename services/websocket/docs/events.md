# WebSocket Service Events

## NATS Subscriptions

The WebSocket Service subscribes to events from the MESSAGE and ROOM streams.

## Subscribed Events

### message.created

Received from `MESSAGE` stream when a new message is created.

**Subject:** `message.created`

**Payload:**

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440003",
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "sender_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "text",
  "recipient_ids": ["550e8400-e29b-41d4-a716-446655440001"],
  "created_at": "2026-03-30T10:00:00Z"
}
```

**Action:** Broadcasts `message.created` event to all clients in the room.

---

### room.membership.changed

Received from `ROOM` stream when a user joins or leaves a room.

**Subject:** `room.membership.changed`

**Payload:**

```json
{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440002",
  "joined": true
}
```

**Action:** Broadcasts `room.membership.changed` event to all clients in the room.

---

### room.updated

Received from `ROOM` stream when room settings are changed.

**Subject:** `room.updated`

**Payload:**

```json
{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "updated_by": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Action:** Broadcasts `room.updated` event to all clients in the room.

---

## Consumer Configuration

| Consumer                 | Stream  | Subject                   | Ack Policy |
| ------------------------ | ------- | ------------------------- | ---------- |
| `ws-svc-message-created` | MESSAGE | `message.created`         | Explicit   |
| `ws-svc-room-membership` | ROOM    | `room.membership.changed` | Explicit   |
| `ws-svc-room-updated`    | ROOM    | `room.updated`            | Explicit   |

**Consumer Settings:**

- Max deliver: 5
- Ack wait: 30s
- Durable: Yes (survives restarts)
