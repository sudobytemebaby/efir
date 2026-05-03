# API Reference

Complete HTTP API reference for the Efir Gateway service. All endpoints are served at `http://api.localhost`.

## Overview

- **Content-Type:** `application/json` for all requests and responses
- **Authentication:** HttpOnly cookies (`access_token`, `refresh_token`) set on login/register
- **Rate Limiting:** 100 requests per 60 seconds per authenticated user (configurable)
- **Max Body Size:** 1 MB (configurable via `MAX_BODY_SIZE`)
- **Request ID:** Propagated via `X-Request-ID` header; auto-generated if absent

## Error Handling

All errors use a consistent JSON format:

```json
{
  "error": "human-readable description",
  "code": "ERROR_CODE"
}
```

### Error Codes

| Code               | HTTP Status | Description                          |
|--------------------|-------------|--------------------------------------|
| `NOT_FOUND`        | 404         | Requested resource does not exist    |
| `ALREADY_EXISTS`   | 409         | Resource already exists (duplicate)  |
| `PERMISSION_DENIED`| 403         | Insufficient permissions             |
| `UNAUTHENTICATED`  | 401         | Missing or invalid credentials       |
| `INVALID_ARGUMENT` | 400         | Invalid request parameters           |
| `UNAVAILABLE`      | 503         | Service temporarily unavailable      |
| `INTERNAL`         | 500         | Internal server error                |
| `RATE_LIMITED`     | 429         | Rate limit exceeded                  |

---

## Authentication

Authentication uses cookie-based JWT tokens. On successful login or register, the gateway sets two HttpOnly cookies:

- `access_token` -- JWT, 15-minute TTL, path `/`
- `refresh_token` -- opaque token, 30-day TTL, path `/auth/session`

In production, cookies have `Secure: true`. In development, `Secure: false`.

### POST /auth/register

Create a new account. A user profile with a random username is automatically created via NATS event.

**Auth:** None

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

**Validation:**
- `email`: valid email format, max 255 characters
- `password`: 8-72 characters (bcrypt limit)

**Response:** `200 OK`
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Cookies set:** `access_token`, `refresh_token`

**Errors:**
- `409 ALREADY_EXISTS` -- email already registered
- `400 INVALID_ARGUMENT` -- validation failure

---

### POST /auth/login

Authenticate with email and password.

**Auth:** None

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

**Response:** `200 OK`
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Cookies set:** `access_token`, `refresh_token`

**Errors:**
- `401 UNAUTHENTICATED` -- invalid credentials
- `400 INVALID_ARGUMENT` -- validation failure

---

### POST /auth/session/refresh

Rotate tokens using the `refresh_token` cookie.

**Auth:** `refresh_token` cookie

**Request:** No body required.

**Response:** `204 No Content`

**Cookies set:** New `access_token`, new `refresh_token`

**Errors:**
- `401 UNAUTHENTICATED` -- missing or invalid refresh token

---

### POST /auth/session/logout

Invalidate the current session and clear cookies.

**Auth:** `refresh_token` cookie (best-effort; clears cookies regardless)

**Request:** No body required.

**Response:** `204 No Content`

**Cookies cleared:** `access_token`, `refresh_token`

---

### GET /auth/me

Get the currently authenticated user's ID from JWT claims.

**Auth:** `access_token` cookie (JWT)

**Response:** `200 OK`
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

### POST /auth/ws-ticket

Generate a one-time ticket for WebSocket authentication. The ticket is a UUID stored in Valkey and consumed on use (GETDEL pattern).

**Auth:** `access_token` cookie (JWT)

**Response:** `201 Created`
```json
{
  "ticket": "d4f5e6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90"
}
```

**Notes:**
- Ticket TTL is configurable (default: 30s)
- Each ticket can only be used once
- Use this ticket as a query parameter when connecting to WebSocket

---

## Users

All user endpoints require JWT authentication via `access_token` cookie.

### GET /users/me

Get the current user's full profile.

**Auth:** JWT

