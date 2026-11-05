ALTER TABLE "irrigation_commands"
    DROP CONSTRAINT IF EXISTS "irrigation_commands_schedule_id_fkey";

ALTER TABLE "irrigation_commands"
    DROP COLUMN IF EXISTS "schedule_id";
