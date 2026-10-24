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

-- name: BacklogForUser :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
WHERE m.user_id = ? AND c.state = 'open'
ORDER BY c.created_at DESC;

-- name: GetChoreByID :one
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, will_complete_by,
       completed_by, completed_at, created_at, updated_at
FROM chores
WHERE id = ?;
