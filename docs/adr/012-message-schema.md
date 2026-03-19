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
- `sticker` — belong to a set, not uploaded by users
- `video_sticker` — deferred, not implemented in MVP
- `event` — system events in timeline, created only by service

### Explicit Type Field in SendMessageRequest

`SendMessageRequest` includes an explicit `MessageType type` field. This field is required for `SendMediaContent` where the client specifies whether the media is `image` or `video` — both use the same proto message but need to be stored differently. For all other content types (text, file, voice, etc.), the type is determined by the `oneof content` arm and the `type` field must match.

### Type-Content Validation

Handler enforces strict type-content matching. Each `oneof` arm has a corresponding expected type:
- `SendTextContent` → `MESSAGE_TYPE_TEXT`
- `SendMediaContent` → `MESSAGE_TYPE_IMAGE` or `MESSAGE_TYPE_VIDEO`
- `SendFileContent` → `MESSAGE_TYPE_FILE`
- `SendVoiceContent` → `MESSAGE_TYPE_VOICE`
- `SendVideoNoteContent` → `MESSAGE_TYPE_VIDEO_NOTE`
- `SendStickerContent` → `MESSAGE_TYPE_STICKER`
- `SendAudioContent` → `MESSAGE_TYPE_AUDIO`

Proto validation rejects `MESSAGE_TYPE_UNSPECIFIED` via `not_in = 0` annotation. Handler returns `codes.InvalidArgument` for type mismatches.

### `reply_to_id` with `ON DELETE SET NULL`

Replies remain in history if original is deleted. `MessagePreview` is assembled via LEFT JOIN to `messages rm` on read — no separate table. `MessagePreview` is defined as a reusable proto type, to be used in the notification system.

**Replies to deleted messages are not allowed.** If `original.DeletedAt != nil`, `ErrInvalidReplyTarget` is returned.

**Reply preview** includes available preview data:
- Text messages: `text_preview` with message text
- File/audio messages: `file_name` and `mime_type`
- Media/voice/video_note/sticker messages: `mime_type`
- Event messages: `text_preview` with event text

### `edited_at` in Base Table

Records fact and time of last edit. Edit history is not stored.

### Membership Verification via Synchronous gRPC Call to Room Service

Room Service is the single source of truth for membership. Retry up to 3 attempts with exponential backoff (100ms, 300ms, 900ms) only for `codes.Unavailable` and `codes.DeadlineExceeded`.

### `message.created` Published to NATS Best-Effort

If NATS is unavailable — message is saved, event is lost, error is logged. WebSocket Service must re-fetch history from Message Service on reconnect.

### `SendMessageRequest` Does Not Accept Binary Content

Client uploads file to MinIO via presigned URL, receives `file_id`, then calls `SendMessage` with metadata.

### Error Handling

- Repository returns `ErrMessageNotFound` for deleted or non-existent messages
- Service distinguishes `ErrMessageNotFound` from infrastructure errors (network, DB failures)
- Handler maps `ErrMessageNotFound` → `codes.NotFound`, permission errors → `codes.PermissionDenied`, invalid arguments → `codes.InvalidArgument`, infrastructure errors → `codes.Internal`

## Known Limitations

1. **Runtime dependency on Room Service** — `IsMember` is called synchronously on every `SendMessage` and `GetMessages`. When Room Service is unavailable, messages are not accepted. Correct solution — local membership replica in `message_db` via `room.membership.changed` NATS event. Deferred to Module 2.

2. **Loss of `message.created` events when NATS is unavailable** — similar to ADR-011. Transactional outbox deferred to Module 2.

3. **No edit history** — `edited_at` records only last edit, original is lost.

4. **Eventual consistency window when kicked from room** — between removal from room and Message Service learning about it via Room Service, user may send a message.

5. **`GetRoomMembers` on every `SendMessage`** — in large rooms returns thousands of IDs. Acceptable for MVP. Long-term — WebSocket Service should know who is connected.

6. **Deletion only by sender** — room owner cannot delete other's message without additional gRPC call to Room Service.

7. **Cursor pagination by composite `(created_at, id)`** — `gen_random_uuid()` does not guarantee monotonicity. Consider UUIDv7 in the future.

8. **JSONB structure validation only at application level** — no database-level guarantees for fields inside content.

9. **Insecure gRPC for internal service-to-service communication** — Room Service client uses `insecure.NewCredentials()`. Acceptable within docker network. External TLS terminated by Traefik.

10. **Health endpoint reports ready before Room Service connectivity verified** — `SetReady(true)` called on startup. First real call happens on first request with retry.

## Consequences

- Adding new type: new enum value, new Go type, new `oneof` arm in proto, new case in marshal/unmarshal. No migration needed.
- History queries — one simple SELECT with one optional JOIN for reply preview.
- Message Service knows nothing about MinIO — works only with `file_id` as opaque identifier.
- `MessagePreview` available to all services via `services/shared/gen`.
- Strict type-content validation prevents silent data corruption from mismatched type/content pairs.
