ALTER TABLE "irrigation_schedules"
DROP CONSTRAINT IF EXISTS "irrigation_schedules_controller_device_id_fkey";

ALTER TABLE "irrigation_schedules"
DROP COLUMN IF EXISTS "controller_device_id";