package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateDeviceRequest struct {
	DeviceUid string `json:"device_uid" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Status    string `json:"status" binding:"required,oneof=ATIVO INATIVO"`
}

// AssignDeviceToGroupRequest sets or clears the device's group. GroupUuid
// nil/omitted means "remove from whatever group it's in".
type AssignDeviceToGroupRequest struct {
	GroupUuid *string `json:"group_uuid"`
}

type DeviceResponse struct {
	Uuid               uuid.UUID  `json:"uuid"`
	DeviceUid          string     `json:"device_uid"`
	Name               string     `json:"name"`
	LastUpdate         time.Time  `json:"last_update,omitempty"`
	LastSeen           *time.Time `json:"last_seen"`
	IsOnline           bool       `json:"is_online"`
	IsIrrigating       bool       `json:"is_irrigating"`
	Status             string     `json:"status"`
	ControllerDeviceId *int64     `json:"controllerDeviceId,omitempty"`
	SoilMoisture       *int32     `json:"soil_moisture"`
	SensorOk           *bool      `json:"sensor_ok"`
	WifiRssi           *int32     `json:"wifi_rssi"`
	Latitude           *float64   `json:"latitude"`
	Longitude          *float64   `json:"longitude"`
}

func NewDeviceResponse(device db.Device) DeviceResponse {
	var lastSeen *time.Time
	if device.LastSeen.Valid {
		lastSeen = &device.LastSeen.Time
	}

	var controllerDeviceId *int64
	if device.ControllerDeviceID.Valid {
		controllerDeviceId = &device.ControllerDeviceID.Int64
	}

	rsp := DeviceResponse{
		Uuid:               device.Uuid,
		DeviceUid:          device.DeviceUid,
		Name:               device.Name,
		LastUpdate:         device.LastUpdate,
		LastSeen:           lastSeen,
		IsOnline:           device.IsOnline,
		IsIrrigating:       device.IsIrrigating,
		Status:             device.Status,
		ControllerDeviceId: controllerDeviceId,
	}
	if device.SoilMoisture.Valid {
		rsp.SoilMoisture = &device.SoilMoisture.Int32
	}
	if device.SoilSensorOk.Valid {
		rsp.SensorOk = &device.SoilSensorOk.Bool
	}
	if device.WifiRssi.Valid {
		rsp.WifiRssi = &device.WifiRssi.Int32
	}
	if device.Latitude.Valid && device.Longitude.Valid {
		rsp.Latitude = &device.Latitude.Float64
		rsp.Longitude = &device.Longitude.Float64
	}
	return rsp
}

type UpdateDeviceLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude *float64 `json:"longitude" binding:"required,min=-180,max=180"`
}

type GetDeviceRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type GetDeviceByUIDRequest struct {
	UID string `uri:"uid" binding:"required"`
}

type ListDeviceRequest struct {
	PageID         int32  `form:"page_id" binding:"required,min=1"`
	PageSize       int32  `form:"page_size" binding:"required,min=5,max=10"`
	ControllerUUID string `form:"controller_uuid"`
}

type UpdateNameDeviceRequest struct {
	Name string `json:"name" binding:"required"`
}
