ALTER TABLE "devices"
DROP CONSTRAINT IF EXISTS "devices_controller_device_id_fkey";

DROP INDEX IF EXISTS "devices_controller_device_id_idx";

ALTER TABLE "devices"
DROP COLUMN IF EXISTS "controller_device_id";