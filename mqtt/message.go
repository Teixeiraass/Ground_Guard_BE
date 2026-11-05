package mqtt

// TelemetryPayload is the expected JSON payload published by ESP32 devices.
// Pointer fields are optional: older firmware does not send them.
type TelemetryPayload struct {
	Status               string `json:"status"`
	IPAddress            string `json:"ip_address"`
	WifiSSID             string `json:"wifi_ssid"`
	SoilMoisture         *int32 `json:"soil_moisture"`
	SoilRaw              *int32 `json:"soil_raw"`
	SensorOK             *bool  `json:"sensor_ok"`
	RSSI                 *int32 `json:"rssi"`
	ProvisioningRequired bool   `json:"provisioning_required"`
	// Sent by controllers; the source of truth for the installed version.
	FirmwareVersion string `json:"firmware_version"`
	FirmwareBuild   string `json:"firmware_build"`
}

// ValidSoilMoisture returns the reading only when it is present, in range
// and the firmware did not flag the probe as faulty.
func (t TelemetryPayload) ValidSoilMoisture() (int32, bool) {
	if t.SoilMoisture == nil || (t.SensorOK != nil && !*t.SensorOK) {
		return 0, false
	}
	if *t.SoilMoisture < 0 || *t.SoilMoisture > 100 {
		return 0, false
	}
	return *t.SoilMoisture, true
}

// CommandPayload is the JSON payload sent from the API to ESP32 devices.
type CommandPayload struct {
	Action string         `json:"action"`
	Data   map[string]any `json:"data,omitempty"`
}
