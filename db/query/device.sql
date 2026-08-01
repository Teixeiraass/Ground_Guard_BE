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
  user_id,
  qr_token,
  qr_code_file
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
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
WHERE user_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateDevices :one
UPDATE devices
set status = $2
WHERE uuid = $1
RETURNING *;

-- name: LinkDeviceToUserByQrToken :one
UPDATE devices
SET user_id = $2
WHERE qr_token = $1 AND user_id IS NULL
RETURNING *;

-- name: UnlinkDeviceFromUser :one
UPDATE devices
SET user_id = NULL
WHERE uuid = $1 AND user_id = $2
RETURNING *;

-- name: UpdateNameDevice :one
UPDATE devices
set name = $2
WHERE uuid = $1
RETURNING *;

-- name: GetDeviceByUID :one
SELECT * FROM devices
WHERE device_uid = $1 LIMIT 1;

-- name: UpdateDeviceTelemetryByUID :one
UPDATE devices
SET
    last_seen = $2,
    status = $3,
    ip_address = $4,
    wifi_ssid = $5,
    soil_moisture = $6
WHERE device_uid = $1
RETURNING *;

-- name: UpdateDeviceRegistration :one
UPDATE devices
SET
    firmware_version = $2,
    firmware_build = $3,
    ip_address = $4,
    wifi_ssid = $5,
    status = $6,
    last_seen = NOW()
WHERE device_uid = $1
RETURNING *;

-- name: UpdateDeviceState :one
UPDATE devices
SET
    is_online = $2, 
    is_irrigating = $3
WHERE device_uid = $1
RETURNING *;