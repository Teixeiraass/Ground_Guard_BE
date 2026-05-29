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

// CreateDevice
// @Summary      Cadastrar novo dispositivo
// @Description  Registra um novo dispositivo no sistema e gera seu QR Code de pareamento.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateDeviceRequest  true  "Dados do dispositivo"
// @Success      201      {object}  dto.DeviceResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request"
// @Failure      401      {object}  map[string]interface{} "Unauthorized"
// @Failure      403      {object}  map[string]interface{} "Forbidden (Unique Violation)"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices [post]
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

// GetDevice
// @Summary      Obter detalhes de um dispositivo
// @Description  Retorna as informações de um dispositivo específico cadastrado pelo UUID, garantindo que pertença ao usuário autenticado.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid     path      string  true  "UUID do Dispositivo"
// @Success      200      {object}  dto.DeviceResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request (UUID inválido)"
// @Failure      401      {object}  map[string]interface{} "Unauthorized (Não pertence ao usuário)"
// @Failure      404      {object}  map[string]interface{} "Not Found"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices/{uuid} [get]
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

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(device))
}

// ListDevice
// @Summary      Listar dispositivos do usuário
// @Description  Retorna uma lista paginada de todos os dispositivos associados ao usuário autenticado.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page_id    query     int  true  "Número da página (mínimo 1)"
// @Param        page_size  query     int  true  "Quantidade de itens por página (5 a 10)"
// @Success      200        {array}   dto.DeviceResponse
// @Failure      400        {object}  map[string]interface{} "Bad Request (Parâmetros de paginação inválidos)"
// @Failure      401        {object}  map[string]interface{} "Unauthorized"
// @Failure      500        {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices [get]
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

	rsp := []dto.DeviceResponse{} 
	
	for _, device := range devices {
		rsp = append(rsp, dto.NewDeviceResponse(device))
	}

	ctx.JSON(http.StatusOK, rsp)
}
