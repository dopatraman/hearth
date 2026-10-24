-- +goose Up

-- +goose StatementBegin
CREATE TABLE snoozes (
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chore_id       TEXT NOT NULL REFERENCES chores(id) ON DELETE CASCADE,
    snoozed_until  TEXT NOT NULL,
    PRIMARY KEY (user_id, chore_id)
);
-- +goose StatementEnd

-- Backlog read-path index: filter rows where this user has a non-expired snooze.
-- +goose StatementBegin
CREATE INDEX snoozes_user_until ON snoozes (user_id, snoozed_until);
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
DROP INDEX IF EXISTS snoozes_user_until;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS snoozes;
-- +goose StatementEnd
