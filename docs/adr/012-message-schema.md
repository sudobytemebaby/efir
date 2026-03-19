# ADR-012: Message Schema and Service Design

## Status

Accepted

## Context

We need to store message history supporting multiple content types: text, media, files, voice, video notes, stickers, and system events. Key questions: storage schema, membership verification, reply system, and subscriber notifications.

## Decisions

### JSONB for Content

Single `messages` table with `content jsonb NOT NULL` column. Content type is determined by `type message_type` column. Messages are always read as a whole — filtering by content fields via SQL is not required. Content search is handled by Typesense, not PostgreSQL. Content structure validation is at the proto and service layer. Adding new types requires no schema migration.

### Types Split by Behavior

- `image`/`video` — compressed and transcoded, always have thumbnail and dimensions
- `file`/`audio` — served as-is, always have file_name
- `voice` — recorded on-the-fly, has inline waveform (~200-2000 bytes, similar to Telegram TL-schema)
- `video_note` — video note (circle), recorded on-the-fly, has thumbnail
- `sticker`/`video_sticker` — belong to a set, not uploaded by users
- `event` — system events in timeline, created only by service

### `reply_to_id` with `ON DELETE SET NULL`

Replies remain in history if original is deleted. `MessagePreview` is assembled via LEFT JOIN to `messages rm` on read — no separate table. `MessagePreview` is defined as a reusable proto type, to be used in the notification system.

### `edited_at` in Base Table

Records fact and time of last edit. Edit history is not stored.

### Membership Verification via Synchronous gRPC Call to Room Service

Room Service is the single source of truth for membership. Retry up to 3 attempts with exponential backoff only for `codes.Unavailable` and `codes.DeadlineExceeded`.

### `message.created` Published to NATS Best-Effort

If NATS is unavailable — message is saved, event is lost, error is logged. WebSocket Service must re-fetch history from Message Service on reconnect.

### `SendMessageRequest` Does Not Accept Binary Content

Client uploads file to MinIO via presigned URL, receives `file_id`, then calls `SendMessage` with metadata.

## Known Limitations

1. **Runtime dependency on Room Service** — `IsMember` is called synchronously on every `SendMessage` and `GetMessages`. When Room Service is unavailable, messages are not accepted. Correct solution — local membership replica in `message_db` via `room.membership.changed` NATS event. Deferred to Module 2.

2. **Loss of `message.created` events when NATS is unavailable** — similar to ADR-011. Transactional outbox deferred to Module 2.

3. **No edit history** — `edited_at` records only last edit, original is lost.

4. **Eventual consistency window when kicked from room** — between removal from room and Message Service learning about it via Room Service, user may send a message.

5. **`GetRoomMembers` on every `SendMessage`** — in large rooms returns thousands of IDs. Acceptable for MVP. Long-term — WebSocket Service should know who is connected.

6. **Deletion only by sender** — room owner cannot delete other's message without additional gRPC call to Room Service.

7. **Cursor pagination by composite `(created_at, id)`** — `gen_random_uuid()` does not guarantee monotonicity. Consider UUIDv7 in the future.

8. **JSONB structure validation only at application level** — no database-level guarantees for fields inside content.

## Consequences

- Adding new type: new enum value, new Go type, new `oneof` arm in proto, new case in marshal/unmarshal. No migration needed.
- History queries — one simple SELECT with one optional JOIN for reply preview.
- Message Service knows nothing about MinIO — works only with `file_id` as opaque identifier.
- `MessagePreview` available to all services via `services/shared/gen`.
