-- +goose Up
-- +goose StatementBegin
-- idx_users_username is redundant: the UNIQUE constraint on username already creates an implicit index.
DROP INDEX IF EXISTS idx_users_username;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
-- +goose StatementEnd
