package mqtt

import "encoding/json"

const PairingStartAction = "PAIRING_START"

type PairingStartCommand struct {
	CommandID string `json:"command_id"`
	Action    string `json:"action"`
}

func NewPairingStartCommand(commandID string) ([]byte, error) {
	return json.Marshal(PairingStartCommand{
		CommandID: commandID,
		Action:    PairingStartAction,
	})
}
