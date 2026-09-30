CREATE TABLE "controller_devices" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE DEFAULT (gen_random_uuid()),
  "controller_device_uid" varchar(100) UNIQUE,
  "name" varchar(100),
  "firmware_version" varchar(50),
  "firmware_build" varchar(50),
  "last_update" timestamptz,
  "ip_address" inet,
  "qr_token" varchar(64) UNIQUE NOT NULL,
  "qr_code_file" varchar(255),
  "wifi_ssid" varchar(100),
  "is_online" boolean NOT NULL DEFAULT false,
  "last_seen" timestamptz,
  "user_id" bigint,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "controller_devices" ("user_id");

CREATE UNIQUE INDEX ON "controller_devices" ("controller_device_uid");

CREATE UNIQUE INDEX ON "controller_devices" ("uuid");

ALTER TABLE "controller_devices" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;