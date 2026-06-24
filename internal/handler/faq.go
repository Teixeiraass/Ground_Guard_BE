package handler

import (
	"database/sql"
	"net/http"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetFaq godoc
// @Summary      Obter um FAQ
// @Description  Retorna os detalhes de um FAQ específico baseado no seu UUID
// @Tags         faqs
// @Accept       json
// @Produce      json
// @Param        uuid path string true "UUID do FAQ" Format(uuid)
// @Success      200 {object} dto.FaqResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de bind ou UUID mal formatado)"
// @Failure      404 {object} map[string]any "FAQ não encontrado"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /faqs/{uuid} [get]
func (server *Server) GetFaq(ctx *gin.Context) {
	var req dto.GetFaqRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	faqUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	faq, err := server.store.GetFaq(ctx, faqUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewFaqResponse(faq))
}


// ListFaq godoc
// @Summary      Listar FAQs
// @Description  Retorna uma lista paginada de FAQs
// @Tags         faqs
// @Accept       json
// @Produce      json
// @Param        page_id query int true "Número da página" minimum(1)
// @Param        page_size query int true "Tamanho da página" minimum(1) maximum(100)
// @Success      200 {array} dto.FaqResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de paginação)"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /faqs [get]
func (server *Server) ListFaq(ctx *gin.Context) {
	var req dto.ListFaqRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListFaqsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	faqs, err := server.store.ListFaqs(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := []dto.FaqResponse{} 
	
	for _, faq := range faqs {
		rsp = append(rsp, dto.NewFaqResponse(faq))
	}

	ctx.JSON(http.StatusOK, rsp)
}