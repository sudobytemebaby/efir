# WebSocket Service API

## Connection

### GET /ws

Connect to WebSocket with ticket authentication.

**Query Parameters:**

- `ticket` (required): One-time authentication ticket from gateway
- `room_id` (optional): Initial room to subscribe to

**Handshake:**

```
GET /ws?ticket=abc123&room_id=550e8400-e29b-41d4-a716-446655440001 HTTP/1.1
Upgrade: websocket
```

**Authentication:**

- Ticket validated via Valkey GETDEL on `gateway:ws:ticket:{ticket}`
- Returns user_id on success
- Ticket is consumed (one-time use)

**Response:** WebSocket upgrade on success, HTTP 401 on failure

---

## Client Messages

### Subscribe

Join a room to receive events.

```json
{
  "type": "subscribe",
  "payload": {
    "room_id": "550e8400-e29b-41d4-a716-446655440001"
  }
}
```

**Response:** None (passive receive)

---

### Unsubscribe

Leave a room.

```json
{
  "type": "unsubscribe",
  "payload": {
    "room_id": "550e8400-e29b-41d4-a716-446655440001"
  }
}
```

**Response:** None

---

### Ping

Keepalive (optional - ping/pong handled automatically).

```json
{
  "type": "ping",
  "payload": {}
}
```

**Response:**

```json
{
  "type": "pong",
  "payload": {}
}
```

---

## Server Messages

### message.created

New message in a room.

```json
{
  "type": "message.created",
  "payload": {
    "message_id": "550e8400-e29b-41d4-a716-446655440003",
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "content": {
      "text": "Hello, world!"
    }
  }
}
```

---

### room.membership.changed

User joined or left a room.

```json
{
  "type": "room.membership.changed",
  "payload": {
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "550e8400-e29b-41d4-a716-446655440002",
    "joined": true
  }
}
```

---

### room.updated

Room settings changed.

```json
{
  "type": "room.updated",
  "payload": {
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "updated_by": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

### Error

Error response from server.

```json
{
  "type": "error",
  "payload": {
    "code": "invalid_room_id",
    "message": "Room ID must be a valid UUID"
  }
}
```

**Error Codes:**

- `message_too_large` - Message exceeds read limit
- `invalid_json` - Cannot parse message JSON
- `invalid_payload` - Cannot parse payload JSON
- `invalid_room_id` - Room ID not valid UUID
- `unknown_type` - Unrecognized message type

---

## Health Endpoint

### GET /health

Liveness probe.

**Response:**

```json
{ "status": "ok" }
```
