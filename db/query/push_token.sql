-- name: UpsertPushToken :one
INSERT INTO push_tokens (
    user_id,
    token,
    platform
) VALUES ($1, $2, $3)
ON CONFLICT (token) DO UPDATE
SET user_id = $1,
    platform = $3,
    updated_at = now()
RETURNING *;

-- name: ListPushTokensByUser :many
SELECT *
FROM push_tokens
WHERE user_id = $1;

-- name: DeletePushToken :exec
DELETE FROM push_tokens
WHERE token = $1;