**Response:** `200 OK`
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "bright-falcon-42",
  "display_name": "bright-falcon-42",
  "avatar_url": null,
  "bio": null,
  "created_at": "2026-03-15T10:30:00Z",
  "updated_at": "2026-03-15T10:30:00Z"
}
```

**Notes:**
- `username` is auto-generated on registration (adjective-noun-number pattern, immutable)
- `display_name` defaults to the username; can be changed
- `avatar_url` and `bio` are null until updated

---

### GET /users/{id}

Get any user's public profile by UUID.

**Auth:** JWT

**Path parameters:**
- `id` -- User UUID

**Response:** `200 OK` (same format as `/users/me`)

**Errors:**
- `404 NOT_FOUND` -- user does not exist

---

### PATCH /users/me

Update the current user's profile. Only provided fields are updated.

**Auth:** JWT

**Request:**
```json
{
  "display_name": "Alice",
  "avatar_url": "https://cdn.example.com/avatar.png",
  "bio": "Hello from Efir!"
}
```

All fields are optional. Omitted fields are not changed.

**Response:** `200 OK` (full user object with updated fields)

**Errors:**
- `400 INVALID_ARGUMENT` -- invalid field values

---

## Rooms

All room endpoints require JWT authentication. The `requester_id` is automatically injected from the JWT by the gateway.

### GET /rooms

List all rooms the current user is a member of.

**Auth:** JWT

**Response:** `200 OK`
```json
[
  {
    "room_id": "...",
    "name": "Project Chat",
    "type": "ROOM_TYPE_GROUP",
    "created_by": "...",
    "created_at": "...",
    "updated_at": "..."
  }
]
```

Returns an empty array if the user has no rooms.

---

### POST /rooms

Create a new room. The creator is automatically added as the `owner`.

**Auth:** JWT

**Request:**
```json
{
  "name": "Project Chat",
  "type": "ROOM_TYPE_GROUP",
  "participant_id": ""
}
```

**Fields:**
- `name` (required): room name, min 1 character
- `type`: `ROOM_TYPE_DIRECT` (1:1) or `ROOM_TYPE_GROUP`
- `participant_id`: for direct rooms, the other user's UUID

**Response:** `201 Created`
```json
{
  "room_id": "...",
  "name": "Project Chat",
  "type": "ROOM_TYPE_GROUP",
  "created_by": "...",
  "created_at": "...",
  "updated_at": "..."
}
```

**Side effects:** Publishes `room.created` NATS event.

---

### GET /rooms/{id}

Get room details by ID.

**Auth:** JWT

**Path parameters:**
- `id` -- Room UUID

**Response:** `200 OK` (room object)

**Errors:**
- `404 NOT_FOUND` -- room does not exist

---

### PATCH /rooms/{id}

Update room metadata. Only the room owner can update.

**Auth:** JWT

**Path parameters:**
- `id` -- Room UUID

**Request:**
```json
{
  "name": "New Room Name"
}
```

**Response:** `200 OK` (updated room object)

**Side effects:** Publishes `room.updated` NATS event.

**Errors:**
- `403 PERMISSION_DENIED` -- requester is not the room owner
- `404 NOT_FOUND` -- room does not exist

---

### DELETE /rooms/{id}

Delete a room. Only the room owner can delete.

**Auth:** JWT

**Path parameters:**
- `id` -- Room UUID

**Response:** `204 No Content`

**Side effects:** Publishes `room.deleted` NATS event. Cascades to room members.

**Errors:**
- `403 PERMISSION_DENIED` -- requester is not the room owner
- `404 NOT_FOUND` -- room does not exist

---

### POST /rooms/{id}/members

Add a user to a room. Requires the requester to be a member of the room.

**Auth:** JWT

**Path parameters:**
- `id` -- Room UUID

**Request:**
```json
{
  "user_id": "target-user-uuid"
}
```

**Response:** `204 No Content`

**Side effects:** Publishes `room.membership.changed` NATS event with `action: "added"`.

**Errors:**
- `403 PERMISSION_DENIED` -- requester is not a member
- `409 ALREADY_EXISTS` -- user is already a member
- `404 NOT_FOUND` -- room or user does not exist

---

### DELETE /rooms/{id}/members/{userId}

Remove a user from a room. The room owner can remove anyone; members can only remove themselves.

**Auth:** JWT

**Path parameters:**
- `id` -- Room UUID
- `userId` -- User UUID to remove

**Response:** `204 No Content`

**Side effects:** Publishes `room.membership.changed` NATS event with `action: "removed"`.

**Errors:**
- `403 PERMISSION_DENIED` -- insufficient permissions
- `404 NOT_FOUND` -- room, user, or membership does not exist

---

## Messages

All message endpoints require JWT authentication.

### POST /rooms/{id}/messages

Send a message to a room. The sender must be a member of the room.

**Auth:** JWT

**Path parameters:**
- `id` -- Room UUID

**Request (text message):**
```json
{
  "type": "MESSAGE_TYPE_TEXT",
  "text": {
    "text": "Hello, world!"
  }
}
```

**Request (with reply):**
```json
{
  "type": "MESSAGE_TYPE_TEXT",
  "reply_to_id": "original-message-uuid",
  "text": {
    "text": "This is a reply"
  }
}
```

**Content types by message type:**

| Type                       | Content field  | Required fields                                               |
|----------------------------|----------------|---------------------------------------------------------------|
| `MESSAGE_TYPE_TEXT`        | `text`         | `text` (1-4096 chars)                                         |
| `MESSAGE_TYPE_IMAGE`       | `media`        | `file_id`, `mime_type`, `file_size`, `width`, `height`        |
| `MESSAGE_TYPE_VIDEO`       | `media`        | `file_id`, `mime_type`, `file_size`, `width`, `height`        |
| `MESSAGE_TYPE_VIDEO_NOTE`  | `video_note`   | `file_id`, `mime_type`, `file_size`, `duration_sec`, `width`, `height` |
| `MESSAGE_TYPE_VOICE`       | `voice`        | `file_id`, `mime_type`, `file_size`, `duration_sec`           |
| `MESSAGE_TYPE_AUDIO`       | `audio`        | `file_id`, `mime_type`, `file_size`, `file_name`              |
| `MESSAGE_TYPE_FILE`        | `file`         | `file_id`, `mime_type`, `file_size`, `file_name`              |
| `MESSAGE_TYPE_STICKER`     | `sticker`      | `file_id`, `mime_type`                                        |

**Response:** `201 Created`
```json
{
  "message_id": "...",
  "room_id": "...",
  "sender_id": "...",
  "type": "MESSAGE_TYPE_TEXT",
  "is_deleted": false,
  "created_at": "2026-03-15T10:30:00Z",
  "updated_at": "2026-03-15T10:30:00Z",
  "text": {
    "text": "Hello, world!"
  }
}
```

**Side effects:** Publishes `message.created` NATS event (delivered via WebSocket to room subscribers).

**Errors:**
- `403 PERMISSION_DENIED` -- sender is not a room member
- `400 INVALID_ARGUMENT` -- validation failure
- `404 NOT_FOUND` -- room or reply_to message does not exist

---

### GET /rooms/{id}/messages

Get message history with cursor-based pagination. Returns messages in reverse chronological order.

**Auth:** JWT

**Path parameters:**
- `id` -- Room UUID

**Query parameters:**
| Parameter | Type   | Default | Description                         |
|-----------|--------|---------|-------------------------------------|
| `cursor`  | string | --      | Opaque cursor from previous response|
| `limit`   | int    | 50      | Messages per page (1-100)           |

**Response:** `200 OK`
```json
{
  "messages": [
    {
      "message_id": "...",
      "room_id": "...",
      "sender_id": "...",
      "type": "MESSAGE_TYPE_TEXT",
      "is_deleted": false,
      "edited_at": null,
      "created_at": "2026-03-15T10:30:00Z",
      "updated_at": "2026-03-15T10:30:00Z",
      "reply_to": {
        "message_id": "...",
        "sender_id": "...",
        "type": "MESSAGE_TYPE_TEXT",
        "text_preview": "Original message text..."
      },
      "text": {
        "text": "Hello!"
      }
    }
  ],
  "next_cursor": "eyJ0IjoiMjAyNi0wMy0xNVQxMDozMDowMFoiLCJpIjoiLi4uIn0="
}
```

**Notes:**
- `next_cursor` is `null` when there are no more messages
- `reply_to` contains a preview of the replied-to message (if any)
- Deleted messages have `is_deleted: true` and content stripped

**Errors:**
- `403 PERMISSION_DENIED` -- requester is not a room member

---

## Health

### GET /health

Liveness probe. Always returns `200 OK` when the process is running.

**Auth:** None

### GET /ready

Readiness probe. Returns `200 OK` when the service is ready to accept traffic, `503` otherwise.

**Auth:** None

---

## Internal Endpoints

These endpoints are used for service-to-service communication and are not exposed to clients.

### GET /auth/validate

Validate a WebSocket ticket. Used by the WebSocket service via nginx `auth_request` forward auth.

**Headers:**
- `X-Ws-Ticket` -- the ticket UUID to validate

**Response:** `200 OK`
```json
{
  "user_id": "..."
}
```

**Response headers:**
- `X-User-Id` -- the authenticated user's UUID

**Errors:**
- `401 UNAUTHENTICATED` -- ticket missing, invalid, or expired
