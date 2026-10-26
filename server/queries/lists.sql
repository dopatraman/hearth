-- name: CreateList :one
INSERT INTO lists (id, name)
VALUES (?, ?)
RETURNING id, name, created_at, updated_at;

-- name: AddMembership :exec
INSERT INTO list_memberships (list_id, user_id)
VALUES (?, ?)
ON CONFLICT (list_id, user_id) DO NOTHING;

-- name: ListMembershipsForUser :many
SELECT list_id, user_id
FROM list_memberships
WHERE user_id = ?;

-- MembersForUser returns the distinct users who share any list with the given
-- user, including the user themselves. Scoped by list_memberships; sorted by
-- display_name so clients get a stable order without a client-side sort.
-- name: MembersForUser :many
SELECT DISTINCT u.id, u.display_name
FROM users u
JOIN list_memberships m ON m.user_id = u.id
WHERE m.list_id IN (
    SELECT list_id FROM list_memberships WHERE list_memberships.user_id = ?
)
ORDER BY u.display_name, u.id;
