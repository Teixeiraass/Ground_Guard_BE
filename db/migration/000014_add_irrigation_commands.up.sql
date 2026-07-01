CREATE TABLE "irrigation_commands" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "device_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "action" varchar(20) NOT NULL,
  "duration_seconds" integer,
  "status" varchar(20) NOT NULL DEFAULT 'PENDING',
  "error_message" text,
  "requested_at" timestamptz NOT NULL DEFAULT (now()),
  "processed_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "irrigation_commands" ("uuid");

CREATE INDEX ON "irrigation_commands" ("device_id");
CREATE INDEX ON "irrigation_commands" ("user_id");
CREATE INDEX ON "irrigation_commands" ("status");

ALTER TABLE "irrigation_commands"
ADD FOREIGN KEY ("device_id") REFERENCES "devices" ("id");

ALTER TABLE "irrigation_commands"
ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "irrigation_actions"
ADD COLUMN "command_id" bigint;

ALTER TABLE "irrigation_actions"
ADD FOREIGN KEY ("command_id")
REFERENCES "irrigation_commands" ("id");

ALTER TABLE "irrigation_actions"
ALTER COLUMN "command_id" SET NOT NULL;