-- name: CreateChore :one
INSERT INTO chores (id, list_id, creator_id, title, notes)
VALUES (?, ?, ?, ?, ?)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at;

-- name: BacklogForList :many
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, will_complete_by,
       completed_by, completed_at, created_at, updated_at
FROM chores
WHERE list_id = ? AND state = 'open'
ORDER BY created_at DESC;

-- name: GetChoreByID :one
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, will_complete_by,
       completed_by, completed_at, created_at, updated_at
FROM chores
WHERE id = ?;
