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

// GetGroupStatistics godoc
// @Summary      Estatísticas do grupo
// @Description  Consumo de água, histórico de irrigação e resumo de umidade agregados de todos os dispositivos do grupo.
// @Tags         device-groups
// @Produce      json
// @Security     BearerAuth
// @Param        uuid        path  string true  "UUID do grupo"
// @Param        date        query string false "Data no formato YYYY-MM-DD"
// @Param        period      query string false "Período de consumo (day, week ou month)"
// @Success      200 {object} dto.DeviceGroupStatisticsResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups/{uuid}/statistics [get]
func (server *Server) GetGroupStatistics(ctx *gin.Context) {
	groupUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	date, err := parseDateQuery(ctx.Query("date"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	stats, err := server.DeviceGroupService.GetGroupStatistics(ctx, authPayload.UserID, groupUUID, date, ctx.Query("period"))
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, stats)
}

// CreateDeviceGroup godoc
// @Summary      Criar grupo de dispositivos
// @Tags         device-groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateDeviceGroupRequest true "Nome do grupo"
// @Success      201 {object} dto.DeviceGroupResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups [post]
func (server *Server) CreateDeviceGroup(ctx *gin.Context) {
	var req dto.CreateDeviceGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	group, err := server.DeviceGroupService.CreateGroup(ctx, authPayload.UserID, req.Name)
	if err != nil {
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewDeviceGroupResponse(*group, nil))
}

// ListDeviceGroups godoc
// @Summary      Listar grupos de dispositivos
// @Tags         device-groups
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} dto.DeviceGroupResponse
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups [get]
func (server *Server) ListDeviceGroups(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	groups, err := server.DeviceGroupService.ListGroups(ctx, authPayload.UserID)
	if err != nil {
		serverError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, groups)
}

// GetDeviceGroup godoc
// @Summary      Obter grupo de dispositivos
// @Tags         device-groups
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do grupo"
// @Success      200 {object} dto.DeviceGroupDetailResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups/{uuid} [get]
func (server *Server) GetDeviceGroup(ctx *gin.Context) {
	groupUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	group, err := server.DeviceGroupService.GetGroup(ctx, authPayload.UserID, groupUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, group)
}

// UpdateDeviceGroupName godoc
// @Summary      Renomear grupo de dispositivos
// @Tags         device-groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do grupo"
// @Param        request body dto.UpdateDeviceGroupNameRequest true "Novo nome"
// @Success      200 {object} dto.DeviceGroupResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups/{uuid} [put]
func (server *Server) UpdateDeviceGroupName(ctx *gin.Context) {
	groupUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.UpdateDeviceGroupNameRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	group, err := server.DeviceGroupService.RenameGroup(ctx, authPayload.UserID, groupUUID, req.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	detail, err := server.DeviceGroupService.GetGroup(ctx, authPayload.UserID, group.Uuid)
	if err != nil {
		serverError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, detail.DeviceGroupResponse)
}

// DeleteDeviceGroup godoc
// @Summary      Excluir grupo de dispositivos
// @Description  Exclui o grupo; os dispositivos e agendamentos dele continuam existindo, só ficam sem grupo.
// @Tags         device-groups
// @Security     BearerAuth
// @Param        uuid path string true "UUID do grupo"
// @Success      204
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups/{uuid} [delete]
func (server *Server) DeleteDeviceGroup(ctx *gin.Context) {
	groupUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if err := server.DeviceGroupService.DeleteGroup(ctx, authPayload.UserID, groupUUID); err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// TriggerGroupIrrigationCommand godoc
// @Summary      Irrigar/parar o grupo inteiro
// @Description  Dispara START ou STOP em todos os dispositivos do grupo. Um device falhando (ex: já irrigando) não cancela os demais.
// @Tags         device-groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do grupo"
// @Param        request body dto.GroupIrrigationCommandRequest true "Ação"
// @Success      200 {array} dto.GroupIrrigationCommandResult
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups/{uuid}/irrigation-commands [post]
func (server *Server) TriggerGroupIrrigationCommand(ctx *gin.Context) {
	groupUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.GroupIrrigationCommandRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	results, err := server.DeviceGroupService.TriggerGroupIrrigation(ctx, authPayload.UserID, groupUUID, req.Action)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, results)
}

// CreateGroupSchedule godoc
// @Summary      Criar agendamento para o grupo inteiro
// @Description  Cria um agendamento de irrigação idêntico para cada dispositivo do grupo.
// @Tags         device-groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do grupo"
// @Param        request body dto.CreateGroupScheduleRequest true "Dados do agendamento"
// @Success      201 {array} dto.IrrigationScheduleResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /device-groups/{uuid}/schedules [post]
func (server *Server) CreateGroupSchedule(ctx *gin.Context) {
	groupUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.CreateGroupScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	schedules, err := server.DeviceGroupService.CreateGroupSchedule(ctx, authPayload.UserID, groupUUID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	response := make([]dto.IrrigationScheduleResponse, 0, len(schedules))
	for _, s := range schedules {
		response = append(response, dto.NewIrrigationScheduleResponse(s))
	}
	ctx.JSON(http.StatusCreated, response)
}
