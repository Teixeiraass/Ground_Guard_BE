CREATE TABLE "irrigation_schedule_history" (
  "id" bigserial PRIMARY KEY,

  "schedule_id" bigint NOT NULL,

  "started_at" timestamptz,

  "finished_at" timestamptz,

  "status" varchar(20) NOT NULL,

  "error_message" text,

  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "irrigation_schedule_history" ("schedule_id");

CREATE INDEX ON "irrigation_schedule_history" ("started_at");

CREATE INDEX ON "irrigation_schedule_history" ("status");

ALTER TABLE "irrigation_schedule_history"
ADD FOREIGN KEY ("schedule_id")
REFERENCES "irrigation_schedules" ("id")
DEFERRABLE INITIALLY IMMEDIATE;