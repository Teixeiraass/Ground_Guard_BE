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

	return DeviceResponse{
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
