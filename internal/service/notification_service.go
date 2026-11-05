package service

import (
	"context"
	"errors"
	"log"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/push"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/google/uuid"
)

const (
	NotificationTypeAutoIrrigationStart  = "AUTO_IRRIGATION_START"
	NotificationTypeLowHumidity          = "LOW_HUMIDITY"
	NotificationTypeBatteryLow           = "BATTERY_LOW"
	NotificationTypePlatformAnnouncement = "PLATFORM_ANNOUNCEMENT"
	NotificationTypeSupportReply         = "SUPPORT_REPLY"

	notificationSourceSystem = "SYSTEM"
	notificationSourceAdmin  = "ADMIN"
)

// preferenceCatalog lists the automatic notification types a user can
// toggle. PLATFORM_ANNOUNCEMENT is intentionally excluded — it's always
// delivered, like an important platform-wide notice, not an opt-in alert.
var preferenceCatalog = []string{
	NotificationTypeAutoIrrigationStart,
	NotificationTypeLowHumidity,
	NotificationTypeBatteryLow,
}

type NotificationService interface {
	CreateAndDeliver(ctx context.Context, userID int64, notifType, title, body string) error
	ListNotifications(ctx context.Context, userID int64, pageID, pageSize int32) ([]db.Notification, error)
	MarkAsRead(ctx context.Context, userID int64, notificationUUID uuid.UUID) (*db.Notification, error)
	MarkAllAsRead(ctx context.Context, userID int64) error
	GetPreferences(ctx context.Context, userID int64) ([]dto.NotificationPreferenceResponse, error)
	UpdatePreference(ctx context.Context, userID int64, notifType string, enabled bool) error
	RegisterPushToken(ctx context.Context, userID int64, token, platform string) error
	DeletePushToken(ctx context.Context, token string) error
}

type notificationService struct {
	store      db.Store
	hub        *websocket.Hub
	pushSender push.Sender
}

func NewNotificationService(store db.Store, hub *websocket.Hub, pushSender push.Sender) NotificationService {
	return &notificationService{store: store, hub: hub, pushSender: pushSender}
}

func (s *notificationService) CreateAndDeliver(ctx context.Context, userID int64, notifType, title, body string) error {
	source := notificationSourceSystem
	if notifType == NotificationTypePlatformAnnouncement {
		source = notificationSourceAdmin
	} else {
		enabled, err := s.isTypeEnabled(ctx, userID, notifType)
		if err != nil {
			return err
		}
		if !enabled {
			return nil
		}
	}

	notification, err := s.store.CreateNotification(ctx, db.CreateNotificationParams{
		UserID: userID,
		Type:   notifType,
		Source: source,
		Title:  title,
		Body:   body,
	})
	if err != nil {
		return err
	}

	s.hub.BroadcastToUser(userID, map[string]any{
		"type":              "notification_created",
		"notification_uuid": notification.Uuid.String(),
		"notification_type": notification.Type,
		"title":             notification.Title,
		"body":              notification.Body,
	})

	return s.sendPush(ctx, userID, title, body)
}

func (s *notificationService) sendPush(ctx context.Context, userID int64, title, body string) error {
	tokens, err := s.store.ListPushTokensByUser(ctx, userID)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	tokenStrings := make([]string, len(tokens))
	for i, t := range tokens {
		tokenStrings[i] = t.Token
	}

	invalidTokens, err := s.pushSender.SendToUser(ctx, tokenStrings, title, body)
	if err != nil {
		// A falha de push não deve impedir a notificação de ter sido criada
		// e entregue por WebSocket — só logamos.
		log.Printf("push send failed for user %d: %v", userID, err)
		return nil
	}
	for _, token := range invalidTokens {
		if err := s.store.DeletePushToken(ctx, token); err != nil {
			log.Printf("failed to remove invalid push token: %v", err)
		}
	}

	return nil
}

func (s *notificationService) isTypeEnabled(ctx context.Context, userID int64, notifType string) (bool, error) {
	prefs, err := s.store.ListNotificationPreferencesByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range prefs {
		if p.Type == notifType {
			return p.Enabled, nil
		}
	}
	return true, nil // sem linha salva = habilitado por padrão
}

func (s *notificationService) ListNotifications(ctx context.Context, userID int64, pageID, pageSize int32) ([]db.Notification, error) {
	return s.store.ListNotificationsByUser(ctx, db.ListNotificationsByUserParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: (pageID - 1) * pageSize,
	})
}

func (s *notificationService) MarkAsRead(ctx context.Context, userID int64, notificationUUID uuid.UUID) (*db.Notification, error) {
	notification, err := s.store.MarkNotificationRead(ctx, db.MarkNotificationReadParams{
		Uuid:   notificationUUID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.store.MarkAllNotificationsRead(ctx, userID)
}

func (s *notificationService) GetPreferences(ctx context.Context, userID int64) ([]dto.NotificationPreferenceResponse, error) {
	prefs, err := s.store.ListNotificationPreferencesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	overrides := make(map[string]bool, len(prefs))
	for _, p := range prefs {
		overrides[p.Type] = p.Enabled
	}

	result := make([]dto.NotificationPreferenceResponse, 0, len(preferenceCatalog))
	for _, t := range preferenceCatalog {
		enabled, ok := overrides[t]
		if !ok {
			enabled = true
		}
		result = append(result, dto.NotificationPreferenceResponse{Type: t, Enabled: enabled})
	}
	return result, nil
}

func (s *notificationService) UpdatePreference(ctx context.Context, userID int64, notifType string, enabled bool) error {
	if !isKnownPreferenceType(notifType) {
		return errors.New("unknown notification type")
	}
	_, err := s.store.UpsertNotificationPreference(ctx, db.UpsertNotificationPreferenceParams{
		UserID:  userID,
		Type:    notifType,
		Enabled: enabled,
	})
	return err
}

func isKnownPreferenceType(t string) bool {
	for _, known := range preferenceCatalog {
		if known == t {
			return true
		}
	}
	return false
}

func (s *notificationService) RegisterPushToken(ctx context.Context, userID int64, token, platform string) error {
	_, err := s.store.UpsertPushToken(ctx, db.UpsertPushTokenParams{
		UserID:   userID,
		Token:    token,
		Platform: platform,
	})
	return err
}

func (s *notificationService) DeletePushToken(ctx context.Context, token string) error {
	return s.store.DeletePushToken(ctx, token)
}
