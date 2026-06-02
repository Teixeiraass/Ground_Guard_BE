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
	user := createRandomUser(t)
	qrToken, err := util.GenerateQRToken(8)
	require.NoError(t, err)

	arg := CreateDeviceParams {
		DeviceUid: util.RandomString(8),
		Name: util.RandomString(6),
		FirmwareVersion: util.RandomFirwareVersion(),
		UserID: sql.NullInt64 {
			Int64: user.ID,
			Valid: true,
		},
		QrToken: qrToken,
	}

	device, err := testQueries.CreateDevice(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, device)

	require.Equal(t, arg.DeviceUid, device.DeviceUid)
	require.Equal(t, arg.Name, device.Name)
	require.Equal(t, arg.FirmwareVersion, device.FirmwareVersion)

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
	require.Equal(t, device1.FirmwareVersion, device2.FirmwareVersion)
	require.Equal(t, device1.FirmwareBuild, device2.FirmwareBuild)
	require.Equal(t, device1.LastUpdate, device2.LastUpdate)
	require.Equal(t, device1.IpAddress, device2.IpAddress)
	require.Equal(t, device1.WifiSsid, device2.WifiSsid)
	require.Equal(t, device1.LastSeen, device2.LastSeen)
	require.Equal(t, device1.Status, device2.Status)
	require.Equal(t, device1.UserID, device2.UserID)
	require.Equal(t, device1.QrToken, device2.QrToken)
	require.Equal(t, device1.QrCodeFile, device2.QrCodeFile)
	require.WithinDuration(t, device1.CreatedAt, device2.CreatedAt, time.Second)
}

func TestUpdateDevice(t *testing.T) {
	device1 := createRandomDevice(t)

	arg := UpdateDevicesParams {
		Uuid: device1.Uuid,
		Status: util.RandomStatus(),	
	}

	device2, err := testQueries.UpdateDevices(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, device2)
	
	require.Equal(t, device1.ID, device2.ID)
	require.Equal(t, device1.Uuid, device2.Uuid)
	require.Equal(t, device1.Name, device2.Name)
	require.Equal(t, device1.FirmwareVersion, device2.FirmwareVersion)
	require.Equal(t, device1.FirmwareBuild, device2.FirmwareBuild)
	require.Equal(t, device1.LastUpdate, device2.LastUpdate)
	require.Equal(t, device1.IpAddress, device2.IpAddress)
	require.Equal(t, device1.WifiSsid, device2.WifiSsid)
	require.Equal(t, device1.LastSeen, device2.LastSeen)
	require.Equal(t, arg.Status, device2.Status)
	require.Equal(t, device1.UserID, device2.UserID)
	require.Equal(t, device1.QrToken, device2.QrToken)
	require.Equal(t, device1.QrCodeFile, device2.QrCodeFile)
	require.WithinDuration(t, device1.CreatedAt, device2.CreatedAt, time.Second)
}

func TestUpdateLinkDeviceToUserByQrToken(t *testing.T) {
	device1 := createRandomDevice(t)
	user := createRandomUser(t)

	arg := LinkDeviceToUserByQrTokenParams {
		QrToken: device1.QrToken,
		UserID: sql.NullInt64{
			Int64: user.ID,
			Valid: true,
		},
	}

	device2, err := testQueries.LinkDeviceToUserByQrToken(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, device2)
	
	require.Equal(t, device1.ID, device2.ID)
	require.Equal(t, device1.Uuid, device2.Uuid)
	require.Equal(t, device1.Name, device2.Name)
	require.Equal(t, device1.FirmwareVersion, device2.FirmwareVersion)
	require.Equal(t, device1.FirmwareBuild, device2.FirmwareBuild)
	require.Equal(t, device1.LastUpdate, device2.LastUpdate)
	require.Equal(t, device1.IpAddress, device2.IpAddress)
	require.Equal(t, device1.WifiSsid, device2.WifiSsid)
	require.Equal(t, device1.LastSeen, device2.LastSeen)
	require.Equal(t, device1.Status, device2.Status)
	require.Equal(t, arg.UserID, device2.UserID)
	require.Equal(t, device1.QrToken, device2.QrToken)
	require.Equal(t, device1.QrCodeFile, device2.QrCodeFile)
	require.WithinDuration(t, device1.CreatedAt, device2.CreatedAt, time.Second)
}

func TestListDevices(t *testing.T) {
	var lastDevice Device
	for i := 0; i < 5; i++ {
		lastDevice = createRandomDevice(t)
	}

	arg := ListDevicesParams {
		UserID: lastDevice.UserID,
		Limit: 5,
		Offset: 0,
	}

	devices, err := testQueries.ListDevices(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, devices)

	for _, device := range devices {
		require.NotEmpty(t, device)
		require.Equal(t, lastDevice.UserID, device.UserID)
	}
}