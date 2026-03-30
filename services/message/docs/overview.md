# Message Service Overview

## Responsibilities

The Message Service is responsible for:

- Sending messages to chat rooms
- Retrieving messages with cursor-based pagination
- Fetching individual messages by ID
- Soft-deleting messages (sender only)
- Publishing message created events to NATS

## Architecture

### Clean Architecture Layers

```
handler/     -> gRPC request handling, proto validation
service/     -> Business logic, message processing
repository/  -> PostgreSQL message storage
client/      -> Room service gRPC client
nats/        -> Event publishing to NATS
```

### Request Flow

```
Client -> Gateway (HTTP) -> Message Service (gRPC)
                              |
                              +-> Repository (PostgreSQL)
                              +-> RoomClient (gRPC to Room Service)
                              +-> NATS (message.created event)
```

### Message Types

The service supports the following message content types:

| Type         | Content                                                                                  | Description          |
| ------------ | ---------------------------------------------------------------------------------------- | -------------------- |
| `TEXT`       | `text`                                                                                   | Plain text message   |
| `IMAGE`      | `file_id`, `mime_type`, `file_size`, `width`, `height`, `thumbnail_id?`, `duration_sec?` | Image attachment     |
| `VIDEO`      | `file_id`, `mime_type`, `file_size`, `width`, `height`, `thumbnail_id?`, `duration_sec?` | Video attachment     |
| `VIDEO_NOTE` | `file_id`, `mime_type`, `file_size`, `width`, `height`, `thumbnail_id?`, `duration_sec?` | Square video message |
| `VOICE`      | `file_id`, `mime_type`, `file_size`, `duration_sec`, `waveform?`                         | Voice message        |
| `AUDIO`      | `file_id`, `mime_type`, `file_size`, `file_name`, `duration_sec?`                        | Audio file           |
| `FILE`       | `file_id`, `mime_type`, `file_size`, `file_name`, `duration_sec?`                        | Generic file         |
| `STICKER`    | `file_id`, `mime_type`, `emoji?`, `set_name?`                                            | Sticker              |
| `EVENT`      | `text`                                                                                   | System event message |

### Reply Messages

Messages can reply to a previous message via `reply_to_id`. The reply target must:

- Exist
- Not be deleted
- Belong to the same room

### Soft Delete

Messages are soft-deleted by setting a `deleted_at` timestamp. Deleted messages are excluded from queries but the data is preserved.

### Error Handling

All errors map to gRPC status codes:

| Code                | Condition                                 |
| ------------------- | ----------------------------------------- |
| `INVALID_ARGUMENT`  | Validation failed, invalid reply target   |
| `NOT_FOUND`         | Message not found                         |
| `PERMISSION_DENIED` | Not a room member, not the message sender |
| `INTERNAL`          | Database or remote service errors         |

### Health Checks

- `/health` - Liveness probe (always returns ok)
- `/ready` - Readiness probe (checks PostgreSQL)

## Dependencies

| Dependency   | Purpose                 |
| ------------ | ----------------------- |
| PostgreSQL   | Message storage         |
| NATS         | Event publishing        |
| Room Service | Membership verification |
