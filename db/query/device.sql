-- name: CreateDevice :one
INSERT INTO devices (
  device_uid, 
  name, 
  firmware_version,
  firmware_build,
  last_update,
  ip_address,
  wifi_ssid,
  last_seen,
  status,
  user_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetDevice :one
SELECT * FROM devices
WHERE uuid = $1 LIMIT 1;

-- name: GetDeviceForUpdate :one
SELECT * FROM devices
WHERE uuid = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListDevices :many
SELECT * FROM devices
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateDevices :one
UPDATE devices
set status = $2
WHERE uuid = $1
RETURNING *;