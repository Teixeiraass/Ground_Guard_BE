package dto

import (
	"time"

	"github.com/google/uuid"
)

type GetStatisticsRequest struct {
	Date       string `form:"date" binding:"omitempty"`
	Period     string `form:"period" binding:"omitempty,oneof=day week month"`
	DeviceUUID string `form:"device_uuid" binding:"omitempty"`
}

type DeviceStatisticsResponse struct {
	Total  int32 `json:"total"`
	Online int32 `json:"online"`
}

type StatisticsResponse struct {
	GeneratedAt       time.Time                   `json:"generated_at"`
	Date              string                      `json:"date"`
	Period            string                      `json:"period,omitempty"`
	DeviceUUID        *uuid.UUID                  `json:"device_uuid,omitempty"`
	DeviceStatistics  DeviceStatisticsResponse    `json:"device_statistics"`
	IrrigationHistory []IrrigationHistoryResponse `json:"irrigation_history"`
	WaterConsumption  WaterConsumptionResponse    `json:"water_consumption"`
}
