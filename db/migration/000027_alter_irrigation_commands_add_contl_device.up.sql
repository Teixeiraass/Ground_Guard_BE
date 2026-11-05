ALTER TABLE "irrigation_commands"
ADD COLUMN controller_device_id BIGINT DEFAULT NULL;

ALTER TABLE "irrigation_commands" ADD FOREIGN KEY ("controller_device_id") REFERENCES "controller_devices" ("id") DEFERRABLE INITIALLY IMMEDIATE;