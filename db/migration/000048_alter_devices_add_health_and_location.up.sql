ALTER TABLE "devices"
  ADD COLUMN "soil_sensor_ok" boolean,
  ADD COLUMN "soil_raw" integer,
  ADD COLUMN "wifi_rssi" integer,
  ADD COLUMN "latitude" double precision,
  ADD COLUMN "longitude" double precision;
