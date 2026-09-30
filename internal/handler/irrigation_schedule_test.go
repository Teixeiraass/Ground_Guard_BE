package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	mockmqtt "github.com/Teixeiraass/ground_guard_be/mqtt/mock"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCreateIrrigationScheduleAPI(t *testing.T) {
	userID := util.RandomInt(1, 1000)
	device := randomIrrigationCommandDevice(userID)
	requestBody := dto.CreateIrrigationScheduleRequest{
		DeviceUUID:      device.Uuid.String(),
		Name:            "Morning watering",
		StartTime:       "06:30",
		DurationSeconds: 120,
		DaysOfWeek:      "1,3,5",
	}
	schedule := db.IrrigationSchedule{
		ID:              1,
		Uuid:            util.RandomUuid(),
		DeviceID:        device.ID,
		UserID:          userID,
		Name:            sql.NullString{String: requestBody.Name, Valid: true},
		Enabled:         true,
		StartTime:       time.Date(0, 1, 1, 6, 30, 0, 0, time.UTC),
		DurationSeconds: requestBody.DurationSeconds,
		DaysOfWeek:      requestBody.DaysOfWeek,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	store.EXPECT().GetDevice(gomock.Any(), device.Uuid).Return(device, nil)
	store.EXPECT().CreateIrrigationSchedule(gomock.Any(), gomock.Eq(db.CreateIrrigationScheduleParams{
		DeviceID: device.ID, UserID: userID, Name: requestBodyName(requestBody.Name),
		Enabled: true, StartTime: schedule.StartTime, DurationSeconds: 120, DaysOfWeek: "1,3,5",
	})).Return(schedule, nil)

	server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store, mqttClient)
	require.NoError(t, err)
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/irrigation/schedules", bytes.NewReader(body))
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, "user", userID, util.RandomUuid(), time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	var response dto.IrrigationScheduleResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, schedule.Uuid, response.Uuid)
	require.Equal(t, "06:30", response.StartTime)
}

func TestCreateIrrigationScheduleValidationAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store, mqttClient)
	require.NoError(t, err)

	body := bytes.NewBufferString(`{"device_uuid":"not-a-uuid","start_time":"25:00","duration_seconds":0,"days_of_week":"1,1"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/irrigation/schedules", body)
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, "user", 1, util.RandomUuid(), time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestListIrrigationSchedulesAPI(t *testing.T) {
	userID := util.RandomInt(1, 1000)
	schedule := db.IrrigationSchedule{
		Uuid:            util.RandomUuid(),
		DeviceID:        1,
		UserID:          userID,
		Enabled:         false,
		StartTime:       time.Date(0, 1, 1, 22, 15, 0, 0, time.UTC),
		DurationSeconds: 60,
		DaysOfWeek:      "0,6",
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	store.EXPECT().ListIrrigationSchedules(gomock.Any(), userID).Return([]db.IrrigationSchedule{schedule}, nil)
	server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store, mqttClient)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/irrigation/schedules", nil)
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, "user", userID, util.RandomUuid(), time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response []dto.IrrigationScheduleResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "22:15", response[0].StartTime)
}

func requestBodyName(name string) sql.NullString {
	return sql.NullString{String: name, Valid: true}
}
