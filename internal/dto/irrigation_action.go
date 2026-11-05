package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type ListIrrigationHistoryRequest struct {
	Date       string `form:"date" binding:"omitempty"`
	DeviceUUID string `form:"device_uuid" binding:"omitempty"`
}

type GetIrrigationHistoryRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type GetWaterConsumptionRequest struct {
	Period     string `form:"period" binding:"omitempty,oneof=day week month"`
	DeviceUUID string `form:"device_uuid" binding:"omitempty"`
}

type IrrigationHistoryResponse struct {
	Uuid            uuid.UUID  `json:"uuid"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	DurationSeconds *int32     `json:"duration_seconds,omitempty"`
	Status          string     `json:"status"`
	TriggerType     string     `json:"trigger_type"`
	WaterVolumeMl   *int32     `json:"water_volume_ml,omitempty"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func NewIrrigationHistoryResponse(action db.IrrigationAction) IrrigationHistoryResponse {
	var finishedAt *time.Time
	if action.FinishedAt.Valid {
		finishedAt = &action.FinishedAt.Time
	}

	var durationSeconds *int32
	if action.DurationSeconds.Valid {
		durationSeconds = &action.DurationSeconds.Int32
	}

	var waterVolumeMl *int32
	if action.WaterVolumeMl.Valid {
		waterVolumeMl = &action.WaterVolumeMl.Int32
	}

	var errorMessage *string
	if action.ErrorMessage.Valid {
		errorMessage = &action.ErrorMessage.String
	}

	return IrrigationHistoryResponse{
		Uuid:            action.Uuid,
		StartedAt:       action.StartedAt,
		FinishedAt:      finishedAt,
		DurationSeconds: durationSeconds,
		Status:          action.Status,
		TriggerType:     action.TriggerType,
		WaterVolumeMl:   waterVolumeMl,
		ErrorMessage:    errorMessage,
		CreatedAt:       action.CreatedAt,
	}
}

type WaterConsumptionPoint struct {
	Label         string `json:"label"`
	WaterVolumeMl int64  `json:"water_volume_ml"`
}

type WaterConsumptionResponse struct {
	Period             string                  `json:"period"`
	CurrentPeriodStart time.Time               `json:"current_period_start"`
	CurrentPeriodEnd   time.Time               `json:"current_period_end"`
	CurrentTotalMl     int64                   `json:"current_total_ml"`
	PreviousTotalMl    int64                   `json:"previous_total_ml"`
	DifferenceMl       int64                   `json:"difference_ml"`
	ChangePercent      float64                 `json:"change_percent"`
	Points             []WaterConsumptionPoint `json:"points"`
}
