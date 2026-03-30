# Auth Service Events

## NATS Streams

| Stream | Subjects | Purpose                    |
| ------ | -------- | -------------------------- |
| `AUTH` | `auth.>` | User authentication events |

## Published Events

### auth.user.registered

Published when a new user successfully registers.

**Subject:** `auth.user.registered`

**Payload:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com"
}
```

**Consumers:**

- User Service - Creates user profile

**Delivery:**

- Acknowledgment-based
- Max deliver: configurable (default from NATS config)
- Ack wait: configurable
