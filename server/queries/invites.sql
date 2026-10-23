-- name: CreateInvite :one
INSERT INTO invites (code, list_id, created_by, expires_at)
VALUES (?, ?, ?, ?)
RETURNING code, list_id, created_by, created_at, expires_at;

-- name: GetInviteByCode :one
SELECT code, list_id, created_by, created_at, expires_at
FROM invites
WHERE code = ?;

-- name: DeleteExpiredInvites :exec
DELETE FROM invites
WHERE expires_at <= strftime('%Y-%m-%dT%H:%M:%fZ', 'now');
