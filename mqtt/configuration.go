package mqtt

import (
	"encoding/json"

	"github.com/google/uuid"
)

const ConfigureControllerAction = "CONFIGURE_CONTROLLER"

type ControllerConfigurationCommand struct {
	CommandID string                             `json:"command_id"`
	Action    string                             `json:"action"`
	Data      ControllerConfigurationCommandData `json:"data"`
}

type ControllerConfigurationCommandData struct {
	ControllerUUID string `json:"controller_uuid"`
	Timezone       string `json:"timezone"`
}

func NewControllerConfigurationCommand(commandID, controllerUUID, timezone string) ([]byte, error) {
	if _, err := uuid.Parse(commandID); err != nil {
		return nil, err
	}
	return json.Marshal(ControllerConfigurationCommand{
		CommandID: commandID,
		Action:    ConfigureControllerAction,
		Data: ControllerConfigurationCommandData{
			ControllerUUID: controllerUUID,
			Timezone:       timezone,
		},
	})
}
