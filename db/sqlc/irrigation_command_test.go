package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createRandomIrrigationCommand(t *testing.T, user User, device Device) IrrigationCommand {
	arg := CreateIrrigationCommandParams{
		DeviceID: device.ID,
		UserID:   user.ID,
		Action:   "START",
	}

	irrigationCommand, err := testQueries.CreateIrrigationCommand(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationCommand)

	require.Equal(t, arg.DeviceID, irrigationCommand.DeviceID)
	require.Equal(t, arg.UserID, irrigationCommand.UserID)
	require.Equal(t, arg.Action, irrigationCommand.Action)

	return irrigationCommand
}

func TestCreateIrrigationCommand(t *testing.T) {
	user := createRandomUser(t)
	device := createRandomDevice(t)
	createRandomIrrigationCommand(t, user, device)
}

func TestGetIrrigationCommand(t *testing.T) {
	irrigationCommand1 := createRandomIrrigationCommand(t, createRandomUser(t), createRandomDevice(t))
	irrigationCommand2, err := testQueries.GetIrrigationCommand(context.Background(), irrigationCommand1.Uuid)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationCommand2)

	require.Equal(t, irrigationCommand1.ID, irrigationCommand2.ID)
	require.Equal(t, irrigationCommand1.Uuid, irrigationCommand2.Uuid)
	require.Equal(t, irrigationCommand1.DeviceID, irrigationCommand2.DeviceID)
	require.Equal(t, irrigationCommand1.UserID, irrigationCommand2.UserID)
	require.Equal(t, irrigationCommand1.Action, irrigationCommand2.Action)
	require.Equal(t, irrigationCommand1.Status, irrigationCommand2.Status)
	require.WithinDuration(t, irrigationCommand1.CreatedAt, irrigationCommand2.CreatedAt, time.Second)
}

func TestListIrrigationCommands(t *testing.T) {
	// Create 10 irrigation commands
	for i := 0; i < 10; i++ {
		user := createRandomUser(t)
		device := createRandomDevice(t)
		createRandomIrrigationCommand(t, user, device)
	}

	arg := ListIrrigationCommandsParams{
		Limit:  5,
		Offset: 5,
	}

	irrigationCommands, err := testQueries.ListIrrigationCommands(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, irrigationCommands, 5)

	for _, irrigationCommand := range irrigationCommands {
		require.NotEmpty(t, irrigationCommand)
	}
}

func TestUpdateIrrigationCommand(t *testing.T) {
	irrigationCommand1 := createRandomIrrigationCommand(t, createRandomUser(t), createRandomDevice(t))

	arg := UpdateIrrigationCommandStatusParams{
		Uuid:   irrigationCommand1.Uuid,
		Status: "SUCCESS",
	}

	irrigationCommand2, err := testQueries.UpdateIrrigationCommandStatus(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationCommand2)

	require.Equal(t, irrigationCommand1.ID, irrigationCommand2.ID)
	require.Equal(t, irrigationCommand1.Uuid, irrigationCommand2.Uuid)
	require.Equal(t, irrigationCommand1.DeviceID, irrigationCommand2.DeviceID)
	require.Equal(t, irrigationCommand1.UserID, irrigationCommand2.UserID)
	require.Equal(t, irrigationCommand1.Action, irrigationCommand2.Action)
	require.Equal(t, arg.Status, irrigationCommand2.Status)
	require.WithinDuration(t, irrigationCommand1.CreatedAt, irrigationCommand2.CreatedAt, time.Second)
}

func TestDeleteIrrigationCommand(t *testing.T) {
	irrigationCommand1 := createRandomIrrigationCommand(t, createRandomUser(t), createRandomDevice(t))

	err := testQueries.DeleteIrrigationCommand(context.Background(), irrigationCommand1.Uuid)
	require.NoError(t, err)

	irrigationCommand2, err := testQueries.GetIrrigationCommand(context.Background(), irrigationCommand1.Uuid)
	require.Error(t, err)
	require.EqualError(t, err, "sql: no rows in result set")
	require.Empty(t, irrigationCommand2)
}
