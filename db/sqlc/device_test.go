package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func createRandomDevice(t *testing.T) Device {
	qrToken, err := util.GenerateQRToken(8)
	require.NoError(t, err)

	arg := CreateDeviceParams{
		DeviceUid: util.RandomString(8),
		Name:      util.RandomString(6),
		QrToken:   qrToken,
	}

	device, err := testQueries.CreateDevice(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, device)

	require.Equal(t, arg.DeviceUid, device.DeviceUid)
	require.Equal(t, arg.Name, device.Name)

	return device
}

func TestCreateDevice(t *testing.T) {
	createRandomDevice(t)
}

func TestGetDevice(t *testing.T) {
	//create device
	device1 := createRandomDevice(t)
	device2, err := testQueries.GetDevice(context.Background(), device1.Uuid)
	require.NoError(t, err)
	require.NotEmpty(t, device2)

	require.Equal(t, device1.ID, device2.ID)
	require.Equal(t, device1.Uuid, device2.Uuid)
	require.Equal(t, device1.DeviceUid, device2.DeviceUid)
	require.Equal(t, device1.LastUpdate, device2.LastUpdate)
	require.Equal(t, device1.LastSeen, device2.LastSeen)
	require.Equal(t, device1.Status, device2.Status)
	require.Equal(t, device1.ControllerDeviceID, device2.ControllerDeviceID)
	require.Equal(t, device1.QrToken, device2.QrToken)
	require.Equal(t, device1.QrCodeFile, device2.QrCodeFile)
	require.WithinDuration(t, device1.CreatedAt, device2.CreatedAt, time.Second)
}

func TestUpdateDevice(t *testing.T) {
	device1 := createRandomDevice(t)

	arg := UpdateDevicesParams{
		Uuid:   device1.Uuid,
		Status: util.RandomStatus(),
	}

	device2, err := testQueries.UpdateDevices(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, device2)

	require.Equal(t, device1.ID, device2.ID)
	require.Equal(t, device1.Uuid, device2.Uuid)
	require.Equal(t, device1.Name, device2.Name)
	require.Equal(t, device1.LastUpdate, device2.LastUpdate)
	require.Equal(t, device1.LastSeen, device2.LastSeen)
	require.Equal(t, arg.Status, device2.Status)
	require.Equal(t, device1.ControllerDeviceID, device2.ControllerDeviceID)
	require.Equal(t, device1.QrToken, device2.QrToken)
	require.Equal(t, device1.QrCodeFile, device2.QrCodeFile)
	require.WithinDuration(t, device1.CreatedAt, device2.CreatedAt, time.Second)
}

func TestUpdateLinkDeviceToControllerDeviceByQrToken(t *testing.T) {
	device1 := createRandomDevice(t)
	controllerDevice := createRandomControllerDevice(t)

	arg := LinkDeviceToControllerDeviceByQrTokenParams{
		QrToken: device1.QrToken,
		ControllerDeviceID: sql.NullInt64{
			Int64: controllerDevice.ID,
			Valid: true,
		},
	}

	device2, err := testQueries.LinkDeviceToControllerDeviceByQrToken(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, device2)

	require.Equal(t, device1.ID, device2.ID)
	require.Equal(t, device1.Uuid, device2.Uuid)
	require.Equal(t, arg.ControllerDeviceID, device2.ControllerDeviceID)
	require.WithinDuration(t, device1.CreatedAt, device2.CreatedAt, time.Second)
}

func TestLinkDeviceToControllerDeviceByQrToken(t *testing.T) {
	device1 := createRandomDevice(t)
	controllerDevice := createRandomControllerDevice(t)

	arg := LinkDeviceToControllerDeviceByQrTokenParams{
		QrToken: device1.QrToken,
		ControllerDeviceID: sql.NullInt64{
			Int64: controllerDevice.ID,
			Valid: true,
		},
	}

	device2, err := testQueries.LinkDeviceToControllerDeviceByQrToken(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, device2)

	require.Equal(t, device1.ID, device2.ID)
	require.Equal(t, device1.Uuid, device2.Uuid)
	require.Equal(t, device1.Name, device2.Name)
	require.Equal(t, device1.LastUpdate, device2.LastUpdate)
	require.Equal(t, device1.LastSeen, device2.LastSeen)
	require.Equal(t, device1.Status, device2.Status)
	require.Equal(t, arg.ControllerDeviceID, device2.ControllerDeviceID)
	require.Equal(t, device1.QrToken, device2.QrToken)
	require.Equal(t, device1.QrCodeFile, device2.QrCodeFile)
	require.WithinDuration(t, device1.CreatedAt, device2.CreatedAt, time.Second)
}

func TestListDevices(t *testing.T) {
	controllerDevice := createRandomControllerDevice(t)

	for i := 0; i < 5; i++ {
		device := createRandomDevice(t)

		_, err := testQueries.LinkDeviceToControllerDeviceByQrToken(
			context.Background(),
			LinkDeviceToControllerDeviceByQrTokenParams{
				QrToken: device.QrToken,
				ControllerDeviceID: sql.NullInt64{
					Int64: controllerDevice.ID,
					Valid: true,
				},
			},
		)

		require.NoError(t, err)
	}

	arg := ListDevicesParams{
		ControllerDeviceID: sql.NullInt64{
			Int64: controllerDevice.ID,
			Valid: true,
		},
		Limit:  5,
		Offset: 0,
	}

	devices, err := testQueries.ListDevices(context.Background(), arg)

	require.NoError(t, err)
	require.Len(t, devices, 5)

	for _, device := range devices {
		require.Equal(t, controllerDevice.ID, device.ControllerDeviceID.Int64)
	}
}
