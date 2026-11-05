ALTER TABLE "users"
    DROP COLUMN IF EXISTS "last_seen_at",
    DROP COLUMN IF EXISTS "last_app_version",
    DROP COLUMN IF EXISTS "last_os_version",
    DROP COLUMN IF EXISTS "last_platform",
    DROP COLUMN IF EXISTS "last_device_model";
