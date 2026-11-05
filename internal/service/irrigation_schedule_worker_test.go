package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	mockmqtt "github.com/Teixeiraass/ground_guard_be/mqtt/mock"
	"github.com/Teixeiraass/ground_guard_be/push"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// newTestNotificationService builds a NotificationService safe for tests
// that don't exercise notification delivery: a hub with no running Run()
// loop (BroadcastToUser just iterates an empty client map) and a no-op push
// sender.
func newTestNotificationService(store db.Store) service.NotificationService {
	return service.NewNotificationService(store, websocket.NewHub(), push.NewNoopSender())
}

func TestProcessDueIrrigationSchedulesFiresMatchingScheduleOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	now := time.Now().In(time.Local)
	device := db.Device{
		ID:        10,
		Uuid:      uuid.New(),
		DeviceUid: "AABBCCDDEEFF",
		Status:    "ativo",
	}
	schedule := db.IrrigationSchedule{
		ID:              1,
		DeviceID:        device.ID,
		UserID:          42,
		Enabled:         true,
		StartTime:       time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC),
		DurationSeconds: 60,
		DaysOfWeek:      weekdayList(now.Weekday()),
	}
	command := db.IrrigationCommand{
		Uuid:   uuid.New(),
		Action: "START",
	}

	store.EXPECT().ListEnabledIrrigationSchedules(gomock.Any()).Return([]db.IrrigationSchedule{schedule}, nil)
	store.EXPECT().GetDeviceByID(gomock.Any(), device.ID).Return(device, nil)
	store.EXPECT().ExistsPendingIrrigationCommand(gomock.Any(), device.ID).Return(false, nil)
	store.EXPECT().ExistsActiveIrrigationAction(gomock.Any(), device.ID).Return(false, nil)
	store.EXPECT().CreateIrrigationCommand(gomock.Any(), gomock.Eq(db.CreateIrrigationCommandParams{
		DeviceID:        device.ID,
		UserID:          schedule.UserID,
		Action:          "START",
		DurationSeconds: sql.NullInt32{Int32: schedule.DurationSeconds, Valid: true},
		ScheduleID:      sql.NullInt64{Int64: schedule.ID, Valid: true},
	})).Return(command, nil)
	mqttClient.EXPECT().TopicPrefix().Return("ground-guard")
	mqttClient.EXPECT().Publish("ground-guard/devices/AABBCCDDEEFF/commands", gomock.Any()).Return(nil)
	store.EXPECT().UpdateIrrigationScheduleLastTriggered(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.UpdateIrrigationScheduleLastTriggeredParams) error {
			require.Equal(t, schedule.ID, arg.ID)
			require.True(t, arg.LastTriggeredAt.Valid)
			return nil
		})

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	require.NoError(t, irrigationService.ProcessDueIrrigationSchedules(context.Background()))
}

func TestProcessDueIrrigationSchedulesSkipsAlreadyTriggeredToday(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	now := time.Now().In(time.Local)
	schedule := db.IrrigationSchedule{
		ID:              1,
		DeviceID:        10,
		UserID:          42,
		Enabled:         true,
		StartTime:       time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC),
		DurationSeconds: 60,
		DaysOfWeek:      weekdayList(now.Weekday()),
		LastTriggeredAt: sql.NullTime{Time: now, Valid: true},
	}

	// No CreateIrrigationCommand/UpdateIrrigationScheduleLastTriggered expectations:
	// the mock fails the test if the worker tries to fire an already-triggered schedule again.
	store.EXPECT().ListEnabledIrrigationSchedules(gomock.Any()).Return([]db.IrrigationSchedule{schedule}, nil)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	require.NoError(t, irrigationService.ProcessDueIrrigationSchedules(context.Background()))
}

func TestProcessScheduledIrrigationCompletionStopsExpiredDurationAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	device := db.Device{
		ID:        10,
		Uuid:      uuid.New(),
		DeviceUid: "AABBCCDDEEFF",
		Status:    "ativo",
	}
	row := db.ListActiveScheduledIrrigationActionsRow{
		ID:                      1,
		DeviceID:                device.ID,
		UserID:                  42,
		StartedAt:               time.Now().Add(-2 * time.Minute),
		ScheduleID:              5,
		IrrigationMode:          "DURATION",
		ScheduleDurationSeconds: 60,
		DeviceUuid:              device.Uuid,
	}
	command := db.IrrigationCommand{Uuid: uuid.New(), Action: "STOP"}

	store.EXPECT().ListActiveScheduledIrrigationActions(gomock.Any()).Return([]db.ListActiveScheduledIrrigationActionsRow{row}, nil)
	store.EXPECT().GetDevice(gomock.Any(), device.Uuid).Return(device, nil)
	store.EXPECT().ExistsPendingIrrigationCommand(gomock.Any(), device.ID).Return(false, nil)
	store.EXPECT().ExistsActiveIrrigationAction(gomock.Any(), device.ID).Return(true, nil)
	store.EXPECT().CreateIrrigationCommand(gomock.Any(), gomock.Eq(db.CreateIrrigationCommandParams{
		DeviceID:   device.ID,
		UserID:     row.UserID,
		Action:     "STOP",
		ScheduleID: sql.NullInt64{Int64: row.ScheduleID, Valid: true},
	})).Return(command, nil)
	mqttClient.EXPECT().TopicPrefix().Return("ground-guard")
	mqttClient.EXPECT().Publish("ground-guard/devices/AABBCCDDEEFF/commands", gomock.Any()).Return(nil)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	require.NoError(t, irrigationService.ProcessScheduledIrrigationCompletion(context.Background()))
}

