package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateDeviceRequest struct {
	DeviceUid       string `json:"device_uid" binding:"required"`
	Name            string `json:"name" binding:"required"`
	FirmwareVersion string `json:"firmware_version"`
	FirmwareBuild   string `json:"firmware_build" binding:"omitempty"`
	IpAddress       string `json:"ip_address" binding:"omitempty,ip"`
	WifiSsid        string `json:"wifi_ssid" binding:"omitempty"`
	Status          string `json:"status" binding:"required,oneof=ATIVO INATIVO"`
}

type DeviceResponse struct {
	Uuid            uuid.UUID    `json:"uuid"`
	DeviceUid       string       `json:"device_uid"`
	Name            string       `json:"name"`
	FirmwareVersion string       `json:"firmware_version"`
	FirmwareBuild   *string      `json:"firmware_build,omitempty"`
	LastUpdate      time.Time    `json:"last_update,omitempty"`
	IpAddress       *string      `json:"ip_address,omitempty"`
	WifiSsid        *string      `json:"wifi_ssid,omitempty"`
	LastSeen        *time.Time   `json:"last_seen"`
	Status          string       `json:"status"`
	User            *int64 		 `json:"user,omitempty"`
}

func NewDeviceResponse(device db.Device) DeviceResponse {
	var firmwareBuild *string
	if device.FirmwareBuild.Valid {
		firmwareBuild = &device.FirmwareBuild.String
	}

	var wifiSsid *string
	if device.WifiSsid.Valid {
		wifiSsid = &device.WifiSsid.String
	}

	var ipAddress *string
	if device.IpAddress.Valid {
		ip := device.IpAddress.IPNet.IP.String()
		ipAddress = &ip
	}

	var lastSeen *time.Time
	if device.LastSeen.Valid {
		lastSeen = &device.LastSeen.Time
	}

	var userId *int64
	if device.UserID.Valid {
		userId = &device.UserID.Int64
	}

	return DeviceResponse{
		Uuid:            device.Uuid,
		DeviceUid:       device.DeviceUid,
		Name:            device.Name,
		FirmwareVersion: device.FirmwareVersion,
		FirmwareBuild:   firmwareBuild,
		LastUpdate:      device.LastUpdate,
		IpAddress:       ipAddress,
		WifiSsid:        wifiSsid,
		LastSeen:        lastSeen,
		Status:          device.Status,
		User: 			 userId,
	}
}

type GetDeviceRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type ListDeviceRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type UpdateNameDeviceRequest struct {
	Name string `json:"name" binding:"required"`
}