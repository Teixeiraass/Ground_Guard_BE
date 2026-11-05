CREATE TABLE "controller_device_configuration_commands" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "controller_device_id" bigint NOT NULL,
  "action" varchar(40) NOT NULL,
  "timezone" varchar(100) NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'PENDING',
  "attempts" integer NOT NULL DEFAULT 0,
  "last_attempt_at" timestamptz,
  "next_attempt_at" timestamptz,
  "error_message" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  "acknowledged_at" timestamptz
);

CREATE INDEX ON "controller_device_configuration_commands" ("controller_device_id");
CREATE INDEX ON "controller_device_configuration_commands" ("status");
CREATE UNIQUE INDEX controller_device_configuration_active_idx
  ON "controller_device_configuration_commands" ("controller_device_id")
  WHERE status IN ('PENDING', 'SENT');

ALTER TABLE "controller_device_configuration_commands"
ADD FOREIGN KEY ("controller_device_id") REFERENCES "controller_devices" ("id")
DEFERRABLE INITIALLY IMMEDIATE;
