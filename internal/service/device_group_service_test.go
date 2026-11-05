package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	mockmqtt "github.com/Teixeiraass/ground_guard_be/mqtt/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTriggerGroupIrrigationPartialFailureDoesNotAbort(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	userID := int64(42)
	groupUUID := uuid.New()
	group := db.DeviceGroup{ID: 5, Uuid: groupUUID, UserID: userID, Name: "Tomate"}

	deviceOK := db.Device{ID: 1, Uuid: uuid.New(), DeviceUid: "AA11", Status: "ativo"}
	deviceBusy := db.Device{ID: 2, Uuid: uuid.New(), DeviceUid: "BB22", Status: "ativo"}

	store.EXPECT().GetDeviceGroup(gomock.Any(), db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID}).Return(group, nil)
	store.EXPECT().ListDevicesByGroupID(gomock.Any(), sql.NullInt64{Int64: group.ID, Valid: true}).
		Return([]db.Device{deviceOK, deviceBusy}, nil)

	// device OK: segue até publicar o comando com sucesso.
	store.EXPECT().GetDevice(gomock.Any(), deviceOK.Uuid).Return(deviceOK, nil)
	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceOK.Uuid).Return(sql.NullInt64{Int64: userID, Valid: true}, nil)
	store.EXPECT().ExistsPendingIrrigationCommand(gomock.Any(), deviceOK.ID).Return(false, nil)
	store.EXPECT().ExistsActiveIrrigationAction(gomock.Any(), deviceOK.ID).Return(false, nil)
	store.EXPECT().CreateIrrigationCommand(gomock.Any(), gomock.Any()).
		Return(db.IrrigationCommand{Uuid: uuid.New(), Action: "START"}, nil)
	mqttClient.EXPECT().TopicPrefix().Return("ground-guard")
	mqttClient.EXPECT().Publish("ground-guard/devices/AA11/commands", gomock.Any()).Return(nil)

	// device já irrigando: CreateIrrigationCommand nunca é chamado pra ele.
	store.EXPECT().GetDevice(gomock.Any(), deviceBusy.Uuid).Return(deviceBusy, nil)
	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceBusy.Uuid).Return(sql.NullInt64{Int64: userID, Valid: true}, nil)
	store.EXPECT().ExistsPendingIrrigationCommand(gomock.Any(), deviceBusy.ID).Return(false, nil)
	store.EXPECT().ExistsActiveIrrigationAction(gomock.Any(), deviceBusy.ID).Return(true, nil)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	groupService := service.NewDeviceGroupService(store, irrigationService)

	results, err := groupService.TriggerGroupIrrigation(context.Background(), userID, groupUUID, "START")
	require.NoError(t, err)
	require.Len(t, results, 2)

	byUUID := map[uuid.UUID]dto.GroupIrrigationCommandResult{}
	for _, r := range results {
		byUUID[r.DeviceUuid] = r
	}
	require.True(t, byUUID[deviceOK.Uuid].Success)
	require.False(t, byUUID[deviceBusy.Uuid].Success)
	require.Contains(t, byUUID[deviceBusy.Uuid].Error, "already irrigating")
}

func TestCreateGroupScheduleTagsAllWithGroupID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	userID := int64(42)
	groupUUID := uuid.New()
	group := db.DeviceGroup{ID: 7, Uuid: groupUUID, UserID: userID, Name: "Alface"}

	device1 := db.Device{ID: 1, Uuid: uuid.New(), Status: "ativo"}
	device2 := db.Device{ID: 2, Uuid: uuid.New(), Status: "ativo"}

	store.EXPECT().GetDeviceGroup(gomock.Any(), db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID}).Return(group, nil)
	store.EXPECT().ListDevicesByGroupID(gomock.Any(), sql.NullInt64{Int64: group.ID, Valid: true}).
		Return([]db.Device{device1, device2}, nil)

	for i, device := range []db.Device{device1, device2} {
		scheduleID := int64(100 + i)
		store.EXPECT().GetDevice(gomock.Any(), device.Uuid).Return(device, nil)
		store.EXPECT().GetDeviceUserID(gomock.Any(), device.Uuid).Return(sql.NullInt64{Int64: userID, Valid: true}, nil)
		store.EXPECT().CreateIrrigationSchedule(gomock.Any(), gomock.Any()).
			Return(db.IrrigationSchedule{ID: scheduleID, DeviceID: device.ID, UserID: userID}, nil)
		store.EXPECT().SetIrrigationScheduleGroup(gomock.Any(), db.SetIrrigationScheduleGroupParams{
			ID:      scheduleID,
			GroupID: sql.NullInt64{Int64: group.ID, Valid: true},
		}).Return(nil)
	}

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	groupService := service.NewDeviceGroupService(store, irrigationService)

	schedules, err := groupService.CreateGroupSchedule(context.Background(), userID, groupUUID, dto.CreateGroupScheduleRequest{
		StartTime:       "06:30",
		DurationSeconds: 60,
		DaysOfWeek:      "1,3,5",
	})
	require.NoError(t, err)
	require.Len(t, schedules, 2)
	for _, s := range schedules {
		require.True(t, s.GroupID.Valid)
		require.Equal(t, group.ID, s.GroupID.Int64)
	}
}

