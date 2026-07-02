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

-- name: ExistsPendingIrrigationCommand :one
SELECT EXISTS (
    SELECT 1
    FROM irrigation_commands
    WHERE device_id = $1
      AND status = 'PENDING'
);

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
WHERE uuid = $1 AND status = 'PENDING'
RETURNING *;

-- name: FailTimedOutCommands :exec
UPDATE irrigation_commands
SET
    status = 'TIMEOUT',
    processed_at = now(),
    error_message = 'Device timeout'
WHERE
    status = 'PENDING'
    AND requested_at <= now() - interval '10 seconds';

-- name: DeleteIrrigationCommand :exec
DELETE
FROM irrigation_commands
WHERE uuid = $1;

