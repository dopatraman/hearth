-- +goose Up

-- +goose StatementBegin
ALTER TABLE chores ADD COLUMN is_recurring INTEGER NOT NULL DEFAULT 0
    CHECK (is_recurring IN (0, 1));
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE chores ADD COLUMN icon TEXT;
-- +goose StatementEnd

-- Hot path for the Recurring rail on the Backlog screen: unclaimed recurring
-- chores on a given list. A claimed recurring chore is served out of
-- chores_list_section_order alongside one-off claimed rows.
-- +goose StatementBegin
CREATE INDEX chores_list_recurring_open
    ON chores (list_id)
    WHERE is_recurring = 1 AND state = 'open' AND claimed_at IS NULL;
-- +goose StatementEnd

-- events.type gains 'replenished', emitted when a recurring chore returns to
-- the rail after completion. The existing 'completed' event keeps firing so
-- audit continuity isn't broken. SQLite CHECK constraints can't be altered
-- in place — rebuild the table following the documented 12-step pattern.
-- +goose StatementBegin
CREATE TABLE events_new (
    id         TEXT PRIMARY KEY,
    list_id    TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    chore_id   TEXT NOT NULL REFERENCES chores(id) ON DELETE CASCADE,
    actor_id   TEXT NOT NULL REFERENCES users(id),
    type       TEXT NOT NULL CHECK (type IN (
        'created', 'claimed', 'unclaimed',
        'will_complete_by_set', 'snoozed', 'completed', 'replenished', 'edited'
    )),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO events_new (id, list_id, chore_id, actor_id, type, created_at)
SELECT id, list_id, chore_id, actor_id, type, created_at FROM events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events_new RENAME TO events;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_list_time ON events (list_id, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_chore_time ON events (chore_id, created_at);
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
CREATE TABLE events_new (
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
INSERT INTO events_new (id, list_id, chore_id, actor_id, type, created_at)
SELECT id, list_id, chore_id, actor_id, type, created_at
FROM events
WHERE type <> 'replenished';
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events_new RENAME TO events;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_list_time ON events (list_id, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX events_chore_time ON events (chore_id, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS chores_list_recurring_open;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE chores DROP COLUMN icon;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE chores DROP COLUMN is_recurring;
-- +goose StatementEnd
