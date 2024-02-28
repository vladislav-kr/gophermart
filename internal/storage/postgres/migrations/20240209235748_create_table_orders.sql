-- +goose Up
-- +goose StatementBegin
CREATE TABLE orders (
    order_id TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    status VARCHAR(15) NOT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    accrual NUMERIC(15, 3) DEFAULT 0,
    CONSTRAINT fk_users FOREIGN KEY (user_id) REFERENCES users (user_id),
    CONSTRAINT fk_accrual CHECK (accrual >= 0 ),
    UNIQUE (order_id, user_id)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS orders;

-- +goose StatementEnd