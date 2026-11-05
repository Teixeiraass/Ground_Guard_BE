ALTER TABLE "irrigation_commands"
    ADD COLUMN "schedule_id" bigint;

ALTER TABLE "irrigation_commands"
    ADD FOREIGN KEY ("schedule_id") REFERENCES "irrigation_schedules" ("id")
    ON DELETE SET NULL DEFERRABLE INITIALLY IMMEDIATE;

CREATE INDEX ON "irrigation_commands" ("schedule_id");
