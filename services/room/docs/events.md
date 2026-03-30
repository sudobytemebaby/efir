# Room Service Events

## NATS Streams

| Stream | Subjects | Purpose             |
| ------ | -------- | ------------------- |
| `ROOM` | `room.>` | Room-related events |

## Published Events

### room.membership.changed

Published when a user joins or leaves a room.

**Subject:** `room.membership.changed`

**Payload:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002",
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "action": "add",
  "recipient_ids": ["550e8400-e29b-41d4-a716-446655440000"]
}
```

**Consumers:**

- WebSocket Service - Notifies connected clients

**Delivery:**

- Acknowledgment-based
- Max deliver: configurable
- Ack wait: configurable

---

### room.updated

Published when a room is updated.

**Subject:** `room.updated`

**Payload:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002",
  "name": "New Room Name",
  "recipient_ids": ["550e8400-e29b-41d4-a716-446655440000"]
}
```

**Consumers:**

- WebSocket Service - Notifies connected clients

**Delivery:**

- Acknowledgment-based
