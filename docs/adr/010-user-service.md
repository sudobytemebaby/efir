## ADR-010: User Service Implementation

## Status

Accepted

## Context

The User Service manages user profiles: username, display name, avatar URL, and bio. User profiles are created asynchronously via NATS events when a user registers via the Auth Service.

Key requirements:
- Idempotent user creation (NATS replay protection)
- Username generated from email prefix
- Separate database from Auth Service
- Event-driven creation via NATS

## Decision

### Database Schema

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    avatar_url TEXT,
    bio TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Event-Driven User Creation

When Auth Service registers a user, it publishes `auth.user.registered` to NATS:
```json
{
  "user_id": "uuid",
  "email": "user@example.com"
}
```

User Service subscribes to this event and creates the profile. The username is derived from the email prefix (e.g., `john@example.com` → `john`).

### Idempotent Creation

The `ON CONFLICT DO NOTHING` PostgreSQL feature is used for idempotency:

```sql
INSERT INTO users (id, username, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO NOTHING
RETURNING ...
```

If a conflict occurs (user already exists), PostgreSQL returns no rows. The repository returns a dedicated `ErrUserAlreadyExists` error, and the service treats this as a successful no-op by fetching the existing user.

This approach ensures:
1. NATS replay doesn't create duplicates
2. No error logs on legitimate replay scenarios
3. The consumer ACKs the message successfully

### NATS Consumer Configuration

- Stream: `AUTH`
- Subject: `auth.user.registered`
- Consumer: `user-svc-auth-registered`
- `MaxDeliver: 5` — After 5 failed attempts, message is discarded
- Uses `ProvisionConsumerWithRetry` to handle stream unavailability at startup

## Alternatives Considered

1. **Separate registration endpoint**: User Service could have its own registration endpoint, but this creates tight coupling between services and complicates the transaction flow.

2. **Optimistic locking**: Could use version numbers instead of ON CONFLICT DO NOTHING, but adds complexity without benefit for this use case.

3. **Return existing user on conflict**: Some implementations return the existing user from the INSERT, but PostgreSQL's behavior with DO NOTHING doesn't support RETURNING the existing row.

## Consequences

- User profile creation is eventually consistent (depends on NATS delivery)
- If NATS is unavailable during registration, the user is created in Auth but no profile exists — requires reconciliation job (out of scope for MVP)
- Username uniqueness is enforced at DB level — duplicate emails get same username prefix, but different UUIDs
