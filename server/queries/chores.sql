-- name: CreateChore :one
INSERT INTO chores (id, list_id, creator_id, title, notes)
VALUES (?, ?, ?, ?, ?)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at, section_order,
          is_recurring, icon;

-- name: CreateRecurringChore :one
INSERT INTO chores (id, list_id, creator_id, title, notes, is_recurring, icon)
VALUES (?, ?, ?, ?, ?, 1, sqlc.narg(icon))
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at, section_order,
          is_recurring, icon;

-- name: BacklogForList :many
SELECT id, list_id, creator_id, title, notes, state,
       claimed_by, claimed_at, will_complete_by,
       completed_by, completed_at, created_at, updated_at, section_order,
       is_recurring, icon
FROM chores
WHERE list_id = ? AND state = 'open'
ORDER BY created_at DESC;

-- name: BacklogForUser :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at, c.section_order,
       c.is_recurring, c.icon,
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
       completed_by, completed_at, created_at, updated_at, section_order,
       is_recurring, icon
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
          completed_by, completed_at, created_at, updated_at, section_order,
          is_recurring, icon;

-- AssignNextSectionOrder places the chore at the bottom of its CURRENT
-- section. MAX is scoped by both list_id and the section the target row is
-- already in (claimed_at IS NULL vs IS NOT NULL), so claim/release that
-- flip claimed_at before calling this land in the correct destination.
-- name: AssignNextSectionOrder :one
UPDATE chores
SET section_order = (
    SELECT COALESCE(MAX(co.section_order), 0) + 1
      FROM chores co
     WHERE co.list_id = chores.list_id
       AND co.state = 'open'
       AND co.section_order IS NOT NULL
       AND (co.claimed_at IS NULL) = (chores.claimed_at IS NULL)
)
WHERE chores.id = ?
RETURNING chores.id, chores.list_id, chores.creator_id, chores.title, chores.notes, chores.state,
          chores.claimed_by, chores.claimed_at, chores.will_complete_by,
          chores.completed_by, chores.completed_at, chores.created_at, chores.updated_at, chores.section_order,
          chores.is_recurring, chores.icon;

-- name: ClearSectionOrder :exec
UPDATE chores
SET section_order = NULL
WHERE id = ?;

-- name: SetSectionOrder :exec
UPDATE chores
SET section_order = sqlc.arg(section_order)
WHERE id = sqlc.arg(id);

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
          completed_by, completed_at, created_at, updated_at, section_order,
          is_recurring, icon;

-- name: SetWillCompleteBy :one
UPDATE chores
SET will_complete_by = sqlc.narg(will_complete_by)
WHERE id = sqlc.arg(id)
  AND state = 'open'
  AND claimed_by = sqlc.arg(claimed_by)
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at, section_order,
          is_recurring, icon;

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
          completed_by, completed_at, created_at, updated_at, section_order,
          is_recurring, icon;

-- ReplenishRecurring returns a recurring chore to the rail after completion:
-- clears claim state and section_order, leaves state='open'. Guarded so it
-- only fires for open recurring rows currently claimed by the caller.
-- name: ReplenishRecurring :one
UPDATE chores
SET claimed_by       = NULL,
    claimed_at       = NULL,
    will_complete_by = NULL,
    section_order    = NULL
WHERE id = ?
  AND state = 'open'
  AND is_recurring = 1
  AND claimed_by = ?
RETURNING id, list_id, creator_id, title, notes, state,
          claimed_by, claimed_at, will_complete_by,
          completed_by, completed_at, created_at, updated_at, section_order,
          is_recurring, icon;

-- name: GraveyardForUser :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at, c.section_order,
       c.is_recurring, c.icon
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
WHERE m.user_id = ? AND c.state = 'completed'
ORDER BY c.completed_at DESC;

-- name: SearchGraveyard :many
SELECT c.id, c.list_id, c.creator_id, c.title, c.notes, c.state,
       c.claimed_by, c.claimed_at, c.will_complete_by,
       c.completed_by, c.completed_at, c.created_at, c.updated_at, c.section_order,
       c.is_recurring, c.icon
FROM chores c
JOIN list_memberships m ON m.list_id = c.list_id
WHERE m.user_id = sqlc.arg(user_id)
  AND c.state = 'completed'
  AND lower(c.title) LIKE '%' || lower(sqlc.arg(query)) || '%'
ORDER BY c.completed_at DESC;
