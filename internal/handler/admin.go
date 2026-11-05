package handler

import (
	"database/sql"
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListUsers godoc
// @Summary      Listar usuários (admin)
// @Description  Retorna uma lista paginada de usuários, com busca opcional por username/email
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page_id query int true "Número da página" minimum(1)
// @Param        page_size query int true "Tamanho da página" minimum(5) maximum(100)
// @Param        search query string false "Busca por username ou email"
// @Success      200 {array} dto.AdminUserResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida"
// @Failure      403 {object} map[string]any "Acesso de admin necessário"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /admin/users [get]
func (server *Server) ListUsers(ctx *gin.Context) {
	var req dto.ListUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	users, err := server.UserService.ListUsers(ctx, req.Search, req.PageSize, (req.PageID-1)*req.PageSize)
	if err != nil {
		serverError(ctx, err)
		return
	}

	rsp := []dto.AdminUserResponse{}
	for _, user := range users {
		rsp = append(rsp, dto.NewAdminUserResponse(user))
	}

	ctx.JSON(http.StatusOK, rsp)
}

// BanUser godoc
// @Summary      Banir usuário (admin)
// @Description  Suspende o acesso de um usuário à plataforma
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do usuário" Format(uuid)
// @Param        request body dto.BanUserRequest true "Motivo do banimento"
// @Success      200 {object} dto.AdminUserResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida"
// @Failure      403 {object} map[string]any "Acesso de admin necessário"
// @Failure      404 {object} map[string]any "Usuário não encontrado"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /admin/users/{uuid}/ban [put]
func (server *Server) BanUser(ctx *gin.Context) {
	var uriReq dto.BanUserUriRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req dto.BanUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	userUUID, err := uuid.Parse(uriReq.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.UserService.BanUser(ctx, userUUID, req.Reason)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewAdminUserResponse(*user))
}

// UnbanUser godoc
// @Summary      Desbanir usuário (admin)
// @Description  Restaura o acesso de um usuário à plataforma
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do usuário" Format(uuid)
// @Success      200 {object} dto.AdminUserResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida"
// @Failure      403 {object} map[string]any "Acesso de admin necessário"
// @Failure      404 {object} map[string]any "Usuário não encontrado"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /admin/users/{uuid}/unban [put]
func (server *Server) UnbanUser(ctx *gin.Context) {
	var uriReq dto.BanUserUriRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	userUUID, err := uuid.Parse(uriReq.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.UserService.UnbanUser(ctx, userUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewAdminUserResponse(*user))
}
