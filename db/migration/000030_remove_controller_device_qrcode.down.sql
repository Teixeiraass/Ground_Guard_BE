ALTER TABLE "controller_devices"
ADD COLUMN "qr_token" varchar(64) UNIQUE,
ADD COLUMN "qr_code_file" varchar(255);
