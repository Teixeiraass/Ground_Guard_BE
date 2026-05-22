CREATE TABLE "irrigation_actions" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "device_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "started_at" timestamptz NOT NULL DEFAULT (now()),
  "finished_at" timestamptz,
  "duration_seconds" integer,
  "status" varchar(20) NOT NULL DEFAULT 'ATIVO',
  "trigger_type" varchar(20) NOT NULL,
  "water_volume_ml" integer,
  "error_message" text,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "irrigation_actions" ("device_id");

CREATE INDEX ON "irrigation_actions" ("user_id");

CREATE INDEX ON "irrigation_actions" ("started_at");

CREATE UNIQUE INDEX ON "irrigation_actions" ("uuid");

ALTER TABLE "irrigation_actions" ADD FOREIGN KEY ("device_id") REFERENCES "devices" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_actions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;