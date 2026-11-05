-- name: CreateNotification :one
INSERT INTO notifications (
    user_id,
    type,
    source,
    title,
    body
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListNotificationsByUser :many
SELECT *
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: MarkNotificationRead :one
UPDATE notifications
SET read_at = now()
WHERE uuid = $1 AND user_id = $2 AND read_at IS NULL
RETURNING *;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET read_at = now()
WHERE user_id = $1 AND read_at IS NULL;
