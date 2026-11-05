ALTER TABLE "devices"
  DROP COLUMN IF EXISTS "soil_sensor_ok",
  DROP COLUMN IF EXISTS "soil_raw",
  DROP COLUMN IF EXISTS "wifi_rssi",
  DROP COLUMN IF EXISTS "latitude",
  DROP COLUMN IF EXISTS "longitude";
