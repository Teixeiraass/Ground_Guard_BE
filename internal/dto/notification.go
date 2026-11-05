package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type NotificationResponse struct {
	Uuid      uuid.UUID  `json:"uuid"`
	Type      string     `json:"type"`
	Source    string     `json:"source"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func NewNotificationResponse(notification db.Notification) NotificationResponse {
	var readAt *time.Time
	if notification.ReadAt.Valid {
		readAt = &notification.ReadAt.Time
	}

	return NotificationResponse{
		Uuid:      notification.Uuid,
		Type:      notification.Type,
		Source:    notification.Source,
		Title:     notification.Title,
		Body:      notification.Body,
		ReadAt:    readAt,
		CreatedAt: notification.CreatedAt,
	}
}

type NotificationPreferenceResponse struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

type UpdateNotificationPreferenceRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

type RegisterPushTokenRequest struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=android ios"`
}
