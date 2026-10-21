-- name: ServerNow :one
SELECT now()::timestamptz AS server_time;
