package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func createRandomIrrigationPreferences(t *testing.T) IrrigationPreference {
	device := createRandomDevice(t)
	require.NotEmpty(t, device)

	startHour := time.Date(0, 1, 1, 7, 0, 0, 0, time.UTC)
	endHour := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	arg := CreateIrrigationPreferencesParams{
		DeviceID: device.ID,
		IrrigationMode: util.RandomIrrigationMode(),
		MoistureThreshold: int32(util.RandomInt(1, 100)),
		DryTimeMinutes: int32(util.RandomInt(1, 120)),
		MaxIrrigationsPerDay: int32(util.RandomInt(1, 5)),
		StartHour: sql.NullTime{
			Time: startHour,
			Valid: true,
		},
		EndHour: sql.NullTime{
			Time: endHour,
			Valid: true,
		},
	}

	irrigationPreference, err := testQueries.CreateIrrigationPreferences(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationPreference)

	require.Equal(t, arg.DeviceID, irrigationPreference.DeviceID)
	require.Equal(t, arg.IrrigationMode, irrigationPreference.IrrigationMode)
	require.Equal(t, arg.MoistureThreshold, irrigationPreference.MoistureThreshold)
	require.Equal(t, arg.DryTimeMinutes, irrigationPreference.DryTimeMinutes)
	require.Equal(t, arg.MaxIrrigationsPerDay, irrigationPreference.MaxIrrigationsPerDay)
	require.Equal(t, arg.StartHour, irrigationPreference.StartHour)
	require.Equal(t, arg.EndHour, irrigationPreference.EndHour)

	return irrigationPreference
}

func TestCreateIrrigationPreferences(t *testing.T) {
	createRandomIrrigationPreferences(t)
}

func TestGetIrrigationPreferences(t *testing.T) {
	irrigationPreferences1 := createRandomIrrigationPreferences(t)
	irrigationPreferences2, err := testQueries.GetIrrigationPreference(context.Background(), irrigationPreferences1.Uuid)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationPreferences2)

	require.Equal(t, irrigationPreferences1.ID, irrigationPreferences2.ID)
	require.Equal(t, irrigationPreferences1.Uuid, irrigationPreferences2.Uuid)
	require.Equal(t, irrigationPreferences1.DeviceID, irrigationPreferences2.DeviceID)
	require.Equal(t, irrigationPreferences1.Enabled, irrigationPreferences2.Enabled)
	require.Equal(t, irrigationPreferences1.IrrigationMode, irrigationPreferences2.IrrigationMode)
	require.Equal(t, irrigationPreferences1.MoistureThreshold, irrigationPreferences2.MoistureThreshold)
	require.Equal(t, irrigationPreferences1.DryTimeMinutes, irrigationPreferences2.DryTimeMinutes)
	require.Equal(t, irrigationPreferences1.IrrigationDurationSeconds, irrigationPreferences2.IrrigationDurationSeconds)
	require.Equal(t, irrigationPreferences1.MaxIrrigationsPerDay, irrigationPreferences2.MaxIrrigationsPerDay)
	require.Equal(t, irrigationPreferences1.StartHour, irrigationPreferences2.StartHour)
	require.Equal(t, irrigationPreferences1.EndHour, irrigationPreferences2.EndHour)
	require.Equal(t, irrigationPreferences1.CreatedAt, irrigationPreferences2.CreatedAt, time.Second)
	require.Equal(t, irrigationPreferences1.UpdatedAt, irrigationPreferences2.UpdatedAt, time.Second)
}

func TestGetIrrigationPreferencesByDevice(t *testing.T) {
	irrigationPreference1 := createRandomIrrigationPreferences(t)
	irrigationPreference2, err := testQueries.GetIrrigationPreferenceByDevice(context.Background(), irrigationPreference1.DeviceID)
	require.NoError(t, err)
	require.NotEmpty(t, irrigationPreference2)

	require.Equal(t, irrigationPreference1.ID, irrigationPreference1.ID)
	require.Equal(t, irrigationPreference1.Uuid, irrigationPreference1.Uuid)
	require.Equal(t, irrigationPreference1.DeviceID, irrigationPreference1.DeviceID)
	require.Equal(t, irrigationPreference1.Enabled, irrigationPreference1.Enabled)
	require.Equal(t, irrigationPreference1.IrrigationMode, irrigationPreference1.IrrigationMode)
	require.Equal(t, irrigationPreference1.MoistureThreshold, irrigationPreference1.MoistureThreshold)
	require.Equal(t, irrigationPreference1.DryTimeMinutes, irrigationPreference1.DryTimeMinutes)
	require.Equal(t, irrigationPreference1.IrrigationDurationSeconds, irrigationPreference1.IrrigationDurationSeconds)
	require.Equal(t, irrigationPreference1.MaxIrrigationsPerDay, irrigationPreference1.MaxIrrigationsPerDay)
	require.Equal(t, irrigationPreference1.StartHour, irrigationPreference1.StartHour)
	require.Equal(t, irrigationPreference1.EndHour, irrigationPreference1.EndHour)
	require.Equal(t, irrigationPreference1.CreatedAt, irrigationPreference1.CreatedAt, time.Second)
	require.Equal(t, irrigationPreference1.UpdatedAt, irrigationPreference1.UpdatedAt, time.Second)
}

func TestListIrrigationPreferences(t *testing.T) {
	for i := 0; i < 5; i++ {
		createRandomIrrigationPreferences(t)
	}

	args := ListIrrigationPreferencesParams{
		Limit: 5,
		Offset: 0,
	}

	irrigationPreferences, err := testQueries.ListIrrigationPreferences(context.Background(), args)
	require.NoError(t, err)
	require.Len(t, irrigationPreferences, 5)
}