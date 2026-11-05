package mqtt

import (
	"encoding/json"

	"github.com/google/uuid"
)

const OtaUpdateAction = "OTA_UPDATE"

// OTA statuses reported by the controller on its events topic.
const (
	OtaStatusPending     = "PENDING"
	OtaStatusDownloading = "DOWNLOADING"
	OtaStatusSuccess     = "SUCCESS"
	OtaStatusFailed      = "FAILED"
	OtaStatusUpToDate    = "UP_TO_DATE"
)

// The command only names the target version; the firmware fetches the
// download URL and SHA-256 from the API itself, so a forged MQTT message
// can at most trigger an update to the officially published release.
type OtaUpdateCommand struct {
	CommandID string               `json:"command_id"`
	Action    string               `json:"action"`
	Data      OtaUpdateCommandData `json:"data"`
}

type OtaUpdateCommandData struct {
	Version string `json:"version"`
}

func NewOtaUpdateCommand(commandID, version string) ([]byte, error) {
	if _, err := uuid.Parse(commandID); err != nil {
		return nil, err
	}
	return json.Marshal(OtaUpdateCommand{
		CommandID: commandID,
		Action:    OtaUpdateAction,
		Data:      OtaUpdateCommandData{Version: version},
	})
}

// OtaEventPayload is published by the firmware on the events topic while
// an update runs. It is told apart from command acks by ota_status.
type OtaEventPayload struct {
	OtaStatus string `json:"ota_status"`
	Version   string `json:"version"`
	Error     string `json:"error,omitempty"`
}
