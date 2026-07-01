package dto

type IrrigationCommandPayload struct {
	CommandID       string `json:"commandId"`
	Action          string `json:"action"`
	DurationSeconds *int32 `json:"durationSeconds,omitempty"`
}
