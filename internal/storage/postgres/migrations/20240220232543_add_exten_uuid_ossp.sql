-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION "uuid-ossp";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP EXTENSION IF EXISTS "uuid-ossp";
-- +goose StatementEnd
