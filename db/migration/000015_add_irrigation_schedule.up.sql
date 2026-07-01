CREATE TABLE "irrigation_schedules" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),

  "device_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,

  "name" varchar(100),

  "enabled" boolean NOT NULL DEFAULT true,

  "start_time" time NOT NULL,

  "duration_seconds" integer NOT NULL,

  "days_of_week" varchar(30) NOT NULL,

  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "irrigation_schedules" ("uuid");

CREATE INDEX ON "irrigation_schedules" ("device_id");

CREATE INDEX ON "irrigation_schedules" ("user_id");

CREATE INDEX ON "irrigation_schedules" ("enabled");

ALTER TABLE "irrigation_schedules"
ADD FOREIGN KEY ("device_id")
REFERENCES "devices" ("id")
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_schedules"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
DEFERRABLE INITIALLY IMMEDIATE;