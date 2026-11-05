ALTER TABLE "irrigation_schedules" DROP CONSTRAINT IF EXISTS "irrigation_schedules_group_id_fkey";
ALTER TABLE "irrigation_schedules" DROP COLUMN IF EXISTS "group_id";

ALTER TABLE "devices" DROP CONSTRAINT IF EXISTS "devices_group_id_fkey";
ALTER TABLE "devices" DROP COLUMN IF EXISTS "group_id";

DROP TABLE IF EXISTS "device_groups";
