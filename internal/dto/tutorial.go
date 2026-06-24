package dto

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type GetTutorialRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type ListTutorialRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type TutorialResponse struct {
	Uuid        uuid.UUID `json:"uuid"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Content     string    `json:"content"`
	ImageUrl    *string   `json:"image_url"`
	VideoUrl    *string   `json:"video_url"`
	Category    *string   `json:"category"`
	Published   bool      `json:"published"`
	OrderNumber int32     `json:"order_number"`
}

func NewTutorialResponse(tutorial db.Tutorial) TutorialResponse {
	var description *string
	if tutorial.Description.Valid {
		description = &tutorial.Description.String
	}

	var imageUrl *string
	if tutorial.ImageUrl.Valid {
		imageUrl = &tutorial.ImageUrl.String
	}

	var videoUrl *string
	if tutorial.VideoUrl.Valid {
		videoUrl = &tutorial.VideoUrl.String
	}

	var category *string
	if tutorial.Category.Valid {
		category = &tutorial.Category.String
	}

	return TutorialResponse{
		Uuid:        tutorial.Uuid,
		Title:       tutorial.Title,
		Description: description,
		Content:     tutorial.Content,
		ImageUrl:    imageUrl,
		VideoUrl:    videoUrl,
		Category:    category,
		Published:   tutorial.Published,
		OrderNumber: tutorial.OrderNumber,
	}
}
