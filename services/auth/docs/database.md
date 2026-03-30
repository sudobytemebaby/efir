# Auth Service Database

## Schema

### accounts

Stores user account credentials.

| Column          | Type         | Constraints             |
| --------------- | ------------ | ----------------------- |
| `id`            | UUID         | PRIMARY KEY             |
| `email`         | VARCHAR(255) | NOT NULL, UNIQUE        |
| `password_hash` | VARCHAR(255) | NOT NULL                |
| `created_at`    | TIMESTAMPTZ  | NOT NULL, DEFAULT now() |
| `updated_at`    | TIMESTAMPTZ  | NOT NULL, DEFAULT now() |

## Migrations

Migrations are managed with [goose](https://github.com/pressly/goose).

```bash
# Run migrations
task auth:migrate

# Create new migration
task auth:migrate:create NAME=add_field
```

### Migration Files

| File                      | Description            |
| ------------------------- | ---------------------- |
| `20260313101721_init.sql` | Initial accounts table |

## Valkey Keys

| Key Pattern                       | Type   | TTL    | Purpose                |
| --------------------------------- | ------ | ------ | ---------------------- |
| `auth:refresh:{token}`            | STRING | 7 days | Refresh token storage  |
| `auth:ratelimit:{action}:{email}` | STRING | 1 min  | Rate limiting counters |

## Password Storage

Passwords are hashed using bcrypt with cost factor 10.
