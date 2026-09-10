package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
)

// GetStatistics
// @Summary      Obter estatísticas do produto
// @Description  Retorna um resumo operacional com histórico de irrigação, consumo de água e métricas de dispositivos.
// @Tags         statistics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        date        query     string  false  "Data no formato YYYY-MM-DD"
// @Param        period      query     string  false  "Período de consumo (day, week ou month)"
// @Param        device_uuid query     string  false  "UUID do dispositivo"
// @Success      200         {object}  dto.StatisticsResponse
// @Failure      400         {object}  map[string]interface{} "Bad Request"
// @Failure      401         {object}  map[string]interface{} "Unauthorized"
// @Failure      403         {object}  map[string]interface{} "Forbidden"
// @Failure      500         {object}  map[string]interface{} "Internal Server Error"
// @Router       /statistics [get]
func (server *Server) GetStatistics(ctx *gin.Context) {
	var req dto.GetStatisticsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	date, err := parseDateQuery(req.Date)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	deviceUUID, err := parseUUIDQuery(req.DeviceUUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	history, err := server.IrrigationService.ListIrrigationHistory(ctx, authPayload.UserID, date, deviceUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		if err.Error() == "device doesn't belong to authenticated user" {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	consumption, err := server.IrrigationService.GetWaterConsumption(ctx, authPayload.UserID, req.Period, deviceUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		if err.Error() == "device doesn't belong to authenticated user" {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	devices, err := server.loadAllDevicesForStatistics(ctx, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var totalDevices int32
	var onlineDevices int32
	for _, device := range devices {
		totalDevices++
		if device.IsOnline {
			onlineDevices++
		}
	}

	rsp := dto.StatisticsResponse{
		GeneratedAt: time.Now().In(time.Local),
		Date:        date.Format("2006-01-02"),
		Period:      consumption.Period,
		DeviceUUID:  deviceUUID,
		DeviceStatistics: dto.DeviceStatisticsResponse{
			Total:  totalDevices,
			Online: onlineDevices,
		},
		WaterConsumption: consumption,
	}

	rsp.IrrigationHistory = make([]dto.IrrigationHistoryResponse, 0, len(history))
	for _, action := range history {
		rsp.IrrigationHistory = append(rsp.IrrigationHistory, dto.NewIrrigationHistoryResponse(action))
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) loadAllDevicesForStatistics(ctx *gin.Context, userID int64) ([]dto.DeviceResponse, error) {
	const pageSize int32 = 100

	var devices []dto.DeviceResponse
	var offset int32

	for {
		items, err := server.DeviceService.ListDevices(ctx, userID, pageSize, offset)
		if err != nil {
			return nil, err
		}

		for _, device := range items {
			devices = append(devices, dto.NewDeviceResponse(device))
		}

		if int32(len(items)) < pageSize {
			break
		}

		offset += pageSize
	}

	return devices, nil
}
