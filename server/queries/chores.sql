-- name: CreateChore :one
INSERT INTO chores (id, list_id, creator_id, title, notes)
VALUES (?, ?, ?, ?, ?)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at, claimed_order;

-- name: BacklogForList :many
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, will_complete_by,
       completed_by, completed_at, created_at, updated_at, claimed_order
FROM chores
WHERE list_id = ? AND state = 'open'
ORDER BY created_at DESC;

-- name: BacklogForUser :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at, c.claimed_order,
       s.snoozed_until
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
LEFT JOIN snoozes s
       ON s.user_id = sqlc.arg(user_id)
      AND s.chore_id = c.id
      AND s.snoozed_until > strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE m.user_id = sqlc.arg(user_id)
  AND c.state = 'open'
ORDER BY c.created_at DESC;

-- name: GetChoreByID :one
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, will_complete_by,
       completed_by, completed_at, created_at, updated_at, claimed_order
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
          completed_by, completed_at, created_at, updated_at, claimed_order;

-- name: AssignNextClaimedOrder :one
UPDATE chores
SET claimed_order = (
    SELECT COALESCE(MAX(co.claimed_order), 0) + 1
      FROM chores co
     WHERE co.list_id = chores.list_id
       AND co.claimed_order IS NOT NULL
)
WHERE chores.id = ?
RETURNING chores.id, chores.list_id, chores.creator_id, chores.title, chores.notes, chores.state,
          chores.claimed_by, chores.claimed_at, chores.will_complete_by,
          chores.completed_by, chores.completed_at, chores.created_at, chores.updated_at, chores.claimed_order;

-- name: ClearClaimedOrder :exec
UPDATE chores
SET claimed_order = NULL
WHERE id = ?;

-- name: SetClaimedOrder :exec
UPDATE chores
SET claimed_order = sqlc.arg(claimed_order)
WHERE id = sqlc.arg(id);

-- name: ClaimedChoresForList :many
SELECT id, claimed_order, created_at
FROM chores
WHERE list_id = sqlc.arg(list_id)
  AND state = 'open'
  AND claimed_at IS NOT NULL
ORDER BY claimed_order IS NULL, claimed_order ASC, created_at ASC;

-- name: ReleaseClaim :one
UPDATE chores
SET claimed_by       = NULL,
    claimed_at       = NULL,
    will_complete_by = NULL
WHERE id = sqlc.arg(id)
  AND state = 'open'
  AND claimed_by = sqlc.arg(claimed_by)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at, claimed_order;

-- name: SetWillCompleteBy :one
UPDATE chores
SET will_complete_by = sqlc.narg(will_complete_by)
WHERE id = sqlc.arg(id)
  AND state = 'open'
  AND claimed_by = sqlc.arg(claimed_by)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at, claimed_order;

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
          completed_by, completed_at, created_at, updated_at, claimed_order;

-- name: GraveyardForUser :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at, c.claimed_order
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
WHERE m.user_id = ? AND c.state = 'completed'
ORDER BY c.completed_at DESC;

-- name: SearchGraveyard :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at, c.claimed_order
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
WHERE m.user_id = sqlc.arg(user_id)
  AND c.state = 'completed'
  AND lower(c.title) LIKE '%' || lower(sqlc.arg(query)) || '%'
ORDER BY c.completed_at DESC;
