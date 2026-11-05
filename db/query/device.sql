-- name: CreateDevice :one
INSERT INTO devices (
  device_uid, 
  name,
  last_update,
  last_seen,
  status,
  qr_token,
  qr_code_file,
  controller_device_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetDevice :one
SELECT * FROM devices
WHERE uuid = $1 LIMIT 1;

-- name: GetDeviceByID :one
SELECT * FROM devices
WHERE id = $1 LIMIT 1;

-- name: GetDeviceForUpdate :one
SELECT * FROM devices
WHERE uuid = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListDevices :many
SELECT * FROM devices
WHERE controller_device_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListDevicesByUserID :many
SELECT devices.*
FROM devices
JOIN controller_devices ON controller_devices.id = devices.controller_device_id
WHERE controller_devices.user_id = $1
ORDER BY devices.id
LIMIT $2
OFFSET $3;

-- name: ListDevicesByControllerUUIDAndUserID :many
SELECT devices.*
FROM devices
JOIN controller_devices ON controller_devices.id = devices.controller_device_id
WHERE controller_devices.uuid = @controller_uuid
  AND controller_devices.user_id = @user_id
ORDER BY devices.id
LIMIT @page_limit
OFFSET @page_offset;

-- name: UpdateDevices :one
UPDATE devices
set status = $2
WHERE uuid = $1
RETURNING *;

-- name: LinkDeviceToControllerDeviceByQrToken :one
UPDATE devices
SET controller_device_id = $2
WHERE qr_token = $1 AND controller_device_id IS NULL
RETURNING *;

-- name: LinkDeviceToUserControllerByQrToken :one
UPDATE devices
SET controller_device_id = controller_devices.id
FROM controller_devices
WHERE devices.qr_token = $1
  AND controller_devices.uuid = $2
  AND controller_devices.user_id = $3
  AND devices.controller_device_id IS NULL
RETURNING devices.*;

-- name: UnlinkDeviceFromUserController :one
UPDATE devices
SET controller_device_id = NULL
FROM controller_devices
WHERE devices.uuid = $1
  AND controller_devices.id = devices.controller_device_id
  AND controller_devices.user_id = $2
RETURNING devices.*;

-- name: UnlinkDeviceFromControllerDevice :one
UPDATE devices
SET controller_device_id = NULL
WHERE uuid = $1 AND controller_device_id = $2
RETURNING *;

-- name: UpdateNameDevice :one
UPDATE devices
set name = $2
WHERE uuid = $1
RETURNING *;

-- name: GetDeviceByUID :one
SELECT * FROM devices
WHERE device_uid = $1 LIMIT 1;

-- name: GetDeviceUserID :one
SELECT controller_devices.user_id
FROM devices
JOIN controller_devices ON controller_devices.id = devices.controller_device_id
WHERE devices.uuid = $1
LIMIT 1;

-- name: GetDeviceUserIDByUID :one
SELECT controller_devices.user_id
FROM devices
JOIN controller_devices ON controller_devices.id = devices.controller_device_id
WHERE devices.device_uid = $1
LIMIT 1;

-- name: UpdateDeviceTelemetryByUID :one
-- A NULL soil_moisture keeps the last good reading (missing field or faulty sensor).
UPDATE devices
SET
    last_seen = @last_seen,
    status = @status,
    soil_moisture = COALESCE(sqlc.narg(soil_moisture), soil_moisture),
    soil_sensor_ok = sqlc.narg(soil_sensor_ok),
    soil_raw = sqlc.narg(soil_raw),
    wifi_rssi = sqlc.narg(wifi_rssi)
WHERE device_uid = @device_uid
RETURNING *;

-- name: UpdateDeviceLocation :one
UPDATE devices
SET
    latitude = sqlc.narg(latitude),
    longitude = sqlc.narg(longitude)
WHERE uuid = @uuid
RETURNING *;

-- name: UpdateDeviceRegistration :one
UPDATE devices
SET
    status = $2,
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

-- name: AssignDeviceToGroup :one
UPDATE devices
SET group_id = $2
WHERE uuid = $1
RETURNING *;