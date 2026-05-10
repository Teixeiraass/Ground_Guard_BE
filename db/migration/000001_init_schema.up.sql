CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE DEFAULT (gen_random_uuid()),
  "username" varchar NOT NULL,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "devices" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE DEFAULT (gen_random_uuid()),
  "device_uid" varchar(100) UNIQUE NOT NULL,
  "name" varchar(100) NOT NULL,
  "firmware_version" varchar(50) NOT NULL DEFAULT '1.0.0', 
  "firmware_build" varchar(50),
  "last_update" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "ip_address" inet,
  "wifi_ssid" varchar(100),
  "last_seen" timestamptz,
  "status" varchar(20) NOT NULL DEFAULT 'ATIVO',
  "user_id" bigint,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "devices" ("user_id");

CREATE UNIQUE INDEX ON "devices" ("device_uid");

CREATE UNIQUE INDEX ON "devices" ("uuid");

ALTER TABLE "devices" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;