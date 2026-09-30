ALTER TABLE "devices"
ADD COLUMN controller_device_id BIGINT DEFAULT NULL;

ALTER TABLE "devices" ADD FOREIGN KEY ("controller_device_id") REFERENCES "controller_devices" ("id") DEFERRABLE INITIALLY IMMEDIATE;

CREATE INDEX ON "devices" ("controller_device_id");