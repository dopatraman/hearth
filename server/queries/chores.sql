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

-- name: ClaimChore :one
UPDATE chores
SET claimed_by = ?,
    claimed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ?
  AND state = 'open'
  AND claimed_by IS NULL
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at;

-- name: SetWillCompleteBy :one
UPDATE chores
SET will_complete_by = sqlc.narg(will_complete_by)
WHERE id = sqlc.arg(id)
  AND state = 'open'
  AND claimed_by = sqlc.arg(claimed_by)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at;

-- name: CompleteChore :one
UPDATE chores
SET state        = 'completed',
    completed_by = ?,
    completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ?
  AND state = 'open'
  AND claimed_by = ?
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at;

-- name: GraveyardForUser :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
WHERE m.user_id = ? AND c.state = 'completed'
ORDER BY c.completed_at DESC;

-- name: SearchGraveyard :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
WHERE m.user_id = sqlc.arg(user_id)
  AND c.state = 'completed'
  AND lower(c.title) LIKE '%' || lower(sqlc.arg(query)) || '%'
ORDER BY c.completed_at DESC;
