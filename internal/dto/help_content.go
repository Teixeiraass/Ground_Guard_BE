package dto

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type GetHelpContentRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type ListHelpContentRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type HelpContentResponse struct {
	Uuid        uuid.UUID `json:"uuid"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Category    string    `json:"category"`
	Content     string    `json:"content"`
	ImageUrl    *string   `json:"image_url"`
	Published   bool      `json:"published"`
	OrderNumber int32     `json:"order_number"`
}

func NewHelpContentResponse(helpContent db.HelpContent) HelpContentResponse {
	var imageUrl *string
	if helpContent.ImageUrl.Valid {
		imageUrl = &helpContent.ImageUrl.String
	}

	return HelpContentResponse{
		Uuid:        helpContent.Uuid,
		Title:       helpContent.Title,
		Slug:        helpContent.Slug,
		Category:    helpContent.Category,
		Content:     helpContent.Content,
		ImageUrl:    imageUrl,
		Published:   helpContent.Published,
		OrderNumber: helpContent.OrderNumber,
	}
}
