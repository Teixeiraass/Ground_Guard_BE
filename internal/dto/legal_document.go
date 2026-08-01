package dto

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type GetLegalDocumentRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type ListLegalDocumentRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type LegalDocumentResponse struct {
	Uuid    uuid.UUID `json:"uuid"`
	Type    string    `json:"type"`
	Version string    `json:"version"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
	Active  bool      `json:"active"`
}

func NewLegalDocumentResponse(legalDocument db.LegalDocument) LegalDocumentResponse {
	return LegalDocumentResponse{
		Uuid:    legalDocument.Uuid,
		Type:    legalDocument.Type,
		Version: legalDocument.Version,
		Title:   legalDocument.Title,
		Content: legalDocument.Content,
		Active:  legalDocument.Active,
	}
}
