ALTER TABLE "users"
    ADD COLUMN "last_device_model" varchar(100),
    ADD COLUMN "last_platform" varchar(10),
    ADD COLUMN "last_os_version" varchar(30),
    ADD COLUMN "last_app_version" varchar(30),
    ADD COLUMN "last_seen_at" timestamptz;
