package service_test

import (
	"context"
	"testing"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type fakeSender struct {
	invalidTokens []string
	calls         int
}

func (f *fakeSender) SendToUser(_ context.Context, _ []string, _, _ string) ([]string, error) {
	f.calls++
	return f.invalidTokens, nil
}

func TestCreateAndDeliverSkipsWhenPreferenceDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	sender := &fakeSender{}

	store.EXPECT().ListNotificationPreferencesByUser(gomock.Any(), int64(42)).Return(
		[]db.NotificationPreference{{UserID: 42, Type: service.NotificationTypeLowHumidity, Enabled: false}}, nil,
	)
	// Nenhum CreateNotification esperado: o mock falha o teste se for chamado.

	notificationService := service.NewNotificationService(store, websocket.NewHub(), sender)
	err := notificationService.CreateAndDeliver(context.Background(), 42, service.NotificationTypeLowHumidity, "Umidade baixa", "...")
	require.NoError(t, err)
	require.Equal(t, 0, sender.calls)
}

func TestCreateAndDeliverPlatformAnnouncementIgnoresPreference(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	sender := &fakeSender{}

	// Nenhuma chamada a ListNotificationPreferencesByUser esperada: anúncios
	// da plataforma são sempre entregues, não passam pela preferência do tipo.
	store.EXPECT().CreateNotification(gomock.Any(), gomock.Any()).Return(db.Notification{
		UserID: 42, Type: service.NotificationTypePlatformAnnouncement, Source: "ADMIN",
	}, nil)
	store.EXPECT().ListPushTokensByUser(gomock.Any(), int64(42)).Return(nil, nil)

	notificationService := service.NewNotificationService(store, websocket.NewHub(), sender)
	err := notificationService.CreateAndDeliver(context.Background(), 42, service.NotificationTypePlatformAnnouncement, "Aviso", "...")
	require.NoError(t, err)
}

func TestCreateAndDeliverRemovesInvalidPushToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	sender := &fakeSender{invalidTokens: []string{"dead-token"}}

	store.EXPECT().ListNotificationPreferencesByUser(gomock.Any(), int64(7)).Return(nil, nil)
	store.EXPECT().CreateNotification(gomock.Any(), gomock.Any()).Return(db.Notification{
		UserID: 7, Type: service.NotificationTypeAutoIrrigationStart, Source: "SYSTEM",
	}, nil)
	store.EXPECT().ListPushTokensByUser(gomock.Any(), int64(7)).Return(
		[]db.PushToken{{UserID: 7, Token: "dead-token", Platform: "android"}}, nil,
	)
	store.EXPECT().DeletePushToken(gomock.Any(), "dead-token").Return(nil)

	notificationService := service.NewNotificationService(store, websocket.NewHub(), sender)
	err := notificationService.CreateAndDeliver(context.Background(), 7, service.NotificationTypeAutoIrrigationStart, "Irrigação", "...")
	require.NoError(t, err)
	require.Equal(t, 1, sender.calls)
}
