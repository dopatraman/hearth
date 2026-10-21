-- name: ServerNow :one
SELECT strftime('%Y-%m-%dT%H:%M:%fZ', 'now') AS server_time;
