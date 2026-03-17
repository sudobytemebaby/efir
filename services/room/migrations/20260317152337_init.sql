-- +goose Up
-- +goose StatementBegin
CREATE TYPE room_type AS ENUM ('direct', 'group');
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
CREATE TYPE member_role AS ENUM ('owner', 'member');
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    type room_type NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_members (
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id)
);
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_room_members_user_id ON room_members(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS room_members;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rooms;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS member_role;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS room_type;
-- +goose StatementEnd
