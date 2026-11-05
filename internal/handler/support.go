package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func supportPagination(ctx *gin.Context) (limit, offset int32) {
	pageID, err := strconv.ParseInt(ctx.DefaultQuery("page_id", "1"), 10, 32)
	if err != nil || pageID < 1 {
		pageID = 1
	}
	pageSize, err := strconv.ParseInt(ctx.DefaultQuery("page_size", "50"), 10, 32)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return int32(pageSize), int32((pageID - 1) * pageSize)
}

func supportMessageError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmptySupportMessage), errors.Is(err, service.ErrSupportMessageTooLong):
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
	case errors.Is(err, sql.ErrNoRows):
		ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
	default:
		serverError(ctx, err)
	}
}

// ListSupportMessages godoc
// @Summary      Listar mensagens do chat de suporte
// @Description  Lista as mensagens do chat de suporte do usuário autenticado, mais recentes primeiro, e marca as respostas do suporte como lidas.
// @Tags         support
// @Produce      json
// @Security     BearerAuth
// @Param        page_id   query int false "Página (default 1)"
// @Param        page_size query int false "Itens por página (default 50, máx 100)"
// @Success      200 {array} dto.SupportMessageResponse
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /support/messages [get]
func (server *Server) ListSupportMessages(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	limit, offset := supportPagination(ctx)

	messages, err := server.SupportService.ListMyMessages(ctx, authPayload.UserID, limit, offset)
	if err != nil {
		supportMessageError(ctx, err)
		return
	}

	response := make([]dto.SupportMessageResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response, dto.NewSupportMessageResponse(m))
	}
	ctx.JSON(http.StatusOK, response)
}

// SendSupportMessage godoc
// @Summary      Enviar mensagem ao suporte
// @Tags         support
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.SendSupportMessageRequest true "Mensagem (até 2000 caracteres)"
// @Success      201 {object} dto.SupportMessageResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /support/messages [post]
func (server *Server) SendSupportMessage(ctx *gin.Context) {
	var req dto.SendSupportMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	message, err := server.SupportService.SendUserMessage(ctx, authPayload.UserID, req.Body)
	if err != nil {
		supportMessageError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewSupportMessageResponse(message))
}

// ListSupportConversations godoc
// @Summary      Listar conversas do suporte (admin)
// @Description  Uma linha por usuário que já escreveu no chat, com a última mensagem e quantas mensagens do usuário ainda não foram lidas.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page_id   query int false "Página (default 1)"
// @Param        page_size query int false "Itens por página (default 50, máx 100)"
// @Success      200 {array} dto.SupportConversationResponse
// @Failure      403 {object} map[string]interface{} "Acesso de admin necessário"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /admin/support/conversations [get]
func (server *Server) ListSupportConversations(ctx *gin.Context) {
	limit, offset := supportPagination(ctx)

	rows, err := server.SupportService.ListConversations(ctx, limit, offset)
	if err != nil {
		supportMessageError(ctx, err)
		return
	}

	response := make([]dto.SupportConversationResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, dto.NewSupportConversationResponse(row))
	}
	ctx.JSON(http.StatusOK, response)
}

// ListSupportConversationMessages godoc
// @Summary      Ler conversa de suporte de um usuário (admin)
// @Description  Lista as mensagens do chat do usuário, mais recentes primeiro, e marca as mensagens dele como lidas.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        uuid      path  string true  "UUID do usuário" Format(uuid)
// @Param        page_id   query int    false "Página (default 1)"
// @Param        page_size query int    false "Itens por página (default 50, máx 100)"
// @Success      200 {array} dto.SupportMessageResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      403 {object} map[string]interface{} "Acesso de admin necessário"
// @Failure      404 {object} map[string]interface{} "Usuário não encontrado"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /admin/support/conversations/{uuid}/messages [get]
func (server *Server) ListSupportConversationMessages(ctx *gin.Context) {
	var uriReq dto.SupportConversationUriRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	limit, offset := supportPagination(ctx)

	messages, err := server.SupportService.ListConversationMessages(ctx, uuid.MustParse(uriReq.UUID), limit, offset)
	if err != nil {
		supportMessageError(ctx, err)
		return
	}

	response := make([]dto.SupportMessageResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response, dto.NewSupportMessageResponse(m))
	}
	ctx.JSON(http.StatusOK, response)
}

// ReplySupportConversation godoc
// @Summary      Responder usuário no chat de suporte (admin)
// @Description  Grava a resposta, entrega em tempo real via WebSocket e envia uma notificação/push ao usuário.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                        true "UUID do usuário" Format(uuid)
// @Param        request body dto.SendSupportMessageRequest true "Resposta (até 2000 caracteres)"
// @Success      201 {object} dto.SupportMessageResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      403 {object} map[string]interface{} "Acesso de admin necessário"
// @Failure      404 {object} map[string]interface{} "Usuário não encontrado"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /admin/support/conversations/{uuid}/messages [post]
func (server *Server) ReplySupportConversation(ctx *gin.Context) {
	var uriReq dto.SupportConversationUriRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.SendSupportMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)
	message, err := server.SupportService.ReplyToUser(ctx, authPayload.UserID, uuid.MustParse(uriReq.UUID), req.Body)
	if err != nil {
		supportMessageError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewSupportMessageResponse(message))
}
