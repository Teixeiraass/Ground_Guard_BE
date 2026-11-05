package dto

// command_id matches every other command and the controller firmware;
// commandId is kept for sensors flashed with the older firmware.
type IrrigationCommandPayload struct {
	CommandID       string `json:"command_id"`
	LegacyCommandID string `json:"commandId"`
	Action          string `json:"action"`
	DurationSeconds *int32 `json:"durationSeconds,omitempty"`
}
