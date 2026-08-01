-- name: CreateIrrigationAction :one
INSERT INTO irrigation_actions (
    device_id,
    user_id,
    duration_seconds,
    status,
    trigger_type,
    error_message
) VALUES (
    $1,$2,$3,$4,$5,$6
)
RETURNING *;
 
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