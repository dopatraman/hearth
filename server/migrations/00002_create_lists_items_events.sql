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
    list_id   TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    user_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (list_id, user_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX list_memberships_user ON list_memberships (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE items (
    id                TEXT PRIMARY KEY,
    list_id           TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    creator_id        TEXT NOT NULL REFERENCES users(id),
    title             TEXT NOT NULL,
    notes             TEXT,
    state             TEXT NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'completed')),
    claimed_by        TEXT REFERENCES users(id),
    claimed_at        TEXT,
    assigned_by       TEXT REFERENCES users(id),
    will_complete_by  TEXT,
    completed_by      TEXT REFERENCES users(id),
    completed_at      TEXT,
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER items_updated_at
AFTER UPDATE ON items
FOR EACH ROW
BEGIN
    UPDATE items SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = OLD.id;
END;
-- +goose StatementEnd

-- Hot-path index for the backlog query (open items in a given list).
-- +goose StatementBegin
CREATE INDEX items_list_open ON items (list_id) WHERE state = 'open';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE events (
    id             TEXT PRIMARY KEY,
    list_id        TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    item_id        TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    actor_id       TEXT NOT NULL REFERENCES users(id),
    target_user_id TEXT REFERENCES users(id),
    type           TEXT NOT NULL CHECK (type IN (
        'created', 'claimed', 'unclaimed', 'assigned',
        'will_complete_by_set', 'snoozed', 'completed', 'edited'
    )),
    payload        TEXT,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_list_time ON events (list_id, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_item_time ON events (item_id, created_at);
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
DROP INDEX IF EXISTS events_item_time;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS events_list_time;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS events;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS items_list_open;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TRIGGER IF EXISTS items_updated_at;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS items;
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
