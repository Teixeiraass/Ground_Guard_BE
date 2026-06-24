package handler

import (
	"database/sql"
	"net/http"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetHelpContent godoc
// @Summary      Obter um conteúdo de ajuda
// @Description  Retorna os detalhes de um conteúdo de ajuda específico baseado no seu UUID
// @Tags         help-contents
// @Accept       json
// @Produce      json
// @Param        uuid path string true "UUID do conteúdo de ajuda" Format(uuid)
// @Success      200 {object} dto.HelpContentResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de bind ou UUID mal formatado)"
// @Failure      404 {object} map[string]any "Conteúdo de ajuda não encontrado"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /help-contents/{uuid} [get]
func (server *Server) GetHelpContent(ctx *gin.Context) {
	var req dto.GetHelpContentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	helpContentUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	helpContent, err := server.store.GetHelpContent(ctx, helpContentUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewHelpContentResponse(helpContent))
}

// ListHelpContent godoc
// @Summary      Listar conteúdos de ajuda
// @Description  Retorna uma lista paginada de conteúdos de ajuda
// @Tags         help-contents
// @Accept       json
// @Produce      json
// @Param        page_id query int true "Número da página" minimum(1)
// @Param        page_size query int true "Tamanho da página" minimum(1) maximum(100)
// @Success      200 {array} dto.HelpContentResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de paginação)"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /help-contents [get]
func (server *Server) ListHelpContent(ctx *gin.Context) {
	var req dto.ListHelpContentRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListHelpContentsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	helpContents, err := server.store.ListHelpContents(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := []dto.HelpContentResponse{} 
	
	for _, helpContent := range helpContents {
		rsp = append(rsp, dto.NewHelpContentResponse(helpContent))
	}

	ctx.JSON(http.StatusOK, rsp)
}