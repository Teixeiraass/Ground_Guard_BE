ALTER TABLE "irrigation_commands"
DROP CONSTRAINT IF EXISTS "irrigation_commands_controller_device_id_fkey";

ALTER TABLE "irrigation_commands"
DROP COLUMN IF EXISTS "controller_device_id";