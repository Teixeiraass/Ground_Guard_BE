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
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var soilNow = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

func liveDevice(moisture int32) db.Device {
	return db.Device{
		Uuid:         uuid.New(),
		Name:         "Sensor",
		LastSeen:     sql.NullTime{Time: soilNow.Add(-time.Minute), Valid: true},
		SoilMoisture: sql.NullInt32{Int32: moisture, Valid: true},
	}
}

func history(values ...int32) []db.DeviceSensorHistory {
	out := make([]db.DeviceSensorHistory, len(values))
	for i, v := range values {
		out[i] = db.DeviceSensorHistory{SoilMoisture: v}
	}
	return out
}

func TestComputeSoilHealthNoDevices(t *testing.T) {
	rsp := service.ComputeSoilHealth(nil, soilNow)
	require.Equal(t, dto.SoilHealthNoDevices, rsp.Status)
	require.Nil(t, rsp.Score)
	require.Nil(t, rsp.AvgMoisture)
}

func TestComputeSoilHealthAllInRangeIsExcellent(t *testing.T) {
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(55), History: history(50, 52, 60, 58)},
		{Device: liveDevice(65), History: history(60, 70, 66)},
	}, soilNow)

	require.Equal(t, dto.SoilHealthExcellent, rsp.Status)
	require.Equal(t, 100, *rsp.Score)
	require.Equal(t, 60, *rsp.AvgMoisture)
	require.Equal(t, 2, rsp.Summary.OK)
	require.Equal(t, 100, *rsp.Sensors[0].TimeInRange24h)
}

func TestComputeSoilHealthUsesTimeInRange(t *testing.T) {
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		// 2 of 4 readings in the default 35-80 range.
		{Device: liveDevice(50), History: history(20, 25, 50, 55)},
	}, soilNow)

	require.Equal(t, 50, *rsp.Sensors[0].Score)
	require.Equal(t, dto.SoilHealthAttention, rsp.Status)
}

func TestComputeSoilHealthCurrentlyDryIsCapped(t *testing.T) {
	// Always in range historically, but dry right now: never reported as healthy.
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(20), History: history(50, 50, 50, 50)},
	}, soilNow)

	require.Equal(t, dto.SensorStateDry, rsp.Sensors[0].State)
	require.Equal(t, 70, *rsp.Sensors[0].Score)
	require.Equal(t, 1, rsp.Summary.Dry)
}

func TestComputeSoilHealthWithoutHistoryUsesDistance(t *testing.T) {
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(90)}, // 10 points above the 80% ceiling
	}, soilNow)

	require.Equal(t, dto.SensorStateWet, rsp.Sensors[0].State)
	require.Equal(t, 40, *rsp.Sensors[0].Score)
}

func TestComputeSoilHealthUsesIrrigationThreshold(t *testing.T) {
	pref := &db.IrrigationPreference{MoistureThreshold: 50}
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(45), Preference: pref},
	}, soilNow)

	require.Equal(t, int32(50), rsp.Sensors[0].IdealMin)
	require.Equal(t, dto.SensorStateDry, rsp.Sensors[0].State)
}

func TestComputeSoilHealthExcludesFaultAndOffline(t *testing.T) {
	faulty := liveDevice(0)
	faulty.SoilSensorOk = sql.NullBool{Bool: false, Valid: true}

	offline := liveDevice(10)
	offline.LastSeen = sql.NullTime{Time: soilNow.Add(-2 * time.Hour), Valid: true}

	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: faulty},
		{Device: offline},
		{Device: liveDevice(60)},
	}, soilNow)

	require.Equal(t, 1, rsp.Summary.Fault)
	require.Equal(t, 1, rsp.Summary.Offline)
	require.Nil(t, rsp.Sensors[0].Score)
	require.Nil(t, rsp.Sensors[1].Score)
	require.Equal(t, 100, *rsp.Score)
	require.Equal(t, 60, *rsp.AvgMoisture, "faulty/offline readings must not skew the average")
}

