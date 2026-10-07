-- name: CreateControllerDevice :one
INSERT INTO controller_devices (
  controller_device_uid, 
  name, 
  firmware_version,
  firmware_build,
  last_update,
  ip_address,
  wifi_ssid,
  last_seen,
  user_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetControllerDevice :one
SELECT * FROM controller_devices
WHERE uuid = $1 LIMIT 1;

-- name: GetControllerDeviceForUpdate :one
SELECT * FROM controller_devices
WHERE uuid = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListControllerDevices :many
SELECT * FROM controller_devices
WHERE user_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UnlinkControllerDeviceFromUser :one
UPDATE controller_devices
SET user_id = NULL
WHERE uuid = $1 AND user_id = $2
RETURNING *;

-- name: UpdateNameControllerDevice :one
UPDATE controller_devices
set name = $2
WHERE uuid = $1
RETURNING *;

-- name: GetControllerDeviceByUID :one
SELECT * FROM controller_devices
WHERE UPPER(controller_device_uid) = UPPER(CAST(@controller_device_uid AS TEXT)) LIMIT 1;

-- name: GetControllerDeviceByID :one
SELECT * FROM controller_devices
WHERE id = $1 LIMIT 1;

-- name: ClaimControllerDeviceByUID :one
UPDATE controller_devices
SET user_id = @user_id
WHERE UPPER(controller_device_uid) = UPPER(CAST(@controller_device_uid AS TEXT)) AND (user_id IS NULL OR user_id = @user_id)
RETURNING *;

-- name: UpdateControllerDeviceTelemetryByUID :one
UPDATE controller_devices
SET
    last_seen = @last_seen,
    ip_address = @ip_address,
    wifi_ssid = @wifi_ssid
WHERE UPPER(controller_device_uid) = UPPER(CAST(@controller_device_uid AS TEXT))
RETURNING *;

-- name: UpdateControllerDeviceRegistration :one
UPDATE controller_devices
SET
    firmware_version = @firmware_version,
    firmware_build = @firmware_build,
    ip_address = @ip_address,
    wifi_ssid = @wifi_ssid,
    last_seen = NOW()
WHERE UPPER(controller_device_uid) = UPPER(CAST(@controller_device_uid AS TEXT))
RETURNING *;

-- name: UpdateControllerDeviceState :one
UPDATE controller_devices
SET
    is_online = @is_online
WHERE UPPER(controller_device_uid) = UPPER(CAST(@controller_device_uid AS TEXT))
RETURNING *;
