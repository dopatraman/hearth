-- name: CreateItem :one
INSERT INTO items (id, list_id, creator_id, title, notes)
VALUES (?, ?, ?, ?, ?)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, assigned_by, will_complete_by,
          completed_by, completed_at, created_at, updated_at;

-- name: BacklogForList :many
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, assigned_by, will_complete_by,
       completed_by, completed_at, created_at, updated_at
FROM items
WHERE list_id = ? AND state = 'open'
ORDER BY created_at DESC;

-- name: GetItemByID :one
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, assigned_by, will_complete_by,
       completed_by, completed_at, created_at, updated_at
FROM items
WHERE id = ?;
