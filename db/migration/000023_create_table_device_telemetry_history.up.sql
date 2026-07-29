CREATE TABLE device_sensor_history (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL,
    soil_moisture INTEGER NOT NULL,
    temperature REAL,
    battery INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_device_sensor_history_device
        FOREIGN KEY (device_id)
        REFERENCES devices(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_device_sensor_history_device_created
ON device_sensor_history (device_id, created_at DESC);