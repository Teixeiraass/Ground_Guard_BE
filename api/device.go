package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type createDeviceRequest struct {
	DeviceUid       string `json:"device_uid" binding:"required"`
	Name            string `json:"name" binding:"required"`
	FirmwareVersion string `json:"firmware_version"`
	FirmwareBuild   string `json:"firmware_build" binding:"omitempty"`
	IpAddress       string `json:"ip_address" binding:"omitempty,ip"`
	WifiSsid        string `json:"wifi_ssid" binding:"omitempty"`
	Status          string `json:"status" binding:"required,oneof=ativo inativo"`
}

type deviceResponse struct {
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
	User            userResponse `json:"user"`
}

func newDeviceResponse(device db.Device) deviceResponse {
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

	return deviceResponse{
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
	}
}

func (server *Server) createDevice(ctx *gin.Context) {
	var req createDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateDeviceParams{
		DeviceUid:       req.DeviceUid,
		Name:            req.Name,
		FirmwareVersion: req.FirmwareBuild,
		FirmwareBuild:   util.ToNullString(req.FirmwareBuild),
		IpAddress:       util.ToInet(req.IpAddress),
		WifiSsid:        util.ToNullString(req.WifiSsid),
		Status:          req.Status,
	}

	device, err := server.store.CreateDevice(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := newDeviceResponse(device)
	ctx.JSON(http.StatusCreated, rsp)
}

type getDeviceRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

func (server *Server) getDevice(ctx *gin.Context) {
	var req getDeviceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	deviceUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	device, err := server.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if device.UserID.Int64 != authPayload.UserID {
		err := errors.New("device doesn't belong to authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, device)
}

type listDeviceRequest struct {
	PageID int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

func (server *Server) listDevice(ctx *gin.Context) {
	var req listDeviceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.ListDevicesParams {
		UserID: sql.NullInt64{
			Int64: authPayload.UserID,
			Valid: true,
		},
		Limit: req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	devices, err := server.store.ListDevices(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, devices)
}