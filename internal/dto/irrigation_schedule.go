package dto

import (
	"database/sql"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateIrrigationScheduleRequest struct {
	DeviceUUID        string `json:"device_uuid" binding:"required"`
	Name              string `json:"name" binding:"omitempty,max=100"`
	Enabled           *bool  `json:"enabled"`
	StartTime         string `json:"start_time" binding:"required"`
	DurationSeconds   int32  `json:"duration_seconds" binding:"required,min=1,max=86400"`
	DaysOfWeek        string `json:"days_of_week" binding:"required"`
	IrrigationMode    string `json:"irrigation_mode" binding:"omitempty,oneof=DURATION HUMIDITY"`
	MoistureThreshold *int32 `json:"moisture_threshold" binding:"omitempty,min=1,max=100"`
}

// CreateGroupScheduleRequest is the same shape as CreateIrrigationScheduleRequest
// minus device_uuid — the device is derived per member from the group, not
// supplied by the client.
type CreateGroupScheduleRequest struct {
	Name              string `json:"name" binding:"omitempty,max=100"`
	Enabled           *bool  `json:"enabled"`
	StartTime         string `json:"start_time" binding:"required"`
	DurationSeconds   int32  `json:"duration_seconds" binding:"required,min=1,max=86400"`
	DaysOfWeek        string `json:"days_of_week" binding:"required"`
	IrrigationMode    string `json:"irrigation_mode" binding:"omitempty,oneof=DURATION HUMIDITY"`
	MoistureThreshold *int32 `json:"moisture_threshold" binding:"omitempty,min=1,max=100"`
}

func (r CreateGroupScheduleRequest) ToScheduleRequest(deviceUUID string) CreateIrrigationScheduleRequest {
	return CreateIrrigationScheduleRequest{
		DeviceUUID:        deviceUUID,
		Name:              r.Name,
		Enabled:           r.Enabled,
		StartTime:         r.StartTime,
		DurationSeconds:   r.DurationSeconds,
		DaysOfWeek:        r.DaysOfWeek,
		IrrigationMode:    r.IrrigationMode,
		MoistureThreshold: r.MoistureThreshold,
	}
}

type UpdateIrrigationScheduleRequest struct {
	Name              string `json:"name" binding:"omitempty,max=100"`
	Enabled           *bool  `json:"enabled" binding:"required"`
	StartTime         string `json:"start_time" binding:"required"`
	DurationSeconds   int32  `json:"duration_seconds" binding:"required,min=1,max=86400"`
	DaysOfWeek        string `json:"days_of_week" binding:"required"`
	IrrigationMode    string `json:"irrigation_mode" binding:"omitempty,oneof=DURATION HUMIDITY"`
	MoistureThreshold *int32 `json:"moisture_threshold" binding:"omitempty,min=1,max=100"`
}

type IrrigationScheduleResponse struct {
	Uuid              uuid.UUID  `json:"uuid"`
	DeviceID          int64      `json:"device_id"`
	Name              *string    `json:"name,omitempty"`
	Enabled           bool       `json:"enabled"`
	StartTime         string     `json:"start_time"`
	DurationSeconds   int32      `json:"duration_seconds"`
	DaysOfWeek        string     `json:"days_of_week"`
	IrrigationMode    string     `json:"irrigation_mode"`
	MoistureThreshold *int32     `json:"moisture_threshold,omitempty"`
	LastTriggeredAt   *time.Time `json:"last_triggered_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func NewIrrigationScheduleResponse(schedule db.IrrigationSchedule) IrrigationScheduleResponse {
	var name *string
	if schedule.Name.Valid {
		name = &schedule.Name.String
	}

	var moistureThreshold *int32
	if schedule.MoistureThreshold.Valid {
		moistureThreshold = &schedule.MoistureThreshold.Int32
	}

	var lastTriggeredAt *time.Time
	if schedule.LastTriggeredAt.Valid {
		lastTriggeredAt = &schedule.LastTriggeredAt.Time
	}

	return IrrigationScheduleResponse{
		Uuid:              schedule.Uuid,
		DeviceID:          schedule.DeviceID,
		Name:              name,
		Enabled:           schedule.Enabled,
		StartTime:         schedule.StartTime.Format("15:04"),
		DurationSeconds:   schedule.DurationSeconds,
		DaysOfWeek:        schedule.DaysOfWeek,
		IrrigationMode:    schedule.IrrigationMode,
		MoistureThreshold: moistureThreshold,
		LastTriggeredAt:   lastTriggeredAt,
		CreatedAt:         schedule.CreatedAt,
		UpdatedAt:         schedule.UpdatedAt,
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
