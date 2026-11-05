-- name: CreateIrrigationAction :one
INSERT INTO irrigation_actions (
    device_id,
    user_id,
    duration_seconds,
    status,
    trigger_type,
    error_message,
    schedule_id
) VALUES (
    $1,$2,$3,$4,$5,$6,$7
)
RETURNING *;

-- name: ListActiveScheduledIrrigationActions :many
SELECT
    ia.id                   AS id,
    ia.uuid                 AS uuid,
    ia.device_id            AS device_id,
    ia.user_id              AS user_id,
    ia.started_at           AS started_at,
    isch.id                 AS schedule_id,
    isch.irrigation_mode    AS irrigation_mode,
    isch.moisture_threshold AS moisture_threshold,
    isch.duration_seconds   AS schedule_duration_seconds,
    d.uuid                  AS device_uuid,
    d.soil_moisture         AS soil_moisture
FROM irrigation_actions ia
JOIN irrigation_schedules isch ON isch.id = ia.schedule_id
JOIN devices d ON d.id = ia.device_id
WHERE ia.status = 'ATIVO' AND ia.schedule_id IS NOT NULL;

-- name: GetIrrigationAction :one
SELECT * FROM irrigation_actions
WHERE uuid = $1 LIMIT 1;

-- name: GetActiveIrrigationActionByDevice :one
SELECT *
FROM irrigation_actions
WHERE device_id = $1
AND status = 'ATIVO'
LIMIT 1;

-- name: ExistsActiveIrrigationAction :one
SELECT EXISTS (
    SELECT 1
    FROM irrigation_actions
    WHERE device_id = $1
      AND status = 'ATIVO'
);

-- name: ListIrrigationAction :many
SELECT * FROM irrigation_actions
WHERE user_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateIrrigationAction :one
UPDATE irrigation_actions
SET
    finished_at = $2,
    duration_seconds = $3,
    status = $4,
    water_volume_ml = $5,
    error_message = $6
WHERE uuid = $1
RETURNING *;

-- name: DeleteIrrigationAction :exec
DELETE FROM irrigation_actions WHERE id = $1;