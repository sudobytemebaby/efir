# Room Service Database

## Schema

### rooms

Stores chat rooms.

| Column       | Type         | Constraints                  |
| ------------ | ------------ | ---------------------------- |
| `id`         | UUID         | PRIMARY KEY                  |
| `name`       | VARCHAR(100) | NOT NULL                     |
| `type`       | room_type    | NOT NULL ('direct', 'group') |
| `created_by` | UUID         | NOT NULL                     |
| `created_at` | TIMESTAMPTZ  | NOT NULL, DEFAULT now()      |
| `updated_at` | TIMESTAMPTZ  | NOT NULL, DEFAULT now()      |

### room_members

Stores room membership information.

| Column      | Type        | Constraints                                         |
| ----------- | ----------- | --------------------------------------------------- |
| `room_id`   | UUID        | PRIMARY KEY, REFERENCES rooms(id) ON DELETE CASCADE |
| `user_id`   | UUID        | PRIMARY KEY                                         |
| `role`      | member_role | NOT NULL DEFAULT 'member' ('owner', 'member')       |
| `joined_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now()                             |

### Indexes

| Index                      | Column    | Type   |
| -------------------------- | --------- | ------ |
| `idx_room_members_user_id` | `user_id` | B-tree |

## Enums

### room_type

- `direct` - 1-on-1 conversations
- `group` - Group chats

### member_role

- `owner` - Room creator with admin privileges
- `member` - Regular member

## Migrations

Migrations are managed with [goose](https://github.com/pressly/goose).

```bash
# Run migrations
task room:migrate

# Create new migration
task room:migrate:create NAME=add_field
```

### Migration Files

| File                      | Description                           |
| ------------------------- | ------------------------------------- |
| `20260317152337_init.sql` | Initial rooms and room_members tables |
