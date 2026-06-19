-- name: CreateIrrigationPreferences :one
INSERT INTO irrigation_preferences (
  device_id, 
  irrigation_mode,
  moisture_threshold,
  dry_time_minutes,
  max_irrigations_per_day,
  start_hour,
  end_hour
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetIrrigationPreference :one
SELECT * FROM irrigation_preferences
WHERE uuid = $1 LIMIT 1;

-- name: GetIrrigationPreferenceByDevice :one
SELECT * FROM irrigation_preferences 
WHERE device_id = $1 LIMIT 1;

-- name: ListIrrigationPreferences :many
SELECT * FROM irrigation_preferences
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateIrrigationPreference :one
UPDATE irrigation_preferences
SET
  enabled = COALESCE(sqlc.narg('enabled'), enabled),
  irrigation_mode = COALESCE(sqlc.narg('irrigation_mode'), irrigation_mode),
  moisture_threshold = COALESCE(sqlc.narg('moisture_threshold'), moisture_threshold),
  dry_time_minutes = COALESCE(sqlc.narg('dry_time_minutes'), dry_time_minutes),
  irrigation_duration_seconds = COALESCE(sqlc.narg('irrigation_duration_seconds'), irrigation_duration_seconds),
  max_irrigations_per_day = COALESCE(sqlc.narg('max_irrigations_per_day'), max_irrigations_per_day),
  start_hour = COALESCE(sqlc.narg('start_hour'), start_hour),
  end_hour = COALESCE(sqlc.narg('end_hour'), end_hour),
  updated_at = now()
WHERE uuid = sqlc.arg('uuid')
RETURNING *;

-- name: DeleteIrrigationPreference :exec
DELETE FROM irrigation_preferences WHERE uuid = $1;

-- name: DeleteIrrigationPreferenceByDeviceId :exec
DELETE FROM irrigation_preferences WHERE device_id = $1;