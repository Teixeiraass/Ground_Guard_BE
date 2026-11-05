-- name: CreateFirmwareRelease :one
INSERT INTO firmware_releases (
    version,
    build,
    file_path,
    file_size,
    sha256,
    release_notes,
    signature
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetLatestFirmwareRelease :one
SELECT * FROM firmware_releases
WHERE published = true
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: GetFirmwareRelease :one
SELECT * FROM firmware_releases
WHERE uuid = $1 LIMIT 1;

-- name: ListFirmwareReleases :many
SELECT * FROM firmware_releases
ORDER BY created_at DESC, id DESC
LIMIT $1
OFFSET $2;

-- name: SetControllerAutoUpdate :one
UPDATE controller_devices
SET auto_update = @auto_update
WHERE id = @id
RETURNING *;

-- name: StartControllerOta :one
UPDATE controller_devices
SET
    ota_status = 'PENDING',
    ota_target_version = @target_version,
    ota_error = NULL,
    ota_requested_at = now(),
    ota_updated_at = now()
WHERE id = @id
RETURNING *;

-- name: UpdateControllerOtaStatus :one
UPDATE controller_devices
SET
    ota_status = @ota_status,
    ota_error = sqlc.narg(ota_error),
    ota_updated_at = now()
WHERE id = @id
RETURNING *;

-- name: FailStaleControllerOta :many
UPDATE controller_devices
SET
    ota_status = 'FAILED',
    ota_error = 'timeout',
    ota_updated_at = now()
WHERE ota_status IN ('PENDING', 'DOWNLOADING')
  AND ota_requested_at < @older_than
RETURNING *;

-- name: ListAutoUpdateCandidates :many
SELECT * FROM controller_devices
WHERE auto_update = true
  AND user_id IS NOT NULL
  AND is_online = true
  AND (ota_status IS NULL OR ota_status NOT IN ('PENDING', 'DOWNLOADING'));

-- name: ListOwnedControllersNotNotified :many
SELECT * FROM controller_devices
WHERE user_id IS NOT NULL
  AND update_notified_version IS DISTINCT FROM @version::varchar;

-- name: MarkControllerUpdateNotified :exec
UPDATE controller_devices
SET update_notified_version = @version
WHERE id = @id;

-- name: UpdateControllerDeviceFirmware :one
UPDATE controller_devices
SET
    firmware_version = @firmware_version,
    firmware_build = @firmware_build,
    last_seen = now()
WHERE id = @id
RETURNING *;

-- name: CountIrrigatingDevicesByController :one
SELECT count(*) FROM devices
WHERE controller_device_id = $1 AND is_irrigating = true;