func TestGetGroupStatisticsAggregatesMoisture(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	userID := int64(42)
	groupUUID := uuid.New()
	group := db.DeviceGroup{ID: 11, Uuid: groupUUID, UserID: userID, Name: "Cenoura"}

	deviceLow := db.Device{ID: 1, Uuid: uuid.New(), IsOnline: true, SoilMoisture: sql.NullInt32{Int32: 20, Valid: true}}
	deviceHigh := db.Device{ID: 2, Uuid: uuid.New(), IsOnline: false, IsIrrigating: false, SoilMoisture: sql.NullInt32{Int32: 60, Valid: true}}
	deviceNoReading := db.Device{ID: 3, Uuid: uuid.New(), IsOnline: true, IsIrrigating: true}

	store.EXPECT().GetDeviceGroup(gomock.Any(), db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID}).Return(group, nil)
	store.EXPECT().ListDevicesByGroupID(gomock.Any(), sql.NullInt64{Int64: group.ID, Valid: true}).
		Return([]db.Device{deviceLow, deviceHigh, deviceNoReading}, nil)
	// Chamado 2x: uma vez por ListIrrigationHistoryForDevices, outra por
	// GetWaterConsumptionForDevices (cada uma carrega as ações do zero).
	store.EXPECT().ListIrrigationAction(gomock.Any(), gomock.Any()).Return([]db.IrrigationAction{}, nil).Times(2)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	groupService := service.NewDeviceGroupService(store, irrigationService)

	stats, err := groupService.GetGroupStatistics(context.Background(), userID, groupUUID, time.Time{}, "week")
	require.NoError(t, err)
	require.Equal(t, 3, stats.DeviceCount)
	require.Equal(t, 2, stats.OnlineCount)
	require.Equal(t, 1, stats.IrrigatingCount)
	require.NotNil(t, stats.AvgSoilMoisture)
	require.Equal(t, 40.0, *stats.AvgSoilMoisture) // (20+60)/2, o device sem leitura é ignorado
	require.Equal(t, int32(20), *stats.MinSoilMoisture)
	require.Equal(t, int32(60), *stats.MaxSoilMoisture)
}

func TestGetGroupStatisticsEmptyGroupReturnsZeroedStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)

	userID := int64(42)
	groupUUID := uuid.New()
	group := db.DeviceGroup{ID: 12, Uuid: groupUUID, UserID: userID, Name: "Vazio"}

	store.EXPECT().GetDeviceGroup(gomock.Any(), db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID}).Return(group, nil)
	store.EXPECT().ListDevicesByGroupID(gomock.Any(), sql.NullInt64{Int64: group.ID, Valid: true}).Return(nil, nil)
	store.EXPECT().ListIrrigationAction(gomock.Any(), gomock.Any()).Return([]db.IrrigationAction{}, nil)

	irrigationService := service.NewIrrigationService(store, mqttClient, newTestNotificationService(store))
	groupService := service.NewDeviceGroupService(store, irrigationService)

	stats, err := groupService.GetGroupStatistics(context.Background(), userID, groupUUID, time.Time{}, "week")
	require.NoError(t, err)
	require.Equal(t, 0, stats.DeviceCount)
	require.Nil(t, stats.AvgSoilMoisture)
	require.Empty(t, stats.IrrigationHistory)
	require.Equal(t, int64(0), stats.WaterConsumption.CurrentTotalMl)
}
