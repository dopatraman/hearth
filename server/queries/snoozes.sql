-- name: UpsertSnooze :exec
INSERT INTO snoozes (user_id, chore_id, snoozed_until)
VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+8 hours'))
ON CONFLICT (user_id, chore_id)
DO UPDATE SET snoozed_until = excluded.snoozed_until;

-- name: DeleteSnooze :exec
DELETE FROM snoozes WHERE user_id = ? AND chore_id = ?;
