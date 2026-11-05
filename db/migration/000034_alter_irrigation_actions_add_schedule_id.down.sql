ALTER TABLE "irrigation_actions"
    DROP CONSTRAINT IF EXISTS "irrigation_actions_schedule_id_fkey";

ALTER TABLE "irrigation_actions"
    DROP COLUMN IF EXISTS "schedule_id";
