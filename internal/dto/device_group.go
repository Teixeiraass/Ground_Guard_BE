package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateDeviceGroupRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

type UpdateDeviceGroupNameRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

type DeviceGroupResponse struct {
	Uuid            uuid.UUID `json:"uuid"`
	Name            string    `json:"name"`
	DeviceCount     int       `json:"device_count"`
	OnlineCount     int       `json:"online_count"`
	IrrigatingCount int       `json:"irrigating_count"`
	AvgSoilMoisture *float64  `json:"avg_soil_moisture,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewDeviceGroupResponse builds the summary response from the group row plus
// its member devices (fetched separately — see ListDevicesByGroupID — since
// aggregating soil_moisture in SQL runs into Postgres reporting AVG() over a
// possibly-empty set as NOT NULL, which sqlc then scans unsafely).
func NewDeviceGroupResponse(group db.DeviceGroup, devices []db.Device) DeviceGroupResponse {
	onlineCount, irrigatingCount := 0, 0
	var moistureSum int64
	var moistureReadings int

	for _, d := range devices {
		if d.IsOnline {
			onlineCount++
		}
		if d.IsIrrigating {
			irrigatingCount++
		}
		if d.SoilMoisture.Valid {
			moistureSum += int64(d.SoilMoisture.Int32)
			moistureReadings++
		}
	}

	var avgMoisture *float64
	if moistureReadings > 0 {
		avg := float64(moistureSum) / float64(moistureReadings)
		avgMoisture = &avg
	}

	return DeviceGroupResponse{
		Uuid:            group.Uuid,
		Name:            group.Name,
		DeviceCount:     len(devices),
		OnlineCount:     onlineCount,
		IrrigatingCount: irrigatingCount,
		AvgSoilMoisture: avgMoisture,
		CreatedAt:       group.CreatedAt,
		UpdatedAt:       group.UpdatedAt,
	}
}

type DeviceGroupDetailResponse struct {
	DeviceGroupResponse
	Devices []DeviceResponse `json:"devices"`
}

func NewDeviceGroupDetailResponse(group db.DeviceGroup, devices []db.Device) DeviceGroupDetailResponse {
	deviceResponses := make([]DeviceResponse, 0, len(devices))
	for _, d := range devices {
		deviceResponses = append(deviceResponses, NewDeviceResponse(d))
	}

	return DeviceGroupDetailResponse{
		DeviceGroupResponse: NewDeviceGroupResponse(group, devices),
		Devices:             deviceResponses,
	}
}

type DeviceGroupStatisticsResponse struct {
	GroupUuid         uuid.UUID                  `json:"group_uuid"`
	Date              string                     `json:"date"`
	Period            string                     `json:"period"`
	DeviceCount       int                        `json:"device_count"`
	OnlineCount       int                        `json:"online_count"`
	IrrigatingCount   int                        `json:"irrigating_count"`
	AvgSoilMoisture   *float64                   `json:"avg_soil_moisture,omitempty"`
	MinSoilMoisture   *int32                     `json:"min_soil_moisture,omitempty"`
	MaxSoilMoisture   *int32                     `json:"max_soil_moisture,omitempty"`
	WaterConsumption  WaterConsumptionResponse   `json:"water_consumption"`
	IrrigationHistory []IrrigationHistoryResponse `json:"irrigation_history"`
}

type GroupIrrigationCommandRequest struct {
	Action string `json:"action" binding:"required,oneof=START STOP"`
}

type GroupIrrigationCommandResult struct {
	DeviceUuid uuid.UUID `json:"device_uuid"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}
