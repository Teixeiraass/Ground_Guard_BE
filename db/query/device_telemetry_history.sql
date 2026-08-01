-- name: CreateDeviceSensorHistory :one
INSERT INTO device_sensor_history (
    device_id,
    soil_moisture,
    temperature,
    battery
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: ListDeviceSensorHistory :many
SELECT *
FROM device_sensor_history
WHERE device_id = $1
  AND created_at >= $2
ORDER BY created_at ASC;
-- name: GetLatestDeviceSensorHistory :one
SELECT *
FROM device_sensor_history
WHERE device_id = $1
ORDER BY created_at DESC
LIMIT 1;
