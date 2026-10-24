-- name: UpsertDeviceToken :exec
INSERT INTO device_tokens (user_id, token, platform)
VALUES (?, ?, ?)
ON CONFLICT (user_id, token) DO NOTHING;

-- name: DeleteDeviceToken :exec
DELETE FROM device_tokens WHERE token = ?;

-- name: TokensForList :many
SELECT d.user_id, d.token, d.platform
FROM device_tokens d
JOIN list_memberships m ON m.user_id = d.user_id
JOIN users u           ON u.id = d.user_id
WHERE m.list_id = sqlc.arg(list_id)
  AND d.user_id != sqlc.arg(actor_id)
  AND u.notifications_enabled = 1;

-- name: SetNotificationsEnabled :exec
UPDATE users SET notifications_enabled = ? WHERE id = ?;
