-- +goose Up

-- +goose StatementBegin
CREATE TABLE device_tokens (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL,
    platform   TEXT NOT NULL CHECK (platform IN ('ios', 'watchos')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (user_id, token)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX device_tokens_user ON device_tokens (user_id);
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
DROP INDEX IF EXISTS device_tokens_user;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS device_tokens;
-- +goose StatementEnd
