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