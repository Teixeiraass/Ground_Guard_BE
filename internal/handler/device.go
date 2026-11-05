package handler

import (
	"database/sql"
	"errors"
	"net/http"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	device, err := server.DeviceService.CreateDevice(ctx, req)
	if err != nil {
		if err.Error() == "unique_violation" {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewDeviceResponse(*device))
}

// RegisterDevice
// @Summary      Registrar ou atualizar dispositivo
// @Description  Verifica a existência do dispositivo pelo UID. Se não existir, realiza o cadastro de um novo dispositivo. Se já existir, atualiza suas informações de rede e sistema (Firmware, IP, Wi-Fi, Status).
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateDeviceRequest  true  "Dados do dispositivo para registro ou atualização"
// @Success      200      {object}  dto.DeviceResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request (JSON inválido)"
// @Failure      403      {object}  map[string]interface{} "Forbidden (Unique Violation)"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices/register [post]
func (server *Server) RegisterDevice(ctx *gin.Context) {
	var req dto.CreateDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	device, err := server.DeviceService.RegisterDevice(ctx, req)
	if err != nil {
		if err.Error() == "unique_violation" {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		if err == sql.ErrNoRows {
			serverError(ctx, err)
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
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

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	device, err := server.DeviceService.GetDevice(ctx, deviceUUID, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		if err.Error() == "device doesn't belong to authenticated user" {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}

// ListDevice
// @Summary      Listar dispositivos do usuário
// @Description  Retorna uma lista paginada de dispositivos associados ao usuário autenticado. Pode filtrar por controller_uuid.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page_id    query     int  true  "Número da página (mínimo 1)"
// @Param        page_size  query     int  true  "Quantidade de itens por página (5 a 10)"
// @Param        controller_uuid query string false "UUID da controladora"
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
	var devices []db.Device
	var err error
	if req.ControllerUUID == "" {
		devices, err = server.DeviceService.ListDevices(ctx, authPayload.UserID, req.PageSize, (req.PageID-1)*req.PageSize)
	} else {
		controllerUUID, parseErr := uuid.Parse(req.ControllerUUID)
		if parseErr != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(parseErr))
			return
		}
		devices, err = server.DeviceService.ListDevicesByController(ctx, controllerUUID, authPayload.UserID, req.PageSize, (req.PageID-1)*req.PageSize)
	}
	if err != nil {
		serverError(ctx, err)
		return
	}

	rsp := []dto.DeviceResponse{}

	for _, device := range devices {
		rsp = append(rsp, dto.NewDeviceResponse(device))
	}

	ctx.JSON(http.StatusOK, rsp)
}

// GetDeviceByUID
// @Summary      Obter detalhes de um dispositivo pelo UID
// @Description  Retorna as informações de um dispositivo específico cadastrado utilizando o seu UID (Unique Identifier).
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uid      path      string  true  "UID do Dispositivo"
// @Success      200      {object}  dto.DeviceResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request (UID inválido)"
// @Failure      404      {object}  map[string]interface{} "Not Found (Dispositivo não encontrado)"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices/uid/{uid} [get]
func (server *Server) GetDeviceByUID(ctx *gin.Context) {
	var req dto.GetDeviceByUIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	device, err := server.DeviceService.GetDeviceByUID(ctx, req.UID)
	if err == nil {
		// The response includes the sensor's map position: owner only.
		authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
		owner, ownerErr := server.store.GetDeviceUserIDByUID(ctx, device.DeviceUid)
		if ownerErr != nil || !owner.Valid || owner.Int64 != authPayload.UserID {
			err = sql.ErrNoRows
		}
	}
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}

// LinkDeviceToUserByQrToken
// @Summary      Vincular dispositivo ao usuário
// @Description  Associa um dispositivo à conta do usuário autenticado lendo o QR Token passado na URL.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        qr_token   path      string  true  "Token do QR Code do dispositivo"
// @Param        controller_uuid query string true  "UUID da controladora proprietária"
// @Success      200        {object}  dto.DeviceResponse
// @Failure      401        {object}  map[string]interface{} "Unauthorized (Usuário não autenticado)"
// @Failure      404        {object}  map[string]interface{} "Not Found (QR Token inválido)"
// @Failure      500        {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices/link/{qr_token} [put]
func (server *Server) LinkDeviceToUserByQrToken(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	qrToken := ctx.Param("qr_token")
	controllerDeviceUUID, err := uuid.Parse(ctx.Query("controller_uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	device, err := server.DeviceService.LinkDeviceToControllerDeviceByQrToken(ctx, qrToken, controllerDeviceUUID, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}

// UnlinkDeviceFromUser
// @Summary      Desvincular dispositivo do usuário
// @Description  Remove a associação de um dispositivo com a conta do usuário autenticado.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid   path      string  true  "UUID do Dispositivo"
// @Success      200    {object}  dto.DeviceResponse
// @Failure      400    {object}  map[string]interface{} "Bad Request (UUID inválido)"
// @Failure      401    {object}  map[string]interface{} "Unauthorized (Usuário não autenticado)"
// @Failure      404    {object}  map[string]interface{} "Not Found (Dispositivo não encontrado ou não pertence a você)"
// @Failure      500    {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices/unlink/{uuid} [put]
func (server *Server) UnlinkDeviceFromUser(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	uuidStr := ctx.Param("uuid")

	deviceUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	device, err := server.DeviceService.UnlinkDeviceFromControllerDevice(ctx, deviceUUID, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}

// UpdateNameDevice
// @Summary      Atualizar nome do dispositivo
// @Description  Modifica o nome de um dispositivo cadastrado pelo UUID, garantindo que pertença ao usuário autenticado.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid   path      string  true  "UUID do Dispositivo"
// @Param        request  body      dto.UpdateNameDeviceRequest  true  "Novo nome do dispositivo"
// @Success      200      {object}  dto.DeviceResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request (UUID inválido)"
// @Failure      401      {object}  map[string]interface{} "Unauthorized (Usuário não autenticado)"
// @Failure      404      {object}  map[string]interface{} "Not Found (Dispositivo não encontrado ou não pertence a você)"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices/name/{uuid} [put]
func (server *Server) UpdateNameDevice(ctx *gin.Context) {
	var req dto.UpdateNameDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	uuidStr := ctx.Param("uuid")

	deviceUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	device, err := server.DeviceService.UpdateNameDevice(ctx, authPayload.UserID, deviceUUID, req.Name)
	if err != nil {
		deviceOwnershipError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}

func deviceOwnershipError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		ctx.JSON(http.StatusNotFound, errorResponse(errors.New("device not found")))
	case errors.Is(err, service.ErrDeviceNotOwned):
		ctx.JSON(http.StatusForbidden, errorResponse(err))
	default:
		serverError(ctx, err)
	}
}

// UpdateDeviceLocation
// @Summary      Posicionar sensor no mapa
// @Description  Grava a latitude/longitude do sensor para exibi-lo na aba Mapa. Só o dono do sensor pode alterar.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid     path  string                           true "UUID do Dispositivo"
// @Param        request  body  dto.UpdateDeviceLocationRequest  true "Coordenadas"
// @Success      200 {object} dto.DeviceResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      403 {object} map[string]interface{} "Forbidden"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /devices/{uuid}/location [put]
func (server *Server) UpdateDeviceLocation(ctx *gin.Context) {
	deviceUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.UpdateDeviceLocationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	device, err := server.DeviceService.UpdateDeviceLocation(ctx, authPayload.UserID, deviceUUID, req.Latitude, req.Longitude)
	if err != nil {
		deviceOwnershipError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}

// ClearDeviceLocation
// @Summary      Remover sensor do mapa
// @Tags         devices
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do Dispositivo"
// @Success      200 {object} dto.DeviceResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      403 {object} map[string]interface{} "Forbidden"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /devices/{uuid}/location [delete]
func (server *Server) ClearDeviceLocation(ctx *gin.Context) {
	deviceUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	device, err := server.DeviceService.UpdateDeviceLocation(ctx, authPayload.UserID, deviceUUID, nil, nil)
	if err != nil {
		deviceOwnershipError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}

// AssignDeviceToGroup
// @Summary      Atribuir dispositivo a um grupo
// @Description  Move o dispositivo para o grupo informado, ou remove de qualquer grupo se group_uuid for omitido/nulo.
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid   path      string  true  "UUID do Dispositivo"
// @Param        request  body      dto.AssignDeviceToGroupRequest  true  "Grupo de destino"
// @Success      200      {object}  dto.DeviceResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request"
// @Failure      403      {object}  map[string]interface{} "Forbidden"
// @Failure      404      {object}  map[string]interface{} "Not Found"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /devices/{uuid}/group [put]
func (server *Server) AssignDeviceToGroup(ctx *gin.Context) {
	deviceUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.AssignDeviceToGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var groupUUID *uuid.UUID
	if req.GroupUuid != nil {
		parsed, err := uuid.Parse(*req.GroupUuid)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}
		groupUUID = &parsed
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	device, err := server.DeviceService.AssignDeviceToGroup(ctx, authPayload.UserID, deviceUUID, groupUUID)
	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		case err.Error() == "device doesn't belong to authenticated user":
			ctx.JSON(http.StatusForbidden, errorResponse(err))
		default:
			serverError(ctx, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, dto.NewDeviceResponse(*device))
}
