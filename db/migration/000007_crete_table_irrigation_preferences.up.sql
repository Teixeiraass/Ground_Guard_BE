CREATE TABLE "irrigation_preferences" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "device_id" bigint UNIQUE NOT NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  "irrigation_mode" varchar(20) NOT NULL DEFAULT 'INTELIGENTE',
  "moisture_threshold" integer NOT NULL DEFAULT 30,
  "dry_time_minutes" integer NOT NULL DEFAULT 10,
  "irrigation_duration_seconds" integer NOT NULL DEFAULT 60,
  "max_irrigations_per_day" integer NOT NULL DEFAULT 10,
  "start_hour" time,
  "end_hour" time,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "irrigation_preferences" ("device_id");

CREATE UNIQUE INDEX ON "irrigation_preferences" ("uuid");

ALTER TABLE "irrigation_preferences" ADD FOREIGN KEY ("device_id") REFERENCES "devices" ("id") DEFERRABLE INITIALLY IMMEDIATE;