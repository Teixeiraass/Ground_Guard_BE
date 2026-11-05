-- name: CreateIrrigationSchedule :one
INSERT INTO irrigation_schedules (
    device_id,
    user_id,
    name,
    enabled,
    start_time,
    duration_seconds,
    days_of_week,
    irrigation_mode,
    moisture_threshold
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListIrrigationSchedules :many
SELECT *
FROM irrigation_schedules
WHERE user_id = $1
ORDER BY start_time, id;

-- name: ListEnabledIrrigationSchedules :many
SELECT *
FROM irrigation_schedules
WHERE enabled = true;

-- name: GetIrrigationSchedule :one
SELECT *
FROM irrigation_schedules
WHERE uuid = $1 AND user_id = $2
LIMIT 1;

-- name: UpdateIrrigationSchedule :one
UPDATE irrigation_schedules
SET name = $3,
    enabled = $4,
    start_time = $5,
    duration_seconds = $6,
    days_of_week = $7,
    irrigation_mode = $8,
    moisture_threshold = $9,
    updated_at = now()
WHERE uuid = $1 AND user_id = $2
RETURNING *;

-- name: UpdateIrrigationScheduleLastTriggered :exec
UPDATE irrigation_schedules
SET last_triggered_at = $2,
    updated_at = now()
WHERE id = $1;

-- name: SetIrrigationScheduleGroup :exec
UPDATE irrigation_schedules
SET group_id = $2
WHERE id = $1;

-- name: UpdateIrrigationScheduleLeadNotified :exec
UPDATE irrigation_schedules
SET lead_notified_at = $2,
    updated_at = now()
WHERE id = $1;

-- name: DeleteIrrigationSchedule :one
DELETE FROM irrigation_schedules
WHERE uuid = $1 AND user_id = $2
RETURNING uuid;
