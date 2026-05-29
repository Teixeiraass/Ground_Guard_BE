package handler

import (
	"database/sql"
	"errors"
	"net/http"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (server *Server) CreateDevice(ctx *gin.Context) {
	var req dto.CreateDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	qrToken, err := util.GenerateQRToken(12)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	qrFileName, err := util.GenerateQRCodeImage(qrToken)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateDeviceParams{
		DeviceUid:       req.DeviceUid,
		Name:            req.Name,
		FirmwareVersion: req.FirmwareVersion,
		FirmwareBuild:   util.ToNullString(req.FirmwareBuild),
		IpAddress:       util.ToInet(req.IpAddress),
		WifiSsid:        util.ToNullString(req.WifiSsid),
		Status:          req.Status,
		QrToken:         qrToken,
		QrCodeFile:      util.ToNullString(qrFileName),
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

	ctx.JSON(http.StatusCreated, dto.NewDeviceResponse(device))
}

func (server *Server) GetDevice(ctx *gin.Context) {
	var req dto.GetDeviceRequest
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

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if device.UserID.Int64 != authPayload.UserID {
		err := errors.New("device doesn't belong to authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, device)
}

func (server *Server) ListDevice(ctx *gin.Context) {
	var req dto.ListDeviceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	arg := db.ListDevicesParams{
		UserID: sql.NullInt64{
			Int64: authPayload.UserID,
			Valid: true,
		},
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	devices, err := server.store.ListDevices(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, devices)
}