func TestComputeSoilHealthOnlyUnusableSensorsIsNoData(t *testing.T) {
	neverSeen := db.Device{Uuid: uuid.New()}
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{{Device: neverSeen}}, soilNow)

	require.Equal(t, dto.SoilHealthNoData, rsp.Status)
	require.Equal(t, dto.SensorStateOffline, rsp.Sensors[0].State)
}

func TestGetSoilHealthLoadsPreferencesAndHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	device := db.Device{
		ID:           3,
		Uuid:         uuid.New(),
		LastSeen:     sql.NullTime{Time: time.Now(), Valid: true},
		SoilMoisture: sql.NullInt32{Int32: 60, Valid: true},
	}
	store.EXPECT().
		ListDevicesByUserID(gomock.Any(), db.ListDevicesByUserIDParams{
			UserID: sql.NullInt64{Int64: 9, Valid: true}, Limit: 500,
		}).
		Return([]db.Device{device}, nil)
	store.EXPECT().GetIrrigationPreferenceByDevice(gomock.Any(), int64(3)).Return(db.IrrigationPreference{}, sql.ErrNoRows)
	store.EXPECT().
		ListDeviceSensorHistory(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.ListDeviceSensorHistoryParams) ([]db.DeviceSensorHistory, error) {
			require.Equal(t, int64(3), arg.DeviceID)
			require.WithinDuration(t, time.Now().Add(-24*time.Hour), arg.CreatedAt, time.Minute)
			return history(55, 60, 65), nil
		})

	rsp, err := service.NewSoilHealthService(store).GetSoilHealth(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, dto.SoilHealthExcellent, rsp.Status)
	require.Equal(t, 3, rsp.Sensors[0].Readings24h)
}

func TestUpdateDeviceLocationRejectsOtherUsersDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	deviceUUID := uuid.New()

	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceUUID).Return(sql.NullInt64{Int64: 2, Valid: true}, nil)
	store.EXPECT().UpdateDeviceLocation(gomock.Any(), gomock.Any()).Times(0)

	lat, lng := -23.5, -46.6
	_, err := service.NewDeviceService(store).UpdateDeviceLocation(context.Background(), 1, deviceUUID, &lat, &lng)
	require.ErrorIs(t, err, service.ErrDeviceNotOwned)
}

func TestUpdateDeviceLocationSetsAndClears(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	deviceUUID := uuid.New()
	lat, lng := -23.5, -46.6

	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceUUID).Times(2).Return(sql.NullInt64{Int64: 1, Valid: true}, nil)
	gomock.InOrder(
		store.EXPECT().UpdateDeviceLocation(gomock.Any(), db.UpdateDeviceLocationParams{
			Uuid:      deviceUUID,
			Latitude:  sql.NullFloat64{Float64: lat, Valid: true},
			Longitude: sql.NullFloat64{Float64: lng, Valid: true},
		}).Return(db.Device{}, nil),
		store.EXPECT().UpdateDeviceLocation(gomock.Any(), db.UpdateDeviceLocationParams{Uuid: deviceUUID}).Return(db.Device{}, nil),
	)

	svc := service.NewDeviceService(store)
	_, err := svc.UpdateDeviceLocation(context.Background(), 1, deviceUUID, &lat, &lng)
	require.NoError(t, err)
	_, err = svc.UpdateDeviceLocation(context.Background(), 1, deviceUUID, nil, nil)
	require.NoError(t, err)
}

func TestUpdateNameDeviceRejectsOtherUsersDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	deviceUUID := uuid.New()

	store.EXPECT().GetDeviceUserID(gomock.Any(), deviceUUID).Return(sql.NullInt64{Int64: 2, Valid: true}, nil)
	store.EXPECT().UpdateNameDevice(gomock.Any(), gomock.Any()).Times(0)

	_, err := service.NewDeviceService(store).UpdateNameDevice(context.Background(), 1, deviceUUID, "Horta")
	require.ErrorIs(t, err, service.ErrDeviceNotOwned)
}
