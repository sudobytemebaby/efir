# Message Service Database

## Schema

### messages

Stores all messages within chat rooms.

| Column        | Type         | Constraints                                |
| ------------- | ------------ | ------------------------------------------ |
| `id`          | UUID         | PRIMARY KEY, DEFAULT gen_random_uuid()     |
| `room_id`     | UUID         | NOT NULL                                   |
| `sender_id`   | UUID         | NOT NULL                                   |
| `type`        | message_type | NOT NULL                                   |
| `content`     | JSONB        | NOT NULL                                   |
| `reply_to_id` | UUID         | REFERENCES messages(id) ON DELETE SET NULL |
| `deleted_at`  | TIMESTAMPTZ  | NULL                                       |
| `edited_at`   | TIMESTAMPTZ  | NULL                                       |
| `created_at`  | TIMESTAMPTZ  | NOT NULL, DEFAULT now()                    |
| `updated_at`  | TIMESTAMPTZ  | NOT NULL, DEFAULT now()                    |

### message_type Enum

```sql
CREATE TYPE message_type AS ENUM (
    'text', 'image', 'video', 'video_note', 'voice',
    'audio', 'file', 'sticker', 'video_sticker', 'event'
);
```

### Content Storage

The `content` column is stored as JSONB with different structures based on message type:

**TEXT:**

```json
{ "text": "Hello, world!" }
```

**IMAGE, VIDEO:**

```json
{
  "file_id": "file-uuid",
  "mime_type": "image/jpeg",
  "file_size": 102400,
  "width": 1920,
  "height": 1080,
  "thumbnail_id": "thumb-uuid",
  "duration_sec": 30
}
```

**FILE, AUDIO:**

```json
{
  "file_id": "file-uuid",
  "mime_type": "application/pdf",
  "file_size": 512000,
  "file_name": "document.pdf",
  "duration_sec": 120
}
```

**VOICE:**

```json
{
  "file_id": "file-uuid",
  "mime_type": "audio/ogg",
  "file_size": 25600,
  "duration_sec": 15,
  "waveform": [0.1, 0.5, 0.3, 0.8]
}
```

**VIDEO_NOTE:**

```json
{
  "file_id": "file-uuid",
  "mime_type": "video/mp4",
  "file_size": 204800,
  "duration_sec": 60,
  "width": 512,
  "height": 512,
  "thumbnail_id": "thumb-uuid"
}
```

**STICKER:**

```json
{
  "file_id": "file-uuid",
  "mime_type": "image/webp",
  "emoji": "😀",
  "set_name": "StickerSet1"
}
```

**EVENT:**

```json
{ "text": "User joined the room" }
```

## Indexes

| Index                       | Columns                      | Condition                  |
| --------------------------- | ---------------------------- | -------------------------- |
| `idx_messages_room_created` | `(room_id, created_at DESC)` | -                          |
| `idx_messages_sender`       | `(sender_id)`                | -                          |
| `idx_messages_room_active`  | `(room_id, created_at DESC)` | `WHERE deleted_at IS NULL` |

## Migrations

Migrations are managed with [goose](https://github.com/pressly/goose).

```bash
# Run migrations
task message:migrate

# Create new migration
task message:migrate:create NAME=add_field
```

### Migration Files

| File                               | Description            |
| ---------------------------------- | ---------------------- |
| `000001_create_messages_table.sql` | Initial messages table |
