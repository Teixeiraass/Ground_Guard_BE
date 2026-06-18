-- name: CreateIrrigationPreferenceHistory :one
INSERT INTO irrigation_preferences_history (
  preference_id,
  user_id,
  old_data,
  new_data
)
VALUES (
  $1,
  $2,
  $3,
  $4
)
RETURNING *;