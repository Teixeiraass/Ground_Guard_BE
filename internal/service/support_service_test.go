package service_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeNotifier struct {
	service.NotificationService
	userID    int64
	notifType string
	body      string
	calls     int
}

func (f *fakeNotifier) CreateAndDeliver(_ context.Context, userID int64, notifType, _, body string) error {
	f.calls++
	f.userID = userID
	f.notifType = notifType
	f.body = body
	return nil
}

func newTestSupportService(store db.Store) (service.SupportService, *fakeNotifier) {
	notifier := &fakeNotifier{}
	return service.NewSupportService(store, websocket.NewHub(), notifier), notifier
}

func TestSendUserMessageRejectsBlankBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CreateSupportMessage(gomock.Any(), gomock.Any()).Times(0)

	svc, _ := newTestSupportService(store)
	_, err := svc.SendUserMessage(context.Background(), 1, "   \n ")
	require.ErrorIs(t, err, service.ErrEmptySupportMessage)
}

func TestSendUserMessageRejectsTooLongBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CreateSupportMessage(gomock.Any(), gomock.Any()).Times(0)

	svc, _ := newTestSupportService(store)
	_, err := svc.SendUserMessage(context.Background(), 1, strings.Repeat("á", service.SupportMessageMaxLength+1))
	require.ErrorIs(t, err, service.ErrSupportMessageTooLong)
}

func TestSendUserMessageTrimsAndStoresAsUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		CreateSupportMessage(gomock.Any(), db.CreateSupportMessageParams{
			UserID: 7,
			Sender: service.SupportSenderUser,
			Body:   "Meu sensor não conecta",
		}).
		Times(1).
		Return(db.SupportMessage{UserID: 7, Sender: service.SupportSenderUser, Body: "Meu sensor não conecta"}, nil)

	svc, _ := newTestSupportService(store)
	msg, err := svc.SendUserMessage(context.Background(), 7, "  Meu sensor não conecta  ")
	require.NoError(t, err)
	require.Equal(t, service.SupportSenderUser, msg.Sender)
}

func TestListMyMessagesMarksSupportRepliesRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	gomock.InOrder(
		store.EXPECT().
			MarkSupportMessagesRead(gomock.Any(), db.MarkSupportMessagesReadParams{UserID: 7, Sender: service.SupportSenderSupport}).
			Return(nil),
		store.EXPECT().
			ListSupportMessagesByUser(gomock.Any(), db.ListSupportMessagesByUserParams{UserID: 7, Limit: 50, Offset: 0}).
			Return([]db.SupportMessage{{Body: "oi"}}, nil),
	)

	svc, _ := newTestSupportService(store)
	msgs, err := svc.ListMyMessages(context.Background(), 7, 50, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
}

func TestReplyToUserStoresReplyAndNotifies(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	userUUID := uuid.New()

	store.EXPECT().GetUser(gomock.Any(), userUUID).Return(db.User{ID: 7, Uuid: userUUID}, nil)
	store.EXPECT().
		CreateSupportMessage(gomock.Any(), db.CreateSupportMessageParams{
			UserID:   7,
			Sender:   service.SupportSenderSupport,
			AuthorID: sql.NullInt64{Int64: 1, Valid: true},
			Body:     "Já verificamos, tente de novo",
		}).
		Return(db.SupportMessage{UserID: 7, Sender: service.SupportSenderSupport, Body: "Já verificamos, tente de novo"}, nil)

	svc, notifier := newTestSupportService(store)
	msg, err := svc.ReplyToUser(context.Background(), 1, userUUID, "Já verificamos, tente de novo")
	require.NoError(t, err)
	require.Equal(t, service.SupportSenderSupport, msg.Sender)
	require.Equal(t, 1, notifier.calls)
	require.Equal(t, int64(7), notifier.userID)
	require.Equal(t, service.NotificationTypeSupportReply, notifier.notifType)
}

func TestReplyToUserTruncatesNotificationPreview(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	userUUID := uuid.New()
	longBody := strings.Repeat("a", 500)

	store.EXPECT().GetUser(gomock.Any(), userUUID).Return(db.User{ID: 7}, nil)
	store.EXPECT().CreateSupportMessage(gomock.Any(), gomock.Any()).Return(db.SupportMessage{Body: longBody}, nil)

	svc, notifier := newTestSupportService(store)
	_, err := svc.ReplyToUser(context.Background(), 1, userUUID, longBody)
	require.NoError(t, err)
	require.Less(t, len([]rune(notifier.body)), 200)
}

func TestReplyToUserUnknownUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(db.User{}, sql.ErrNoRows)
	store.EXPECT().CreateSupportMessage(gomock.Any(), gomock.Any()).Times(0)

	svc, notifier := newTestSupportService(store)
	_, err := svc.ReplyToUser(context.Background(), 1, uuid.New(), "oi")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Zero(t, notifier.calls)
}
