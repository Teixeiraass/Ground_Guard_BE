package mqtt

// TelemetryPayload is the expected JSON payload published by ESP32 devices.
type TelemetryPayload struct {
	Status    string `json:"status"`
	IPAddress string `json:"ip_address"`
	WifiSSID  string `json:"wifi_ssid"`
}

// CommandPayload is the JSON payload sent from the API to ESP32 devices.
type CommandPayload struct {
	Action string         `json:"action"`
	Data   map[string]any `json:"data,omitempty"`
}
