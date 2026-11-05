ALTER TABLE "irrigation_schedules"
    DROP COLUMN IF EXISTS "last_triggered_at",
    DROP COLUMN IF EXISTS "moisture_threshold",
    DROP COLUMN IF EXISTS "irrigation_mode";
