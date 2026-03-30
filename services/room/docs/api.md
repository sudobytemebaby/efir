# Room Service API

gRPC service definition in `proto/efir/room/room.proto`

## RoomService

### CreateRoom

Create a new chat room.

**Request:**

```json
{
  "name": "General Chat",
  "type": "GROUP",
  "created_by": "550e8400-e29b-41d4-a716-446655440000",
  "participant_id": "660e8400-e29b-41d4-a716-446655440001"
}
```

**Validation:**

- `name`: non-empty string
- `type`: `DIRECT` or `GROUP`
- `created_by`: valid UUID

**Response:**

```json
{
  "room": {
    "room_id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "General Chat",
    "type": "GROUP",
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

**Errors:**

- `INVALID_ARGUMENT` - Validation failed

---

### GetRoom

Get room details by ID.

**Request:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002"
}
```

**Response:**

```json
{
  "room": {
    "room_id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "General Chat",
    "type": "GROUP",
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

**Errors:**

- `NOT_FOUND` - Room not found

---

### UpdateRoom

Update room name.

**Request:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Updated Room Name"
}
```

**Response:**

```json
{
  "room": {
    "room_id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "Updated Room Name",
    "type": "GROUP",
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-20T14:22:00Z"
  }
}
```

**Errors:**

- `NOT_FOUND` - Room not found
- `PERMISSION_DENIED` - Not room owner

---

### DeleteRoom

Delete a room.

**Request:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** Empty (HTTP 204)

**Errors:**

- `NOT_FOUND` - Room not found
- `PERMISSION_DENIED` - Not room owner

---

### AddMember

Add a user to a room.

**Request:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002",
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** Empty (HTTP 204)

**Errors:**

- `NOT_FOUND` - Room not found
- `PERMISSION_DENIED` - Not authorized to add members

---

### RemoveMember

Remove a user from a room.

**Request:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002",
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** Empty (HTTP 204)

**Errors:**

- `NOT_FOUND` - Room or member not found
- `PERMISSION_DENIED` - Not authorized to remove members

---

### GetRoomMembers

Get list of user IDs in a room.

**Request:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002"
}
```

**Response:**

```json
{
  "user_ids": [
    "550e8400-e29b-41d4-a716-446655440000",
    "660e8400-e29b-41d4-a716-446655440001"
  ]
}
```

**Errors:**

- `NOT_FOUND` - Room not found

---

### IsMember

Check if a user is a member of a room.

**Request:**

```json
{
  "room_id": "770e8400-e29b-41d4-a716-446655440002",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:**

```json
{
  "is_member": true
}
```

**Errors:**

- `NOT_FOUND` - Room not found
