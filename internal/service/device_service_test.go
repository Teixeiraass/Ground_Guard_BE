package service_test

import (
	"context"
	"database/sql"
	"testing"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAssignDeviceToGroupMovesDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	userID := int64(42)
	deviceUUID := uuid.New()
	groupUUID := uuid.New()
	group := db.DeviceGroup{ID: 9, Uuid: groupUUID, UserID: userID}

	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceUUID).Return(sql.NullInt64{Int64: userID, Valid: true}, nil)
	store.EXPECT().GetDeviceGroup(gomock.Any(), db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID}).Return(group, nil)
	store.EXPECT().AssignDeviceToGroup(gomock.Any(), db.AssignDeviceToGroupParams{
		Uuid:    deviceUUID,
		GroupID: sql.NullInt64{Int64: group.ID, Valid: true},
	}).Return(db.Device{Uuid: deviceUUID, GroupID: sql.NullInt64{Int64: group.ID, Valid: true}}, nil)

	deviceService := service.NewDeviceService(store)
	device, err := deviceService.AssignDeviceToGroup(context.Background(), userID, deviceUUID, &groupUUID)
	require.NoError(t, err)
	require.True(t, device.GroupID.Valid)
	require.Equal(t, group.ID, device.GroupID.Int64)
}

func TestAssignDeviceToGroupRejectsUnownedDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	deviceUUID := uuid.New()
	groupUUID := uuid.New()

	// Device pertence a outro usuário (99), não a quem está chamando (42).
	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceUUID).Return(sql.NullInt64{Int64: 99, Valid: true}, nil)
	// Nenhuma chamada a GetDeviceGroup/AssignDeviceToGroup esperada.

	deviceService := service.NewDeviceService(store)
	_, err := deviceService.AssignDeviceToGroup(context.Background(), 42, deviceUUID, &groupUUID)
	require.Error(t, err)
}

func TestAssignDeviceToGroupClearsGroupWhenNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	userID := int64(42)
	deviceUUID := uuid.New()

	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceUUID).Return(sql.NullInt64{Int64: userID, Valid: true}, nil)
	store.EXPECT().AssignDeviceToGroup(gomock.Any(), db.AssignDeviceToGroupParams{
		Uuid:    deviceUUID,
		GroupID: sql.NullInt64{},
	}).Return(db.Device{Uuid: deviceUUID}, nil)

	deviceService := service.NewDeviceService(store)
	device, err := deviceService.AssignDeviceToGroup(context.Background(), userID, deviceUUID, nil)
	require.NoError(t, err)
	require.False(t, device.GroupID.Valid)
}
