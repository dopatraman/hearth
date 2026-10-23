-- +goose Up
-- +goose StatementBegin
CREATE TABLE invites (
    code       TEXT PRIMARY KEY,
    list_id    TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX invites_expires ON invites (expires_at);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS invites_expires;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS invites;
-- +goose StatementEnd
