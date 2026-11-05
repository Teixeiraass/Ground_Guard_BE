package handler

import (
	"net/http"
	"strconv"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListNotifications godoc
// @Summary      Listar notificações
// @Description  Lista as notificações do usuário autenticado, mais recentes primeiro.
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        page_id   query int false "Página (default 1)"
// @Param        page_size query int false "Itens por página (default 20)"
// @Success      200 {array} dto.NotificationResponse
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /notifications [get]
func (server *Server) ListNotifications(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	pageID, err := strconv.ParseInt(ctx.DefaultQuery("page_id", "1"), 10, 32)
	if err != nil || pageID < 1 {
		pageID = 1
	}
	pageSize, err := strconv.ParseInt(ctx.DefaultQuery("page_size", "20"), 10, 32)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	notifications, err := server.NotificationService.ListNotifications(ctx, authPayload.UserID, int32(pageID), int32(pageSize))
	if err != nil {
		serverError(ctx, err)
		return
	}

	response := make([]dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		response = append(response, dto.NewNotificationResponse(n))
	}
	ctx.JSON(http.StatusOK, response)
}

// MarkNotificationRead godoc
// @Summary      Marcar notificação como lida
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da notificação"
// @Success      200 {object} dto.NotificationResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /notifications/{uuid}/read [patch]
func (server *Server) MarkNotificationRead(ctx *gin.Context) {
	notificationUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	notification, err := server.NotificationService.MarkAsRead(ctx, authPayload.UserID, notificationUUID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewNotificationResponse(*notification))
}

// MarkAllNotificationsRead godoc
// @Summary      Marcar todas as notificações como lidas
// @Tags         notifications
// @Security     BearerAuth
// @Success      204
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /notifications/read-all [patch]
func (server *Server) MarkAllNotificationsRead(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if err := server.NotificationService.MarkAllAsRead(ctx, authPayload.UserID); err != nil {
		serverError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// ListNotificationPreferences godoc
// @Summary      Listar preferências de notificação
// @Description  Lista cada tipo de notificação automática e se está habilitado para o usuário autenticado.
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} dto.NotificationPreferenceResponse
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /notification-preferences [get]
func (server *Server) ListNotificationPreferences(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	preferences, err := server.NotificationService.GetPreferences(ctx, authPayload.UserID)
	if err != nil {
		serverError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, preferences)
}

// UpdateNotificationPreference godoc
// @Summary      Atualizar preferência de notificação
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        type path string true "Tipo da notificação"
// @Param        request body dto.UpdateNotificationPreferenceRequest true "Preferência"
// @Success      204
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /notification-preferences/{type} [put]
func (server *Server) UpdateNotificationPreference(ctx *gin.Context) {
	notifType := ctx.Param("type")

	var req dto.UpdateNotificationPreferenceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if err := server.NotificationService.UpdatePreference(ctx, authPayload.UserID, notifType, *req.Enabled); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	ctx.Status(http.StatusNoContent)
}

// RegisterPushToken godoc
// @Summary      Registrar token de push notification
// @Tags         notifications
// @Accept       json
// @Security     BearerAuth
// @Param        request body dto.RegisterPushTokenRequest true "Token"
// @Success      204
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /push-tokens [post]
func (server *Server) RegisterPushToken(ctx *gin.Context) {
	var req dto.RegisterPushTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	if err := server.NotificationService.RegisterPushToken(ctx, authPayload.UserID, req.Token, req.Platform); err != nil {
		serverError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// DeletePushToken godoc
// @Summary      Remover token de push notification
// @Tags         notifications
// @Security     BearerAuth
// @Param        token path string true "Token"
// @Success      204
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /push-tokens/{token} [delete]
func (server *Server) DeletePushToken(ctx *gin.Context) {
	if err := server.NotificationService.DeletePushToken(ctx, ctx.Param("token")); err != nil {
		serverError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
