CREATE TABLE "firmware_releases" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "version" varchar(50) UNIQUE NOT NULL,
  "build" varchar(50) NOT NULL,
  "file_path" varchar(255) NOT NULL,
  "file_size" bigint NOT NULL,
  "sha256" varchar(64) NOT NULL,
  "release_notes" text NOT NULL DEFAULT '',
  "published" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "firmware_releases" ("published", "created_at");

ALTER TABLE "controller_devices"
  ADD COLUMN "auto_update" boolean NOT NULL DEFAULT false,
  ADD COLUMN "ota_status" varchar(20),
  ADD COLUMN "ota_target_version" varchar(50),
  ADD COLUMN "ota_error" varchar(255),
  ADD COLUMN "ota_requested_at" timestamptz,
  ADD COLUMN "ota_updated_at" timestamptz,
  ADD COLUMN "update_notified_version" varchar(50);
