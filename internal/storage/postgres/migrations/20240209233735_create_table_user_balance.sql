-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    user_balance (
        user_id UUID PRIMARY KEY,
        current NUMERIC(15, 3) DEFAULT 0,
        withdrawn NUMERIC(15, 3) DEFAULT 0,
        CONSTRAINT fk_users FOREIGN KEY (user_id) REFERENCES users (user_id),
        CONSTRAINT fk_current CHECK (current >= 0 ),
        CONSTRAINT fk_withdrawn CHECK (withdrawn >= 0 )
    );

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_balance;

-- +goose StatementEnd