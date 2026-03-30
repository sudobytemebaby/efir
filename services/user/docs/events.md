# User Service Events

## NATS Streams

| Stream | Subjects | Purpose                    |
| ------ | -------- | -------------------------- |
| `AUTH` | `auth.>` | User authentication events |

## Subscribed Events

### auth.user.registered

Triggered when a new user successfully registers via Auth Service.

**Subject:** `auth.user.registered`

**Payload:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com"
}
```

**Consumer:** `user-svc-auth-registered`

**Processing:**

- Creates user profile with:
  - `id`: from `user_id`
  - `username`: derived from email (before @)
  - `display_name`: derived from email

**Retry Policy:**

- Max deliver: configurable (default 5)
- Ack wait: configurable (default 30s)
