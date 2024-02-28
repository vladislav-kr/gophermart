-- +goose Up
-- +goose StatementBegin
CREATE TABLE withdrawals (
    user_id UUID,
    order_id TEXT,
    sum NUMERIC(15, 3) DEFAULT 0,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_users FOREIGN KEY (user_id) REFERENCES users (user_id),
    CONSTRAINT fk_sum CHECK (sum >= 0),
    PRIMARY KEY (user_id, order_id)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS withdrawals;

-- +goose StatementEnd