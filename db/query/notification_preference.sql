-- name: ListNotificationPreferencesByUser :many
SELECT *
FROM notification_preferences
WHERE user_id = $1;

-- name: UpsertNotificationPreference :one
INSERT INTO notification_preferences (
    user_id,
    type,
    enabled
) VALUES ($1, $2, $3)
ON CONFLICT (user_id, type) DO UPDATE
SET enabled = $3,
    updated_at = now()
RETURNING *;
