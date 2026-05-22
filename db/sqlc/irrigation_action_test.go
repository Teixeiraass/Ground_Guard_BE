package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func createRandomIrrigationAction(t *testing.T) IrrigationAction {
	user := createRandomUser(t)
	device := createRandomDevice(t)

	arg := CreateIrrigationActionParams {
		DeviceID: device.ID,
		UserID: user.ID,
		TriggerType: "MANUAL",
	}


	irrigationAction, err := testQueries.CreateIrrigationAction(context.Background(), arg) 
	require.NoError(t, err)
	require.NotEmpty(t, irrigationAction)

	require.Equal(t, arg.DeviceID, irrigationAction.DeviceID)
	require.Equal(t, arg.UserID, irrigationAction.UserID)
	require.Equal(t, arg.TriggerType, irrigationAction.TriggerType)

	return irrigationAction
}

func TestCreateIrrigationAction(t *testing.T) {
	createRandomIrrigationAction(t)
}

func TestGetIrrigationAction(t *testing.T) {
	irrigationAction1 := createRandomIrrigationAction(t)
	irrigationAction2, err := testQueries.GetIrrigationAction(context.Background(), irrigationAction1.Uuid)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationAction2)

	require.Equal(t, irrigationAction1.ID, irrigationAction2.ID)
	require.Equal(t, irrigationAction1.Uuid, irrigationAction2.Uuid)
	require.Equal(t, irrigationAction1.DeviceID, irrigationAction2.DeviceID)
	require.Equal(t, irrigationAction1.UserID, irrigationAction2.UserID)
	require.Equal(t, irrigationAction1.StartedAt, irrigationAction2.StartedAt)
	require.Equal(t, irrigationAction1.FinishedAt, irrigationAction2.FinishedAt)
	require.Equal(t, irrigationAction1.DurationSeconds, irrigationAction2.DurationSeconds)
	require.Equal(t, irrigationAction1.Status, irrigationAction2.Status)
	require.Equal(t, irrigationAction1.TriggerType, irrigationAction2.TriggerType)
	require.Equal(t, irrigationAction1.WaterVolumeMl, irrigationAction2.WaterVolumeMl)
	require.Equal(t, irrigationAction1.ErrorMessage, irrigationAction2.ErrorMessage)
	require.WithinDuration(t, irrigationAction1.CreatedAt, irrigationAction2.CreatedAt, time.Second)
}

func TestUpdateIrrigationAction(t * testing.T) {
	irrigationAction1 := createRandomIrrigationAction(t)

	arg := UpdateIrrigationActionParams {
		Uuid: irrigationAction1.Uuid,
		FinishedAt: sql.NullTime{
			Time: time.Now(),
			Valid: true,
		},
		DurationSeconds: sql.NullInt32{
			Int32: int32(util.RandomInt(1, 180)),
			Valid: true,
		},
		Status: "FINALIZADO",
	}

	irrigationAction2, err := testQueries.UpdateIrrigationAction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationAction2)

	require.Equal(t, irrigationAction1.ID, irrigationAction2.ID)
	require.Equal(t, irrigationAction1.Uuid, irrigationAction2.Uuid)
	require.Equal(t, irrigationAction1.DeviceID, irrigationAction2.DeviceID)
	require.Equal(t, irrigationAction1.UserID, irrigationAction2.UserID)
	require.Equal(t, irrigationAction1.StartedAt, irrigationAction2.StartedAt)
	require.Equal(t, arg.FinishedAt.Valid, irrigationAction2.FinishedAt.Valid)
	require.True(t,
		arg.FinishedAt.Time.Equal(irrigationAction2.FinishedAt.Time),
	)
	require.Equal(t, arg.DurationSeconds, irrigationAction2.DurationSeconds)
	require.Equal(t, arg.Status, irrigationAction2.Status)
	require.Equal(t, irrigationAction1.TriggerType, irrigationAction2.TriggerType)
	require.Equal(t, irrigationAction1.WaterVolumeMl, irrigationAction2.WaterVolumeMl)
	require.Equal(t, irrigationAction1.ErrorMessage, irrigationAction2.ErrorMessage)
	require.WithinDuration(t, irrigationAction1.CreatedAt, irrigationAction2.CreatedAt, time.Second)
}

func TestListIrrigationAction(t *testing.T) {
	var lastIrrigationAction IrrigationAction

	for i := 0; i < 5; i++ {
		lastIrrigationAction = createRandomIrrigationAction(t)
	}

	arg := ListIrrigationActionParams {
		UserID: lastIrrigationAction.UserID,
		Limit: 5,
		Offset: 0,
	}

	irrigationActions, err := testQueries.ListIrrigationAction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationActions)

	for _, irrigationAction := range irrigationActions {
		require.NotEmpty(t, irrigationAction)
		require.Equal(t, irrigationAction.UserID, irrigationAction.UserID)
	}

}