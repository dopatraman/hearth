-- name: UpsertUserByAppleSub :one
INSERT INTO users (id, apple_sub, display_name)
VALUES (?, ?, ?)
ON CONFLICT (apple_sub) DO UPDATE SET
    display_name = COALESCE(excluded.display_name, users.display_name)
RETURNING id, apple_sub, display_name, notifications_enabled, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, apple_sub, display_name, notifications_enabled, created_at, updated_at
FROM users
WHERE id = ?;

-- name: GetUserByAppleSub :one
SELECT id, apple_sub, display_name, notifications_enabled, created_at, updated_at
FROM users
WHERE apple_sub = ?;
