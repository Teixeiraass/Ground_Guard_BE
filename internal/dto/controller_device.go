package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type RegisterControllerDeviceRequest struct {
	DeviceUid       string `json:"device_uid" binding:"required"`
	Name            string `json:"name"`
	FirmwareVersion string `json:"firmware_version" binding:"required"`
	FirmwareBuild   string `json:"firmware_build" binding:"required"`
}

type CreateControllerDeviceRequest struct {
	DeviceUid       string `json:"device_uid" binding:"required"`
	Name            string `json:"name" binding:"required"`
	FirmwareVersion string `json:"firmware_version"`
	FirmwareBuild   string `json:"firmware_build" binding:"omitempty"`
	IpAddress       string `json:"ip_address" binding:"omitempty,ip"`
	WifiSsid        string `json:"wifi_ssid" binding:"omitempty"`
}

type ControllerDeviceResponse struct {
	Uuid                uuid.UUID                              `json:"uuid"`
	ControllerDeviceUid string                                 `json:"device_uid"`
	Name                string                                 `json:"name"`
	FirmwareVersion     string                                 `json:"firmware_version"`
	FirmwareBuild       *string                                `json:"firmware_build,omitempty"`
	LastUpdate          *time.Time                             `json:"last_update,omitempty"`
	IpAddress           *string                                `json:"ip_address,omitempty"`
	WifiSsid            *string                                `json:"wifi_ssid,omitempty"`
	LastSeen            *time.Time                             `json:"last_seen"`
	IsOnline            bool                                   `json:"is_online"`
	Configuration       *ControllerDeviceConfigurationResponse `json:"configuration,omitempty"`
}

type ControllerDeviceConfigurationResponse struct {
	Status    string    `json:"status"`
	CommandID uuid.UUID `json:"command_id"`
}

func NewControllerDeviceResponse(controllerDevice db.ControllerDevice) ControllerDeviceResponse {
	var firmwareBuild *string
	if controllerDevice.FirmwareBuild.Valid {
		firmwareBuild = &controllerDevice.FirmwareBuild.String
	}

	var wifiSsid *string
	if controllerDevice.WifiSsid.Valid {
		wifiSsid = &controllerDevice.WifiSsid.String
	}

	var ipAddress *string
	if controllerDevice.IpAddress.Valid {
		ip := controllerDevice.IpAddress.IPNet.IP.String()
		ipAddress = &ip
	}

	var lastSeen *time.Time
	if controllerDevice.LastSeen.Valid {
		lastSeen = &controllerDevice.LastSeen.Time
	}

	var lastUpdate *time.Time
	if controllerDevice.LastUpdate.Valid {
		lastUpdate = &controllerDevice.LastUpdate.Time
	}

	return ControllerDeviceResponse{
		Uuid:                controllerDevice.Uuid,
		ControllerDeviceUid: controllerDevice.ControllerDeviceUid,
		Name:                controllerDevice.Name,
		FirmwareVersion:     controllerDevice.FirmwareVersion,
		FirmwareBuild:       firmwareBuild,
		LastUpdate:          lastUpdate,
		IpAddress:           ipAddress,
		WifiSsid:            wifiSsid,
		LastSeen:            lastSeen,
		IsOnline:            controllerDevice.IsOnline,
	}
}

type GetControllerDeviceRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type GetControllerDeviceByUIDRequest struct {
	UID string `uri:"uid" binding:"required"`
}

type ListControllerDeviceRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type UpdateNameControllerDeviceRequest struct {
	Name string `json:"name" binding:"required"`
}
