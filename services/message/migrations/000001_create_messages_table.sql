-- +goose Up
-- +goose StatementBegin

CREATE TYPE message_type AS ENUM (
    'text',
    'image',
    'video',
    'video_note',
    'voice',
    'audio',
    'file',
    'sticker',
    'video_sticker',
    'event'
);

CREATE TABLE messages (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id     uuid         NOT NULL,
    sender_id   uuid         NOT NULL,
    type        message_type NOT NULL,
    content     jsonb        NOT NULL,
    reply_to_id uuid         REFERENCES messages(id) ON DELETE SET NULL,
    deleted_at  timestamptz,
    edited_at   timestamptz,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_room_created
    ON messages (room_id, created_at DESC);

CREATE INDEX idx_messages_sender
    ON messages (sender_id);

CREATE INDEX idx_messages_room_active
    ON messages (room_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS messages;
DROP TYPE IF EXISTS message_type;

-- +goose StatementEnd
