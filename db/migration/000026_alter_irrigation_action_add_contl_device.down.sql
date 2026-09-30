ALTER TABLE "irrigation_actions"
DROP CONSTRAINT IF EXISTS "irrigation_actions_controller_device_id_fkey";

ALTER TABLE "irrigation_actions"
DROP COLUMN IF EXISTS "controller_device_id";