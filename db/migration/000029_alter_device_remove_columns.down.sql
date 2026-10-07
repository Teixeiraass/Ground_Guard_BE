ALTER TABLE "devices"
ADD COLUMN "firmware_version" VARCHAR(50),
ADD COLUMN "firmware_build" VARCHAR(50),
ADD COLUMN "ip_address" VARCHAR(45),
ADD COLUMN "wifi_ssid" VARCHAR(255),
ADD COLUMN "user_id" BIGINT;