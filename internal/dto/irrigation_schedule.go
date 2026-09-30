package dto

import (
	"database/sql"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateIrrigationScheduleRequest struct {
	DeviceUUID      string `json:"device_uuid" binding:"required"`
	Name            string `json:"name" binding:"omitempty,max=100"`
	Enabled         *bool  `json:"enabled"`
	StartTime       string `json:"start_time" binding:"required"`
	DurationSeconds int32  `json:"duration_seconds" binding:"required,min=1,max=86400"`
	DaysOfWeek      string `json:"days_of_week" binding:"required"`
}

type UpdateIrrigationScheduleRequest struct {
	Name            string `json:"name" binding:"omitempty,max=100"`
	Enabled         bool   `json:"enabled"`
	StartTime       string `json:"start_time" binding:"required"`
	DurationSeconds int32  `json:"duration_seconds" binding:"required,min=1,max=86400"`
	DaysOfWeek      string `json:"days_of_week" binding:"required"`
}

type IrrigationScheduleResponse struct {
	Uuid            uuid.UUID `json:"uuid"`
	DeviceID        int64     `json:"device_id"`
	Name            *string   `json:"name,omitempty"`
	Enabled         bool      `json:"enabled"`
	StartTime       string    `json:"start_time"`
	DurationSeconds int32     `json:"duration_seconds"`
	DaysOfWeek      string    `json:"days_of_week"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewIrrigationScheduleResponse(schedule db.IrrigationSchedule) IrrigationScheduleResponse {
	var name *string
	if schedule.Name.Valid {
		name = &schedule.Name.String
	}

	return IrrigationScheduleResponse{
		Uuid:            schedule.Uuid,
		DeviceID:        schedule.DeviceID,
		Name:            name,
		Enabled:         schedule.Enabled,
		StartTime:       schedule.StartTime.Format("15:04"),
		DurationSeconds: schedule.DurationSeconds,
		DaysOfWeek:      schedule.DaysOfWeek,
		CreatedAt:       schedule.CreatedAt,
		UpdatedAt:       schedule.UpdatedAt,
	}
}

func ScheduleTime(value string) (time.Time, error) {
	return time.Parse("15:04", value)
}

func ScheduleName(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
