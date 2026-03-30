# Message Service Events

## NATS Streams

| Stream    | Subjects    | Purpose        |
| --------- | ----------- | -------------- |
| `MESSAGE` | `message.>` | Message events |

## Published Events

### message.created

Published when a new message is created in a room.

**Subject:** `message.created`

**Payload:**

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440003",
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "sender_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "text",
  "recipient_ids": [
    "550e8400-e29b-41d4-a716-446655440001",
    "550e8400-e29b-41d4-a716-446655440002"
  ],
  "created_at": "2026-03-30T10:00:00Z"
}
```

**Consumers:**

- WebSocket Service - Real-time message delivery to connected clients

**Delivery:**

- Fire-and-forget (async)
- Failures are logged but do not fail the message creation request
