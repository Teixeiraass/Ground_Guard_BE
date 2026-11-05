package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"unicode/utf8"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/google/uuid"
)

const (
	SupportSenderUser    = "USER"
	SupportSenderSupport = "SUPPORT"

	SupportMessageMaxLength = 2000
	supportPushPreviewLen   = 120
)

var (
	ErrEmptySupportMessage   = errors.New("message must not be empty")
	ErrSupportMessageTooLong = errors.New("message is too long")
)

type SupportService interface {
	ListMyMessages(ctx context.Context, userID int64, limit, offset int32) ([]db.SupportMessage, error)
	SendUserMessage(ctx context.Context, userID int64, body string) (db.SupportMessage, error)
	ListConversations(ctx context.Context, limit, offset int32) ([]db.ListSupportConversationsRow, error)
	ListConversationMessages(ctx context.Context, userUUID uuid.UUID, limit, offset int32) ([]db.SupportMessage, error)
	ReplyToUser(ctx context.Context, adminID int64, userUUID uuid.UUID, body string) (db.SupportMessage, error)
}

type supportService struct {
	store         db.Store
	hub           *websocket.Hub
	notifications NotificationService
}

func NewSupportService(store db.Store, hub *websocket.Hub, notifications NotificationService) SupportService {
	return &supportService{store: store, hub: hub, notifications: notifications}
}

func normalizeSupportMessage(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ErrEmptySupportMessage
	}
	if utf8.RuneCountInString(body) > SupportMessageMaxLength {
		return "", ErrSupportMessageTooLong
	}
	return body, nil
}

func (s *supportService) ListMyMessages(ctx context.Context, userID int64, limit, offset int32) ([]db.SupportMessage, error) {
	if err := s.store.MarkSupportMessagesRead(ctx, db.MarkSupportMessagesReadParams{
		UserID: userID,
		Sender: SupportSenderSupport,
	}); err != nil {
		log.Printf("support: failed to mark replies read for user %d: %v", userID, err)
	}

	return s.store.ListSupportMessagesByUser(ctx, db.ListSupportMessagesByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *supportService) SendUserMessage(ctx context.Context, userID int64, body string) (db.SupportMessage, error) {
	body, err := normalizeSupportMessage(body)
	if err != nil {
		return db.SupportMessage{}, err
	}

	return s.store.CreateSupportMessage(ctx, db.CreateSupportMessageParams{
		UserID: userID,
		Sender: SupportSenderUser,
		Body:   body,
	})
}

func (s *supportService) ListConversations(ctx context.Context, limit, offset int32) ([]db.ListSupportConversationsRow, error) {
	return s.store.ListSupportConversations(ctx, db.ListSupportConversationsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *supportService) ListConversationMessages(ctx context.Context, userUUID uuid.UUID, limit, offset int32) ([]db.SupportMessage, error) {
	user, err := s.store.GetUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	if err := s.store.MarkSupportMessagesRead(ctx, db.MarkSupportMessagesReadParams{
		UserID: user.ID,
		Sender: SupportSenderUser,
	}); err != nil {
		log.Printf("support: failed to mark user %d messages read: %v", user.ID, err)
	}

	return s.store.ListSupportMessagesByUser(ctx, db.ListSupportMessagesByUserParams{
		UserID: user.ID,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *supportService) ReplyToUser(ctx context.Context, adminID int64, userUUID uuid.UUID, body string) (db.SupportMessage, error) {
	body, err := normalizeSupportMessage(body)
	if err != nil {
		return db.SupportMessage{}, err
	}

	user, err := s.store.GetUser(ctx, userUUID)
	if err != nil {
		return db.SupportMessage{}, err
	}

	message, err := s.store.CreateSupportMessage(ctx, db.CreateSupportMessageParams{
		UserID:   user.ID,
		Sender:   SupportSenderSupport,
		AuthorID: sql.NullInt64{Int64: adminID, Valid: true},
		Body:     body,
	})
	if err != nil {
		return db.SupportMessage{}, err
	}

	s.hub.BroadcastToUser(user.ID, map[string]any{
		"type":         "support_message",
		"message_uuid": message.Uuid.String(),
		"body":         message.Body,
	})

	if err := s.notifications.CreateAndDeliver(
		ctx,
		user.ID,
		NotificationTypeSupportReply,
		"Nova resposta do suporte",
		truncateRunes(message.Body, supportPushPreviewLen),
	); err != nil {
		log.Printf("support: failed to notify user %d about reply: %v", user.ID, err)
	}

	return message, nil
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}
