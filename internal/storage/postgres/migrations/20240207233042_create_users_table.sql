-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS
    users (
        user_id UUID PRIMARY KEY,
        login TEXT NOT NULL,
        pass_hash BYTEA NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        is_delete BOOL DEFAULT FALSE,
        is_blocked BOOL DEFAULT FALSE,
        is_admin BOOL DEFAULT FALSE
    );
CREATE UNIQUE INDEX IF NOT EXISTS login_idx ON users (login);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
DROP INDEX login_idx;
-- +goose StatementEnd
