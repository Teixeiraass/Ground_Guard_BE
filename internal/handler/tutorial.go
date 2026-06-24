package handler

import (
	"database/sql"
	"net/http"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetTutorial godoc
// @Summary      Obter um tutorial
// @Description  Retorna os detalhes de um tutorial específico baseado no seu UUID
// @Tags         tutorials
// @Accept       json
// @Produce      json
// @Param        uuid path string true "UUID do tutorial" Format(uuid)
// @Success      200 {object} dto.TutorialResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de bind ou UUID mal formatado)"
// @Failure      404 {object} map[string]any "Tutorial não encontrado"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /tutorials/{uuid} [get]
func (server *Server) GetTutorial(ctx *gin.Context) {
	var req dto.GetTutorialRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	tutorialUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	tutorial, err := server.store.GetTutorial(ctx, tutorialUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewTutorialResponse(tutorial))
}


// ListTutorials godoc
// @Summary      Listar tutoriais
// @Description  Retorna uma lista paginada de tutoriais
// @Tags         tutorials
// @Accept       json
// @Produce      json
// @Param        page_id query int true "Número da página" minimum(1)
// @Param        page_size query int true "Tamanho da página" minimum(1) maximum(100)
// @Success      200 {array} dto.TutorialResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de paginação)"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /tutorials [get]
func (server *Server) ListTutorials(ctx *gin.Context) {
	var req dto.ListTutorialRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListTutorialsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	tutorials, err := server.store.ListTutorials(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := []dto.TutorialResponse{} 
	
	for _, tutorial := range tutorials {
		rsp = append(rsp, dto.NewTutorialResponse(tutorial))
	}

	ctx.JSON(http.StatusOK, rsp)
}