func TestProcessScheduledIrrigationCompletionStopsOnHumidityThresholdBeforeCap(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	device := db.Device{
		ID:        10,
		Uuid:      uuid.New(),
		DeviceUid: "AABBCCDDEEFF",
		Status:    "ativo",
	}
	row := db.ListActiveScheduledIrrigationActionsRow{
		ID:                      1,
		DeviceID:                device.ID,
		UserID:                  42,
		StartedAt:               time.Now(), // just started, nowhere near the safety-cap duration
		ScheduleID:              5,
		IrrigationMode:          "HUMIDITY",
		MoistureThreshold:       sql.NullInt32{Int32: 50, Valid: true},
		ScheduleDurationSeconds: 600,
		DeviceUuid:              device.Uuid,
		SoilMoisture:            sql.NullInt32{Int32: 60, Valid: true}, // already above threshold
	}
	command := db.IrrigationCommand{Uuid: uuid.New(), Action: "STOP"}

	store.EXPECT().ListActiveScheduledIrrigationActions(gomock.Any()).Return([]db.ListActiveScheduledIrrigationActionsRow{row}, nil)
	store.EXPECT().GetDevice(gomock.Any(), device.Uuid).Return(device, nil)
	store.EXPECT().ExistsPendingIrrigationCommand(gomock.Any(), device.ID).Return(false, nil)
	store.EXPECT().ExistsActiveIrrigationAction(gomock.Any(), device.ID).Return(true, nil)
	store.EXPECT().CreateIrrigationCommand(gomock.Any(), gomock.Eq(db.CreateIrrigationCommandParams{
		DeviceID:   device.ID,
		UserID:     row.UserID,
		Action:     "STOP",
		ScheduleID: sql.NullInt64{Int64: row.ScheduleID, Valid: true},
	})).Return(command, nil)
	mqttClient.EXPECT().TopicPrefix().Return("ground-guard")
	mqttClient.EXPECT().Publish("ground-guard/devices/AABBCCDDEEFF/commands", gomock.Any()).Return(nil)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	require.NoError(t, irrigationService.ProcessScheduledIrrigationCompletion(context.Background()))
}

func TestProcessUpcomingIrrigationNotificationsFiresWithinLeadWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	now := time.Now().In(time.Local)
	scheduledAt := now.Add(3 * time.Minute) // dentro da janela de 5min de antecedência
	device := db.Device{ID: 10, Name: "Sensor Horta"}
	schedule := db.IrrigationSchedule{
		ID:              1,
		DeviceID:        device.ID,
		UserID:          42,
		Enabled:         true,
		StartTime:       time.Date(0, 1, 1, scheduledAt.Hour(), scheduledAt.Minute(), 0, 0, time.UTC),
		DurationSeconds: 60,
		DaysOfWeek:      weekdayList(now.Weekday()),
	}

	store.EXPECT().ListEnabledIrrigationSchedules(gomock.Any()).Return([]db.IrrigationSchedule{schedule}, nil)
	store.EXPECT().GetDeviceByID(gomock.Any(), device.ID).Return(device, nil)
	store.EXPECT().ListNotificationPreferencesByUser(gomock.Any(), schedule.UserID).Return(nil, nil)
	store.EXPECT().CreateNotification(gomock.Any(), gomock.Any()).Return(db.Notification{
		UserID: schedule.UserID,
		Type:   service.NotificationTypeAutoIrrigationStart,
	}, nil)
	store.EXPECT().ListPushTokensByUser(gomock.Any(), schedule.UserID).Return(nil, nil)
	store.EXPECT().UpdateIrrigationScheduleLeadNotified(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.UpdateIrrigationScheduleLeadNotifiedParams) error {
			require.Equal(t, schedule.ID, arg.ID)
			require.True(t, arg.LeadNotifiedAt.Valid)
			return nil
		})

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	require.NoError(t, irrigationService.ProcessUpcomingIrrigationNotifications(context.Background()))
}

func TestProcessUpcomingIrrigationNotificationsSkipsAlreadyNotifiedToday(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	now := time.Now().In(time.Local)
	scheduledAt := now.Add(3 * time.Minute)
	schedule := db.IrrigationSchedule{
		ID:              1,
		DeviceID:        10,
		UserID:          42,
		Enabled:         true,
		StartTime:       time.Date(0, 1, 1, scheduledAt.Hour(), scheduledAt.Minute(), 0, 0, time.UTC),
		DurationSeconds: 60,
		DaysOfWeek:      weekdayList(now.Weekday()),
		LeadNotifiedAt:  sql.NullTime{Time: now, Valid: true},
	}

	// Sem GetDeviceByID/CreateNotification esperados: já notificou hoje.
	store.EXPECT().ListEnabledIrrigationSchedules(gomock.Any()).Return([]db.IrrigationSchedule{schedule}, nil)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	require.NoError(t, irrigationService.ProcessUpcomingIrrigationNotifications(context.Background()))
}

func TestProcessUpcomingIrrigationNotificationsSkipsOutsideWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	now := time.Now().In(time.Local)
	scheduledAt := now.Add(10 * time.Minute) // fora da janela de 5min
	schedule := db.IrrigationSchedule{
		ID:              1,
		DeviceID:        10,
		UserID:          42,
		Enabled:         true,
		StartTime:       time.Date(0, 1, 1, scheduledAt.Hour(), scheduledAt.Minute(), 0, 0, time.UTC),
		DurationSeconds: 60,
		DaysOfWeek:      weekdayList(now.Weekday()),
	}

	store.EXPECT().ListEnabledIrrigationSchedules(gomock.Any()).Return([]db.IrrigationSchedule{schedule}, nil)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	require.NoError(t, irrigationService.ProcessUpcomingIrrigationNotifications(context.Background()))
}

func weekdayList(day time.Weekday) string {
	days := []string{"0", "1", "2", "3", "4", "5", "6"}
	return days[int(day)]
}
