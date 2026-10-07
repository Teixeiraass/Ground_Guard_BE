package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func createRandomControllerDevice(t *testing.T) ControllerDevice {
	user := createRandomUser(t)
	controllerDevice, err := testQueries.CreateControllerDevice(context.Background(), CreateControllerDeviceParams{
		ControllerDeviceUid: util.RandomString(8),
		Name:                util.RandomString(6),
		FirmwareVersion:     "1.0.0",
		UserID:              sql.NullInt64{Int64: user.ID, Valid: true},
	})
	require.NoError(t, err)
	return controllerDevice
}

func TestCreateControllerDevice(t *testing.T) {
	controllerDevice := createRandomControllerDevice(t)
	require.NotEmpty(t, controllerDevice)
}

func TestGetControllerDevice(t *testing.T) {
	created := createRandomControllerDevice(t)
	found, err := testQueries.GetControllerDevice(context.Background(), created.Uuid)
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, created.ControllerDeviceUid, found.ControllerDeviceUid)
	require.WithinDuration(t, created.CreatedAt, found.CreatedAt, time.Second)
}

func TestListControllerDevices(t *testing.T) {
	user := createRandomUser(t)
	for i := 0; i < 5; i++ {
		_, err := testQueries.CreateControllerDevice(context.Background(), CreateControllerDeviceParams{
			ControllerDeviceUid: util.RandomString(8),
			Name:                util.RandomString(6),
			FirmwareVersion:     "1.0.0",
			UserID:              sql.NullInt64{Int64: user.ID, Valid: true},
		})
		require.NoError(t, err)
	}

	devices, err := testQueries.ListControllerDevices(context.Background(), ListControllerDevicesParams{
		UserID: sql.NullInt64{Int64: user.ID, Valid: true},
		Limit:  5,
	})
	require.NoError(t, err)
	require.Len(t, devices, 5)
}
