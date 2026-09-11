-- name: CreateNotificationLog :one
INSERT INTO notifications_log (order_id, type, channel)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetNotificationLogsByOrder :many
SELECT * FROM notifications_log
WHERE order_id = $1
ORDER BY sent_at ASC;
