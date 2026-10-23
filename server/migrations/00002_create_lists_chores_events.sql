-- +goose Up

-- +goose StatementBegin
CREATE TABLE lists (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL DEFAULT 'Home',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE list_memberships (
    list_id TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (list_id, user_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX list_memberships_user ON list_memberships (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE chores (
    id                TEXT PRIMARY KEY,
    list_id           TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    creator_id        TEXT NOT NULL REFERENCES users(id),
    title             TEXT NOT NULL,
    notes             TEXT,
    state             TEXT NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'completed')),
    claimed_by        TEXT REFERENCES users(id),
    claimed_at        TEXT,
    will_complete_by  TEXT,
    completed_by      TEXT REFERENCES users(id),
    completed_at      TEXT,
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER chores_updated_at
AFTER UPDATE ON chores
FOR EACH ROW
BEGIN
    UPDATE chores SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = OLD.id;
END;
-- +goose StatementEnd

-- Hot-path index for the backlog query (open chores in a given list).
-- +goose StatementBegin
CREATE INDEX chores_list_open ON chores (list_id) WHERE state = 'open';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE events (
    id         TEXT PRIMARY KEY,
    list_id    TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    chore_id   TEXT NOT NULL REFERENCES chores(id) ON DELETE CASCADE,
    actor_id   TEXT NOT NULL REFERENCES users(id),
    type       TEXT NOT NULL CHECK (type IN (
        'created', 'claimed', 'unclaimed',
        'will_complete_by_set', 'snoozed', 'completed', 'edited'
    )),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_list_time ON events (list_id, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_chore_time ON events (chore_id, created_at);
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
DROP INDEX IF EXISTS events_chore_time;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS events_list_time;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS events;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS chores_list_open;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TRIGGER IF EXISTS chores_updated_at;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS chores;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS list_memberships_user;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS list_memberships;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS lists;
-- +goose StatementEnd
