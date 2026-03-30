# Message Service API

gRPC service definition in `proto/efir/message/message.proto`

## MessageService

### SendMessage

Send a message to a room.

**Request:**

```json
{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "sender_id": "550e8400-e29b-41d4-a716-446655440000",
  "reply_to_id": "550e8400-e29b-41d4-a716-446655440002",
  "type": "TEXT",
  "content": {
    "text": "Hello, world!"
  }
}
```

**Validation:**

- `room_id`: Required UUID
- `sender_id`: Required UUID
- `reply_to_id`: Optional UUID (must exist, not deleted, same room)
- `type`: Required enum (TEXT, IMAGE, VIDEO, VIDEO_NOTE, VOICE, AUDIO, FILE, STICKER, EVENT)
- `content`: Must match the message type (e.g., TEXT requires `text` field)

**Response:**

```json
{
  "message": {
    "message_id": "550e8400-e29b-41d4-a716-446655440003",
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "sender_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "TEXT",
    "is_deleted": false,
    "reply_to": {
      "message_id": "550e8400-e29b-41d4-a716-446655440002",
      "sender_id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "TEXT",
      "content": { "text": "Original message" }
    },
    "created_at": "2026-03-30T10:00:00Z",
    "updated_at": "2026-03-30T10:00:00Z",
    "content": {
      "text": "Hello, world!"
    }
  }
}
```

**Errors:**

- `PERMISSION_DENIED` - Sender is not a room member
- `INVALID_ARGUMENT` - Invalid reply target or content type mismatch

---

### GetMessages

List messages in a room with cursor-based pagination.

**Request:**

```json
{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000",
  "cursor": "550e8400-e29b-41d4-a716-446655440003",
  "limit": 20
}
```

**Validation:**

- `room_id`: Required UUID
- `requester_id`: Required UUID
- `cursor`: Optional UUID (message ID to start after)
- `limit`: Required integer (1-100, default 20)

**Response:**

```json
{
  "messages": [
    {
      "message_id": "550e8400-e29b-41d4-a716-446655440003",
      "room_id": "550e8400-e29b-41d4-a716-446655440001",
      "sender_id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "TEXT",
      "is_deleted": false,
      "created_at": "2026-03-30T10:00:00Z",
      "updated_at": "2026-03-30T10:00:00Z",
      "content": { "text": "Hello, world!" }
    }
  ],
  "next_cursor": "550e8400-e29b-41d4-a716-446655440002"
}
```

**Notes:**

- Messages are ordered by `created_at DESC, id DESC`
- Deleted messages are excluded
- Includes reply-to previews

**Errors:**

- `PERMISSION_DENIED` - Requester is not a room member

---

### GetMessageById

Fetch a single message by ID.

**Request:**

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440003",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:**

```json
{
  "message": {
    "message_id": "550e8400-e29b-41d4-a716-446655440003",
    "room_id": "550e8400-e29b-41d4-a716-446655440001",
    "sender_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "TEXT",
    "is_deleted": false,
    "created_at": "2026-03-30T10:00:00Z",
    "updated_at": "2026-03-30T10:00:00Z",
    "content": { "text": "Hello, world!" }
  }
}
```

**Errors:**

- `NOT_FOUND` - Message not found
- `PERMISSION_DENIED` - Requester is not a room member

---

### DeleteMessage

Soft-delete a message (sender only).

**Request:**

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440003",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** Empty (HTTP 200)

**Notes:**

- Only the original sender can delete their message
- Soft-delete sets `deleted_at` timestamp
- No membership check - allows deletion even after leaving the room

**Errors:**

- `NOT_FOUND` - Message not found
- `PERMISSION_DENIED` - Requester is not the message sender
