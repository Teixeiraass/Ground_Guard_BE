package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type SendSupportMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

type SupportConversationUriRequest struct {
	UUID string `uri:"uuid" binding:"required,uuid"`
}

type SupportMessageResponse struct {
	Uuid      uuid.UUID  `json:"uuid"`
	Sender    string     `json:"sender"`
	Body      string     `json:"body"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func NewSupportMessageResponse(message db.SupportMessage) SupportMessageResponse {
	var readAt *time.Time
	if message.ReadAt.Valid {
		readAt = &message.ReadAt.Time
	}

	return SupportMessageResponse{
		Uuid:      message.Uuid,
		Sender:    message.Sender,
		Body:      message.Body,
		ReadAt:    readAt,
		CreatedAt: message.CreatedAt,
	}
}

type SupportConversationResponse struct {
	UserUuid      uuid.UUID `json:"user_uuid"`
	FullName      string    `json:"full_name"`
	Email         string    `json:"email"`
	LastBody      string    `json:"last_body"`
	LastSender    string    `json:"last_sender"`
	LastMessageAt time.Time `json:"last_message_at"`
	UnreadCount   int64     `json:"unread_count"`
}

func NewSupportConversationResponse(row db.ListSupportConversationsRow) SupportConversationResponse {
	return SupportConversationResponse{
		UserUuid:      row.UserUuid,
		FullName:      row.FullName,
		Email:         row.Email,
		LastBody:      row.LastBody,
		LastSender:    row.LastSender,
		LastMessageAt: row.LastMessageAt,
		UnreadCount:   row.UnreadCount,
	}
}
