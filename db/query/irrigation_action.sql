-- name: CreateIrrigationAction :one
INSERT INTO irrigation_actions (
  device_id, 
  user_id, 
  started_at,
  finished_at,
  duration_seconds,
  status,
  trigger_type,
  water_volume_ml,
  error_message,
  command_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;
 
-- name: GetIrrigationAction :one
SELECT * FROM irrigation_actions
WHERE uuid = $1 LIMIT 1;

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
  error_message = $6,
  command_id = $7
WHERE uuid = $1
RETURNING *;

-- name: DeleteIrrigationAction :exec
DELETE FROM irrigation_actions WHERE id = $1;