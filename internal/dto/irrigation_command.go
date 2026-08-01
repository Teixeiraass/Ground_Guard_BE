package dto

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateIrrigationCommandRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Action   string `json:"action" binding:"required,oneof=START STOP"`
	Duration *int32 `json:"duration,omitempty"`
}

type CreateIrrigationCommandResponse struct {
	CommandID uuid.UUID `json:"command_id"`
	Status    string    `json:"status"`
}

func NewIrrigationCommandResponse(command db.IrrigationCommand) CreateIrrigationCommandResponse {
	return CreateIrrigationCommandResponse{
		CommandID: command.Uuid,
		Status:    command.Status,
	}
}

type GetIrrigationCommandRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}
