-- name: CreateIrrigationCommand :one
INSERT INTO irrigation_commands (
    device_id,
    user_id,
    action,
    duration_seconds
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetIrrigationCommand :one
SELECT *
FROM irrigation_commands
WHERE uuid = $1
LIMIT 1;

-- name: ListIrrigationCommands :many
SELECT *
FROM irrigation_commands
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: UpdateIrrigationCommandStatus :one
UPDATE irrigation_commands
SET
    status = $2,
    error_message = $3,
    processed_at = now()
WHERE uuid = $1
RETURNING *;

-- name: MarkTimedOutCommands :exec
UPDATE irrigation_commands
SET
    status = 'TIMEOUT',
    processed_at = now()
WHERE
    status = 'PENDING'
    AND requested_at < now() - interval '30 seconds';

-- name: DeleteIrrigationCommand :exec
DELETE
FROM irrigation_commands
WHERE uuid = $1;