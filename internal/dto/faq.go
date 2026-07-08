package dto

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type GetFaqRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type ListFaqRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type FaqResponse struct {
	Uuid        uuid.UUID `json:"uuid"`
	Question    string    `json:"question"`
	Answer      string    `json:"answer"`
	Category    *string   `json:"category"`
	OrderNumber int32     `json:"order_number"`
}

func NewFaqResponse(faq db.Faq) FaqResponse {
	var category *string
	if faq.Category.Valid {
		category = &faq.Category.String
	}

	return FaqResponse{
		Uuid:        faq.Uuid,
		Question:    faq.Question,
		Answer:      faq.Answer,
		Category:    category,
		OrderNumber: faq.OrderNumber,
	}
}
