# User Service Database

## Schema

### users

Stores user profile information.

| Column         | Type         | Constraints                           |
| -------------- | ------------ | ------------------------------------- |
| `id`           | UUID         | PRIMARY KEY (references auth service) |
| `username`     | VARCHAR(50)  | NOT NULL, UNIQUE                      |
| `display_name` | VARCHAR(100) | NOT NULL                              |
| `avatar_url`   | TEXT         | NULLABLE                              |
| `bio`          | TEXT         | NULLABLE                              |
| `created_at`   | TIMESTAMPTZ  | NOT NULL, DEFAULT now()               |
| `updated_at`   | TIMESTAMPTZ  | NOT NULL, DEFAULT now()               |

### Indexes

| Index                    | Column         | Type   |
| ------------------------ | -------------- | ------ |
| `idx_users_username`     | `username`     | B-tree |
| `idx_users_display_name` | `display_name` | B-tree |

## Migrations

Migrations are managed with [goose](https://github.com/pressly/goose).

```bash
# Run migrations
task user:migrate

# Create new migration
task user:migrate:create NAME=add_field
```

### Migration Files

| File                      | Description                      |
| ------------------------- | -------------------------------- |
| `20260317142256_init.sql` | Initial users table with indexes |
