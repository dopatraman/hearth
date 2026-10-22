-- name: InsertEvent :exec
INSERT INTO events (id, list_id, item_id, actor_id, target_user_id, type, payload)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: EventsForItem :many
SELECT id, list_id, item_id, actor_id, target_user_id, type, payload, created_at
FROM events
WHERE item_id = ?
ORDER BY created_at ASC;
