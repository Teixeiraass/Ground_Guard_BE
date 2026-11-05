ALTER TABLE "irrigation_schedules"
    ADD COLUMN "irrigation_mode" varchar(20) NOT NULL DEFAULT 'DURATION',
    ADD COLUMN "moisture_threshold" integer,
    ADD COLUMN "last_triggered_at" timestamptz;
