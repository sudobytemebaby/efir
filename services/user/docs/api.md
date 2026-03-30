# User Service API

gRPC service definition in `proto/efir/user/user.proto`

## UserService

### GetUser

Get a user profile by ID.

**Request:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:**

```json
{
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "display_name": "John Doe",
    "avatar_url": "https://example.com/avatar.jpg",
    "bio": "Software developer",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-20T14:22:00Z"
  }
}
```

**Errors:**

- `NOT_FOUND` - User not found
- `INVALID_ARGUMENT` - Invalid user_id format

---

### GetUsersByIds

Get multiple user profiles by IDs.

**Request:**

```json
{
  "user_ids": [
    "550e8400-e29b-41d4-a716-446655440000",
    "660e8400-e29b-41d4-a716-446655440001"
  ]
}
```

**Response:**

```json
{
  "users": [
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "johndoe",
      "display_name": "John Doe"
    },
    {
      "user_id": "660e8400-e29b-41d4-a716-446655440001",
      "username": "janedoe",
      "display_name": "Jane Doe"
    }
  ]
}
```

**Errors:**

- `INVALID_ARGUMENT` - Empty user_ids array

---

### UpdateUser

Update a user profile.

**Request:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "display_name": "John D.",
  "avatar_url": "https://example.com/new-avatar.jpg",
  "bio": "Senior developer"
}
```

**Note:** All fields are optional. Only provided fields will be updated.

**Response:**

```json
{
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "display_name": "John D.",
    "avatar_url": "https://example.com/new-avatar.jpg",
    "bio": "Senior developer",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-25T09:15:00Z"
  }
}
```

**Errors:**

- `NOT_FOUND` - User not found
- `INVALID_ARGUMENT` - Validation failed
