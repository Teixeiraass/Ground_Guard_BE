package dto

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateIrrigationPreferenceRequest struct {
	DeviceUUID           string  `json:"device_uuid" binding:"required"`
	IrrigationMode       string  `json:"irrigation_mode" binding:"required,oneof=INTELIGENTE MANUAL"`
	MoistureThreshold    int32   `json:"moisture_threshold" binding:"required,max=100"`
	DryTimeMinutes       int32   `json:"dry_time_minutes" binding:"required"`
	MaxIrrigationsPerDay int32   `json:"max_irrigation_per_day" binding:"required,min=1"`
	StartHour            *string `json:"start_hour" binding:"omitempty"`
	EndHour              *string `json:"end_hour" binding:"omitempty"`
}

type IrrigationPreferenceResponse struct {
	Uuid                      uuid.UUID `json:"uuid"`
	DeviceID                  int64     `json:"device_id"`
	Enabled                   bool      `json:"enabled"`
	IrrigationMode            string    `json:"irrigation_mode"`
	MoistureThreshold         int32     `json:"moisture_threshold"`
	DryTimeMinutes            int32     `json:"dry_time_minutes"`
	IrrigationDurationSeconds int32     `json:"irrigation_duration_seconds"`
	MaxIrrigationsPerDay      int32     `json:"max_irrigations_per_day"`
	StartHour                 *string   `json:"start_hour,omitempty"`
	EndHour                   *string   `json:"end_hour,omitempty"`
}

func NewIrrigationPreferenceResponse(irrigationPreference db.IrrigationPreference) IrrigationPreferenceResponse {
	var startHour *string
	if irrigationPreference.StartHour.Valid {
		s := irrigationPreference.StartHour.Time.Format("15:04")
		startHour = &s
	}

	var endHour *string
	if irrigationPreference.EndHour.Valid {
		s := irrigationPreference.EndHour.Time.Format("15:04")
		endHour = &s
	}

	return IrrigationPreferenceResponse{
		Uuid:                      irrigationPreference.Uuid,
		DeviceID:                  irrigationPreference.DeviceID,
		Enabled:                   irrigationPreference.Enabled,
		IrrigationMode:            irrigationPreference.IrrigationMode,
		MoistureThreshold:         irrigationPreference.MoistureThreshold,
		DryTimeMinutes:            irrigationPreference.DryTimeMinutes,
		IrrigationDurationSeconds: irrigationPreference.IrrigationDurationSeconds,
		MaxIrrigationsPerDay:      irrigationPreference.MaxIrrigationsPerDay,
		StartHour:                 startHour,
		EndHour:                   endHour,
	}
}

type GetIrrigationPreferenceRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}