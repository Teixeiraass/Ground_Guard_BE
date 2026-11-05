ALTER TABLE "irrigation_schedules"
ADD COLUMN controller_device_id BIGINT DEFAULT NULL;

ALTER TABLE "irrigation_schedules" ADD FOREIGN KEY ("controller_device_id") REFERENCES "controller_devices" ("id") DEFERRABLE INITIALLY IMMEDIATE;