package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (server *Server) CreateControllerDevice(ctx *gin.Context) {
	var req dto.CreateControllerDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	controllerDevice, err := server.ControllerDeviceService.CreateControllerDevice(ctx, req, authPayload.UserID)
	if err != nil {
		if err.Error() == "unique_violation" {
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusCreated, dto.NewControllerDeviceResponse(*controllerDevice))
}

// RegisterControllerDevice registers hardware metadata during provisioning.
// @Summary Registrar controladora via hardware
// @Tags controller-devices
// @Accept json
// @Produce json
// @Param request body dto.RegisterControllerDeviceRequest true "Dados da controladora"
// @Success 200 {object} dto.ControllerDeviceResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /controller-devices/register [post]
func (server *Server) RegisterControllerDevice(ctx *gin.Context) {
	var req dto.RegisterControllerDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	controllerDevice, err := server.ControllerDeviceService.RegisterControllerDevice(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidControllerDeviceUID) {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}
		if errors.Is(err, service.ErrControllerDeviceConflict) {
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, dto.NewControllerDeviceResponse(*controllerDevice))
}

// GetControllerDevice returns one controller owned by the authenticated user.
// @Summary Obter controladora
// @Tags controller-devices
// @Security BearerAuth
// @Param uuid path string true "UUID da controladora"
// @Success 200 {object} dto.ControllerDeviceResponse
// @Router /controller-devices/{uuid} [get]
func (server *Server) GetControllerDevice(ctx *gin.Context) {
	deviceUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	controllerDevice, err := server.ControllerDeviceService.GetControllerDevice(ctx, deviceUUID, authPayload.UserID)
	if err != nil {
		controllerDeviceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewControllerDeviceResponse(*controllerDevice))
}

// GetControllerDeviceByUID returns a controller by its hardware UID.
// @Summary Obter controladora por UID
// @Tags controller-devices
// @Security BearerAuth
// @Param uid path string true "UID da controladora"
// @Success 200 {object} dto.ControllerDeviceResponse
// @Router /controller-devices/uid/{uid} [get]
func (server *Server) GetControllerDeviceByUID(ctx *gin.Context) {
	controllerDevice, err := server.ControllerDeviceService.GetControllerDeviceByUID(ctx, ctx.Param("uid"))
	if err != nil {
		controllerDeviceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewControllerDeviceResponse(*controllerDevice))
}

// ClaimControllerDeviceByUID associates an unowned provisioned controller with the user.
// @Summary Reivindicar controladora
// @Tags controller-devices
// @Security BearerAuth
// @Param uid path string true "UID da controladora"
// @Success 200 {object} dto.ControllerDeviceResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /controller-devices/claim/{uid} [put]
func (server *Server) ClaimControllerDeviceByUID(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	controllerDevice, configuration, err := server.ControllerDeviceService.ClaimControllerDeviceByUID(ctx, ctx.Param("uid"), authPayload.UserID)
	if err != nil {
		controllerDeviceError(ctx, err)
		return
	}
	response := dto.NewControllerDeviceResponse(*controllerDevice)
	response.Configuration = configuration
	ctx.JSON(http.StatusOK, response)
}

// ListControllerDevices lists every controller owned by the authenticated user.
// @Summary Listar controladoras
// @Tags controller-devices
// @Security BearerAuth
// @Param page_id query int true "Página"
// @Param page_size query int true "Itens por página"
// @Success 200 {array} dto.ControllerDeviceResponse
// @Router /controller-devices [get]
func (server *Server) ListControllerDevices(ctx *gin.Context) {
	var req dto.ListControllerDeviceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	devices, err := server.ControllerDeviceService.ListControllerDevices(ctx, authPayload.UserID, req.PageSize, (req.PageID-1)*req.PageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	response := make([]dto.ControllerDeviceResponse, 0, len(devices))
	for _, device := range devices {
		response = append(response, dto.NewControllerDeviceResponse(device))
	}
	ctx.JSON(http.StatusOK, response)
}

// UnlinkControllerDeviceFromUser removes ownership from a controller.
// @Summary Desvincular controladora
// @Tags controller-devices
// @Security BearerAuth
// @Param uuid path string true "UUID da controladora"
// @Success 200 {object} dto.ControllerDeviceResponse
// @Router /controller-devices/unlink/{uuid} [put]
func (server *Server) UnlinkControllerDeviceFromUser(ctx *gin.Context) {
	deviceUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	controllerDevice, err := server.ControllerDeviceService.UnlinkControllerDeviceFromUser(ctx, deviceUUID, authPayload.UserID)
	if err != nil {
		controllerDeviceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewControllerDeviceResponse(*controllerDevice))
}

// UpdateNameControllerDevice renames an owned controller.
// @Summary Renomear controladora
// @Tags controller-devices
// @Security BearerAuth
// @Param uuid path string true "UUID da controladora"
// @Param request body dto.UpdateNameControllerDeviceRequest true "Novo nome"
// @Success 200 {object} dto.ControllerDeviceResponse
// @Router /controller-devices/name/{uuid} [put]
func (server *Server) UpdateNameControllerDevice(ctx *gin.Context) {
	var req dto.UpdateNameControllerDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	deviceUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	controllerDevice, err := server.ControllerDeviceService.UpdateNameControllerDevice(ctx, deviceUUID, req.Name)
	if err != nil {
		controllerDeviceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewControllerDeviceResponse(*controllerDevice))
}

func controllerDeviceError(ctx *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidControllerDeviceUID) {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if errors.Is(err, service.ErrControllerDeviceConflict) {
		ctx.JSON(http.StatusConflict, errorResponse(err))
		return
	}
	if err == sql.ErrNoRows {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}
	if err.Error() == "device doesn't belong to authenticated user" {
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusInternalServerError, errorResponse(err))
}
