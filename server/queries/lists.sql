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
