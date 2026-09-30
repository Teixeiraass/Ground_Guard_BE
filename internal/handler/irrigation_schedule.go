package handler

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateIrrigationSchedule godoc
// @Summary      Criar agendamento de irrigação
// @Description  Cria um agendamento de irrigação para um dispositivo pertencente ao usuário autenticado.
// @Tags         irrigation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateIrrigationScheduleRequest true "Dados do agendamento"
// @Success      201 {object} dto.IrrigationScheduleResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      403 {object} map[string]interface{} "Forbidden"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /irrigation/schedules [post]
func (server *Server) CreateIrrigationSchedule(ctx *gin.Context) {
	var req dto.CreateIrrigationScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if _, err := uuid.Parse(req.DeviceUUID); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	schedule, err := server.IrrigationService.CreateIrrigationSchedule(ctx, req, authPayload.UserID)
	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		case err.Error() == "device doesn't belong to authenticated user":
			ctx.JSON(http.StatusForbidden, errorResponse(err))
		case strings.HasPrefix(err.Error(), "start_time"), strings.HasPrefix(err.Error(), "days_of_week"):
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewIrrigationScheduleResponse(*schedule))
}

// ListIrrigationSchedules godoc
// @Summary      Listar agendamentos de irrigação
// @Description  Retorna os agendamentos de irrigação do usuário autenticado.
// @Tags         irrigation
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} dto.IrrigationScheduleResponse
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /irrigation/schedules [get]
func (server *Server) ListIrrigationSchedules(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	schedules, err := server.IrrigationService.ListIrrigationSchedules(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]dto.IrrigationScheduleResponse, 0, len(schedules))
	for _, schedule := range schedules {
		response = append(response, dto.NewIrrigationScheduleResponse(schedule))
	}
	ctx.JSON(http.StatusOK, response)
}

// GetIrrigationSchedule godoc
// @Summary      Obter agendamento de irrigação
// @Description  Retorna um agendamento pertencente ao usuário autenticado.
// @Tags         irrigation
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do agendamento" format(uuid)
// @Success      200 {object} dto.IrrigationScheduleResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /irrigation/schedules/{uuid} [get]
func (server *Server) GetIrrigationSchedule(ctx *gin.Context) {
	scheduleUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	schedule, err := server.IrrigationService.GetIrrigationSchedule(ctx, scheduleUUID, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, dto.NewIrrigationScheduleResponse(*schedule))
}

// UpdateIrrigationSchedule godoc
// @Summary      Atualizar agendamento de irrigação
// @Description  Atualiza um agendamento pertencente ao usuário autenticado.
// @Tags         irrigation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do agendamento" format(uuid)
// @Param        request body dto.UpdateIrrigationScheduleRequest true "Dados do agendamento"
// @Success      200 {object} dto.IrrigationScheduleResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /irrigation/schedules/{uuid} [put]
func (server *Server) UpdateIrrigationSchedule(ctx *gin.Context) {
	scheduleUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.UpdateIrrigationScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	schedule, err := server.IrrigationService.UpdateIrrigationSchedule(ctx, scheduleUUID, req, authPayload.UserID)
	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		case strings.HasPrefix(err.Error(), "start_time"), strings.HasPrefix(err.Error(), "days_of_week"):
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}
	ctx.JSON(http.StatusOK, dto.NewIrrigationScheduleResponse(*schedule))
}

// DeleteIrrigationSchedule godoc
// @Summary      Excluir agendamento de irrigação
// @Description  Exclui um agendamento pertencente ao usuário autenticado.
// @Tags         irrigation
// @Security     BearerAuth
// @Param        uuid path string true "UUID do agendamento" format(uuid)
// @Success      204
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /irrigation/schedules/{uuid} [delete]
func (server *Server) DeleteIrrigationSchedule(ctx *gin.Context) {
	scheduleUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if err := server.IrrigationService.DeleteIrrigationSchedule(ctx, scheduleUUID, authPayload.UserID); err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.Status(http.StatusNoContent)
}
