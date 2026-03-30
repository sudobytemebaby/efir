# Gateway Service API

All endpoints return JSON. Authentication uses `Authorization: Bearer <token>` header.

## Public Endpoints

### POST /auth/register

Register a new user account.

**Request:**

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
}
```

**Errors:**

- `409` - Email already registered
- `400` - Validation failed
- `429` - Rate limit exceeded

---

### POST /auth/login

Authenticate user credentials.

**Request:**

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
}
```

**Errors:**

- `404` - User not found
- `403` - Invalid password
- `429` - Rate limit exceeded

---

### POST /auth/logout

Invalidate a refresh token.

**Request:**

```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
}
```

**Response:** Empty (HTTP 204)

---

### POST /auth/refresh

Exchange refresh token for new tokens.

**Request:**

```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
}
```

**Response:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "bmV3IHJlZnJlc2ggdG9rZW4..."
}
```

---

## Protected Endpoints

All protected endpoints require `Authorization: Bearer <token>` header.

### GET /users/me

Get current user's profile.

**Response:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "display_name": "John Doe",
  "avatar_url": "https://example.com/avatar.jpg",
  "bio": "Hello world",
  "created_at": "2026-03-30T10:00:00Z"
}
```

---

### GET /users/{id}

Get user by ID.

**Response:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "display_name": "John Doe",
  "avatar_url": "https://example.com/avatar.jpg",
  "bio": "Hello world",
  "created_at": "2026-03-30T10:00:00Z"
}
```

**Errors:**

- `404` - User not found

---

### PATCH /users/me

Update current user's profile.

**Request:**

```json
{
  "display_name": "Jane Doe",
  "avatar_url": "https://example.com/new-avatar.jpg",
  "bio": "Updated bio"
}
```

**Response:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "display_name": "Jane Doe",
  "avatar_url": "https://example.com/new-avatar.jpg",
  "bio": "Updated bio",
  "created_at": "2026-03-30T10:00:00Z"
}
```

---

### POST /rooms

Create a new room.

**Request:**

```json
{
  "name": "General Chat",
  "description": "A place for general discussion"
}
```

**Response:**

```json
{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "General Chat",
  "description": "A place for general discussion",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-03-30T10:00:00Z"
}
```

---

### GET /rooms/{id}

Get room by ID.

**Response:**

```json
{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "General Chat",
  "description": "A place for general discussion",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-03-30T10:00:00Z"
}
```

**Errors:**

- `404` - Room not found
- `403` - Not a room member

---

### PATCH /rooms/{id}

Update room.

**Request:**

```json
{
  "name": "Updated Room Name",
  "description": "Updated description"
}
```

**Response:**

```json
{
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "Updated Room Name",
  "description": "Updated description",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-03-30T10:00:00Z"
}
```

**Errors:**

- `403` - Not the room owner

---

### DELETE /rooms/{id}

Delete room.

**Response:** Empty (HTTP 204)

**Errors:**

- `403` - Not the room owner

---

### POST /rooms/{id}/members

Add member to room.

**Request:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440002"
}
```

**Response:** Empty (HTTP 204)

**Errors:**

- `404` - Room or user not found
- `403` - Not authorized to add members

---

### DELETE /rooms/{id}/members/{userId}

Remove member from room.

**Response:** Empty (HTTP 204)

**Errors:**

- `403` - Not authorized to remove members
- `404` - Room or member not found

---

### POST /rooms/{id}/messages

Send a message to a room.

**Request:**

```json
{
  "reply_to_id": "550e8400-e29b-41d4-a716-446655440002",
  "type": "TEXT",
  "content": {
    "text": "Hello, world!"
  }
}
```

**Response:**

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440003",
  "room_id": "550e8400-e29b-41d4-a716-446655440001",
  "sender_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "TEXT",
  "is_deleted": false,
  "created_at": "2026-03-30T10:00:00Z",
  "content": { "text": "Hello, world!" }
}
```

**Errors:**

- `403` - Not a room member
- `400` - Invalid content type

---

### GET /rooms/{id}/messages

Get messages from a room.

**Query Parameters:**

- `cursor` (optional): Message ID to start after
- `limit` (optional): Number of messages (1-100, default 20)

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
      "content": { "text": "Hello, world!" }
    }
  ],
  "next_cursor": "550e8400-e29b-41d4-a716-446655440002"
}
```

**Errors:**

- `403` - Not a room member

---

### POST /auth/ws-ticket

Create a WebSocket authentication ticket.

**Response:**

```json
{
  "ticket": "one-time-ticket-string"
}
```

**Usage:**

1. Client connects to WebSocket with `?ticket=<ticket>` or `X-Ws-Ticket` header
2. WebSocket service validates ticket via gateway's `/auth/validate`

---

### GET /health

Liveness probe.

**Response:**

```json
{ "status": "ok" }
```

---

### GET /ready

Readiness probe.

**Response:**

```json
{ "status": "ok" }
```
