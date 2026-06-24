package handler

import (
	"database/sql"
	"net/http"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetLegalDocument godoc
// @Summary      Obter um documento legal
// @Description  Retorna os detalhes de um documento legal específico baseado no seu UUID
// @Tags         legal-documents
// @Accept       json
// @Produce      json
// @Param        uuid path string true "UUID do documento legal" Format(uuid)
// @Success      200 {object} dto.LegalDocumentResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de bind ou UUID mal formatado)"
// @Failure      404 {object} map[string]any "Documento legal não encontrado"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /legal-document/{uuid} [get]
func (server *Server) GetLegalDocument(ctx *gin.Context) {
	var req dto.GetLegalDocumentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	legalDocumentUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	legalDocument, err := server.store.GetLegalDocument(ctx, legalDocumentUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewLegalDocumentResponse(legalDocument))
}

// ListLegalDocument godoc
// @Summary      Listar documentos legais
// @Description  Retorna uma lista paginada de documentos legais
// @Tags         legal-documents
// @Accept       json
// @Produce      json
// @Param        page_id query int true "Número da página" minimum(1)
// @Param        page_size query int true "Tamanho da página" minimum(1) maximum(100)
// @Success      200 {array} dto.LegalDocumentResponse "Sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (Erro de paginação)"
// @Failure      500 {object} map[string]any "Erro interno do servidor"
// @Router       /legal-documents [get]
func (server *Server) ListLegalDocument(ctx *gin.Context) {
	var req dto.ListLegalDocumentRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListLegalDocumentsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	legalDocuments, err := server.store.ListLegalDocuments(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := []dto.LegalDocumentResponse{} 
	
	for _, legalDocument := range legalDocuments {
		rsp = append(rsp, dto.NewLegalDocumentResponse(legalDocument))
	}

	ctx.JSON(http.StatusOK, rsp)
}