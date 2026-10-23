-- name: InsertEvent :exec
INSERT INTO events (id, list_id, chore_id, actor_id, type)
VALUES (?, ?, ?, ?, ?);

-- name: EventsForChore :many
SELECT id, list_id, chore_id, actor_id, type, created_at
FROM events
WHERE chore_id = ?
ORDER BY created_at ASC;
