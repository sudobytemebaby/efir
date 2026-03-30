# Auth Service API

gRPC service definition in `proto/efir/auth/auth.proto`

## AuthService

### Register

Register a new user account.

**Request:**

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Validation:**

- `email`: Valid email format, max 255 characters
- `password`: 8-72 characters

**Response:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
}
```

**Errors:**

- `ALREADY_EXISTS` - Email already registered
- `INVALID_ARGUMENT` - Validation failed
- `RESOURCE_EXHAUSTED` - Rate limit exceeded

---

### Login

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

- `NOT_FOUND` - User not found
- `PERMISSION_DENIED` - Invalid password
- `RESOURCE_EXHAUSTED` - Rate limit exceeded

---

### Logout

Invalidate a refresh token.

**Request:**

```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
}
```

**Response:** Empty (HTTP 204)

**Errors:**

- `INVALID_ARGUMENT` - Invalid token format
- `NOT_FOUND` - Token not found

---

### RefreshToken

Exchange refresh token for new access/refresh tokens.

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

**Errors:**

- `NOT_FOUND` - Refresh token not found or expired
- `UNAUTHENTICATED` - Token already used/invalid
