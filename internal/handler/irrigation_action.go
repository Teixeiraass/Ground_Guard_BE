package handler

import (
	"database/sql"
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateIrrigationCommand
// @Summary      Criar comando de irrigação
// @Description  Registra um comando pendente para ser enviado ao ESP32 via MQTT.
// @Tags         irrigation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateIrrigationCommandRequest  true  "Dados do comando"
// @Success      201      {object}  dto.CreateIrrigationCommandResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request"
// @Failure      401      {object}  map[string]interface{} "Unauthorized"
// @Failure      403      {object}  map[string]interface{} "Forbidden"
// @Failure      404      {object}  map[string]interface{} "Not Found"
// @Failure      409      {object}  map[string]interface{} "Conflict"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /irrigation/commands [post]
func (server *Server) CreateIrrigationCommand(ctx *gin.Context) {
	var req dto.CreateIrrigationCommandRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	command, err := server.IrrigationService.CreateIrrigationCommand(ctx, req, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		if err.Error() == "device doesn't belong to authenticated user" {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		if err.Error() == "device is inactive, cannot send command" ||
			err.Error() == "device already has a pending command" ||
			err.Error() == "device is already irrigating" ||
			err.Error() == "device is not irrigating" {
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewIrrigationCommandResponse(*command))
}

func (server *Server) UpdateIrrigationCommand(ctx *gin.Context) {

}

func (server *Server) ListIrrigationHistory(ctx *gin.Context) {}

// GetIrrigationCommands
// @Summary      Obter comando de irrigação
// @Description  Retorna os detalhes de um comando de irrigação específico através do seu UUID.
// @Tags         irrigation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid  path      string  true  "UUID do comando de irrigação"
// @Success      200   {object}  dto.CreateIrrigationCommandResponse
// @Failure      400   {object}  map[string]interface{} "Bad Request"
// @Failure      401   {object}  map[string]interface{} "Unauthorized"
// @Failure      404   {object}  map[string]interface{} "Not Found"
// @Failure      500   {object}  map[string]interface{} "Internal Server Error"
// @Router       /irrigation/command/{uuid} [get]
func (server *Server) GetIrrigationCommands(ctx *gin.Context) {
	var req dto.GetIrrigationCommandRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	irrigationCommandUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	irrigationCommand, err := server.IrrigationService.GetIrrigationCommand(ctx, irrigationCommandUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewIrrigationCommandResponse(*irrigationCommand))
}

func (server *Server) GetIrrigationHistory(ctx *gin.Context) {}

func (server *Server) GetIrrigationStatus(ctx *gin.Context) {}
