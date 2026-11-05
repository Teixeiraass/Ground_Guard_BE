ALTER TABLE "controller_devices"
  DROP COLUMN IF EXISTS "auto_update",
  DROP COLUMN IF EXISTS "ota_status",
  DROP COLUMN IF EXISTS "ota_target_version",
  DROP COLUMN IF EXISTS "ota_error",
  DROP COLUMN IF EXISTS "ota_requested_at",
  DROP COLUMN IF EXISTS "ota_updated_at",
  DROP COLUMN IF EXISTS "update_notified_version";

DROP TABLE IF EXISTS "firmware_releases";
