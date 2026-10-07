-- name: CreateControllerDeviceConfigurationCommand :one
INSERT INTO controller_device_configuration_commands (
    controller_device_id,
    action,
    timezone
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetControllerDeviceConfigurationCommand :one
SELECT *
FROM controller_device_configuration_commands
WHERE uuid = $1
LIMIT 1;

-- name: GetLatestControllerDeviceConfigurationCommand :one
SELECT *
FROM controller_device_configuration_commands
WHERE controller_device_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPendingControllerDeviceConfigurationCommands :many
SELECT c.*
FROM controller_device_configuration_commands c
WHERE c.status = 'PENDING'
  AND c.attempts < $1
  AND (c.next_attempt_at IS NULL OR c.next_attempt_at <= now())
ORDER BY c.created_at;

-- name: MarkControllerDeviceConfigurationAttempt :one
UPDATE controller_device_configuration_commands
SET
    attempts = attempts + 1,
    last_attempt_at = now(),
    next_attempt_at = $2,
    status = $3,
    error_message = $4,
    updated_at = now()
WHERE uuid = $1
RETURNING *;

-- name: UpdateControllerDeviceConfigurationStatus :one
UPDATE controller_device_configuration_commands
SET
    status = $2,
    error_message = $3,
    acknowledged_at = CASE WHEN $2 = 'ACKNOWLEDGED' THEN now() ELSE acknowledged_at END,
    updated_at = now()
WHERE uuid = $1
  AND status NOT IN ('ACKNOWLEDGED', 'FAILED')
RETURNING *;
