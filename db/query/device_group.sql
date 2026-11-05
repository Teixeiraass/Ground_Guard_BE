-- name: CreateDeviceGroup :one
INSERT INTO device_groups (
    user_id,
    name
) VALUES ($1, $2)
RETURNING *;

-- name: GetDeviceGroup :one
SELECT *
FROM device_groups
WHERE uuid = $1 AND user_id = $2
LIMIT 1;

-- name: ListDeviceGroupsByUser :many
SELECT *
FROM device_groups
WHERE user_id = $1
ORDER BY name;

-- name: UpdateDeviceGroupName :one
UPDATE device_groups
SET name = $3,
    updated_at = now()
WHERE uuid = $1 AND user_id = $2
RETURNING *;

-- name: DeleteDeviceGroup :one
DELETE FROM device_groups
WHERE uuid = $1 AND user_id = $2
RETURNING uuid;

-- name: ListDevicesByGroupID :many
SELECT *
FROM devices
WHERE group_id = $1
ORDER BY name;
