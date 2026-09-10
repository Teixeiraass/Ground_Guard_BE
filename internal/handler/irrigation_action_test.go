package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	mockmqtt "github.com/Teixeiraass/ground_guard_be/mqtt/mock"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestListIrrigationHistoryAPI(t *testing.T) {
	user, _ := randomUser(t)
	user.ID = util.RandomInt(1, 1000)
	user.Uuid = util.RandomUuid()
	date := time.Now().In(time.Local)
	dateString := date.Format("2006-01-02")

	oldestAction := randomIrrigationHistoryAction(user.ID)
	newestAction := randomIrrigationHistoryAction(user.ID)
	newestAction.ID = oldestAction.ID + 1
	newestAction.Uuid = util.RandomUuid()
	oldestAction.StartedAt = date.Add(2 * time.Hour)
	oldestAction.CreatedAt = oldestAction.StartedAt
	newestAction.StartedAt = date.Add(3 * time.Hour)
	newestAction.CreatedAt = newestAction.StartedAt
	oldestAction.FinishedAt = sql.NullTime{Time: oldestAction.StartedAt.Add(30 * time.Minute), Valid: true}
	newestAction.FinishedAt = sql.NullTime{Time: newestAction.StartedAt.Add(30 * time.Minute), Valid: true}

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient) {
				store.EXPECT().
					ListIrrigationAction(gomock.Any(), gomock.Eq(db.ListIrrigationActionParams{UserID: user.ID, Limit: 100, Offset: 0})).
					Return([]db.IrrigationAction{oldestAction, newestAction}, nil)
				mqttClient.EXPECT().TopicPrefix().Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var got []dto.IrrigationHistoryResponse
				require.NoError(t, json.Unmarshal(data, &got))
				require.Len(t, got, 2)
				require.Equal(t, newestAction.Uuid, got[0].Uuid)
				require.Equal(t, oldestAction.Uuid, got[1].Uuid)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			mqttClient := mockmqtt.NewMockClient(ctrl)
			tc.buildStubs(store, mqttClient)

			server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store, mqttClient)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, "/api/v1/irrigation/history?date="+dateString, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetIrrigationHistoryAPI(t *testing.T) {
	user, _ := randomUser(t)
	user.ID = util.RandomInt(1, 1000)
	user.Uuid = util.RandomUuid()
	action := randomIrrigationHistoryAction(user.ID)

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient) {
				store.EXPECT().
					GetIrrigationAction(gomock.Any(), gomock.Eq(action.Uuid)).
					Return(action, nil)
				mqttClient.EXPECT().TopicPrefix().Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var got dto.IrrigationHistoryResponse
				require.NoError(t, json.Unmarshal(data, &got))
				require.Equal(t, action.Uuid, got.Uuid)
				require.NotNil(t, got.WaterVolumeMl)
				require.Equal(t, action.WaterVolumeMl.Int32, *got.WaterVolumeMl)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			mqttClient := mockmqtt.NewMockClient(ctrl)
			tc.buildStubs(store, mqttClient)

			server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store, mqttClient)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, "/api/v1/irrigation/history/"+action.Uuid.String(), nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetWaterConsumptionAPI(t *testing.T) {
	user, _ := randomUser(t)
	user.ID = util.RandomInt(1, 1000)
	user.Uuid = util.RandomUuid()
	now := time.Now().In(time.Local)
	weekStart := startOfISOWeekForTest(now)
	previousWeekStart := weekStart.AddDate(0, 0, -7)
	device := randomIrrigationCommandDevice(user.ID)

	currentAction := randomIrrigationHistoryAction(user.ID)
	currentAction.DeviceID = device.ID
	currentAction.StartedAt = weekStart.Add(2*24*time.Hour + 3*time.Hour)
	currentAction.CreatedAt = currentAction.StartedAt
	currentAction.FinishedAt = sql.NullTime{Time: currentAction.StartedAt.Add(45 * time.Minute), Valid: true}
	currentAction.DurationSeconds = sql.NullInt32{Int32: 2700, Valid: true}
	currentAction.Status = "FINALIZADO"
	currentAction.WaterVolumeMl = sql.NullInt32{Int32: 750, Valid: true}

	previousAction := randomIrrigationHistoryAction(user.ID)
	previousAction.DeviceID = device.ID
	previousAction.StartedAt = previousWeekStart.Add(2*24*time.Hour + 3*time.Hour)
	previousAction.CreatedAt = previousAction.StartedAt
	previousAction.FinishedAt = sql.NullTime{Time: previousAction.StartedAt.Add(30 * time.Minute), Valid: true}
	previousAction.DurationSeconds = sql.NullInt32{Int32: 1800, Valid: true}
	previousAction.Status = "FINALIZADO"
	previousAction.WaterVolumeMl = sql.NullInt32{Int32: 500, Valid: true}

	inactiveAction := randomIrrigationHistoryAction(user.ID)
	inactiveAction.DeviceID = device.ID
	inactiveAction.StartedAt = weekStart.Add(4 * time.Hour)
	inactiveAction.Status = "ATIVO"
	inactiveAction.WaterVolumeMl = sql.NullInt32{}

	storeExpectation := []db.IrrigationAction{currentAction, previousAction, inactiveAction}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Return(device, nil)
	store.EXPECT().
		ListIrrigationAction(gomock.Any(), gomock.Eq(db.ListIrrigationActionParams{UserID: user.ID, Limit: 100, Offset: 0})).
		Return(storeExpectation, nil)
	mqttClient.EXPECT().TopicPrefix().Times(0)

	server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store, mqttClient)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/irrigation/consumption?period=week&device_uuid="+device.Uuid.String(), nil)
	require.NoError(t, err)
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	data, err := io.ReadAll(recorder.Body)
	require.NoError(t, err)

	var got dto.WaterConsumptionResponse
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "week", got.Period)
	require.Equal(t, int64(750), got.CurrentTotalMl)
	require.Equal(t, int64(500), got.PreviousTotalMl)
	require.Equal(t, int64(250), got.DifferenceMl)
	require.Len(t, got.Points, 7)
	require.Greater(t, got.ChangePercent, 49.9)
	require.Less(t, got.ChangePercent, 50.1)
}

func TestGetStatisticsAPI(t *testing.T) {
	user, _ := randomUser(t)
	user.ID = util.RandomInt(1, 1000)
	user.Uuid = util.RandomUuid()
	date := time.Now().In(time.Local)
	dateString := date.Format("2006-01-02")
	weekStart := startOfISOWeekForTest(date)
	previousWeekStart := weekStart.AddDate(0, 0, -7)
	device := randomIrrigationCommandDevice(user.ID)
	device.IsOnline = true

	historyAction := randomIrrigationHistoryAction(user.ID)
	historyAction.DeviceID = device.ID
	historyAction.StartedAt = date.Add(2 * time.Hour)
	historyAction.CreatedAt = historyAction.StartedAt
	historyAction.FinishedAt = sql.NullTime{Time: historyAction.StartedAt.Add(30 * time.Minute), Valid: true}
	historyAction.WaterVolumeMl = sql.NullInt32{Int32: 500, Valid: true}
	historyAction.Status = "FINALIZADO"

	consumptionAction := randomIrrigationHistoryAction(user.ID)
	consumptionAction.DeviceID = device.ID
	consumptionAction.StartedAt = previousWeekStart.Add(2 * 24 * time.Hour)
	consumptionAction.CreatedAt = consumptionAction.StartedAt
	consumptionAction.FinishedAt = sql.NullTime{Time: consumptionAction.StartedAt.Add(30 * time.Minute), Valid: true}
	consumptionAction.WaterVolumeMl = sql.NullInt32{Int32: 500, Valid: true}
	consumptionAction.Status = "FINALIZADO"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Return(device, nil).Times(2)
	store.EXPECT().ListIrrigationAction(gomock.Any(), gomock.Eq(db.ListIrrigationActionParams{UserID: user.ID, Limit: 100, Offset: 0})).Return([]db.IrrigationAction{historyAction, consumptionAction}, nil).Times(2)
	store.EXPECT().ListDevices(gomock.Any(), gomock.Any()).Return([]db.Device{device}, nil)
	mqttClient.EXPECT().TopicPrefix().Times(0)

	server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store, mqttClient)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/statistics?date="+dateString+"&period=week&device_uuid="+device.Uuid.String(), nil)
	require.NoError(t, err)
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	data, err := io.ReadAll(recorder.Body)
	require.NoError(t, err)

	var got dto.StatisticsResponse
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, dateString, got.Date)
	require.Equal(t, "week", got.Period)
	require.NotNil(t, got.DeviceUUID)
	require.Equal(t, device.Uuid, *got.DeviceUUID)
	require.Len(t, got.IrrigationHistory, 1)
	require.Equal(t, int32(1), got.DeviceStatistics.Total)
	require.Equal(t, int32(1), got.DeviceStatistics.Online)
	require.Equal(t, int64(500), got.WaterConsumption.CurrentTotalMl)
	require.Equal(t, int64(500), got.WaterConsumption.PreviousTotalMl)
}

func TestCreateIrrigationCommandAPI(t *testing.T) {
	user, _ := randomUser(t)
	user.ID = util.RandomInt(1, 1000)
	user.Uuid = util.RandomUuid()
	device := randomIrrigationCommandDevice(user.ID)
	commandUUID := util.RandomUuid()

	testCases := []struct {
		name          string
		body          any
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"device_id": "" + device.Uuid.String(),
				"action":    "START",
				"duration":  300,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient) {
				store.EXPECT().
					GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).
					Times(1).
					Return(device, nil)

				store.EXPECT().
					ExistsPendingIrrigationCommand(gomock.Any(), device.ID).
					Times(1).
					Return(false, nil)

				store.EXPECT().
					ExistsActiveIrrigationAction(gomock.Any(), device.ID).
					Times(1).
					Return(false, nil)

				store.EXPECT().
					CreateIrrigationCommand(
						gomock.Any(),
						gomock.Eq(db.CreateIrrigationCommandParams{
							DeviceID:        device.ID,
							UserID:          user.ID,
							Action:          "START",
							DurationSeconds: sql.NullInt32{Int32: 300, Valid: true},
						}),
					).
					Times(1).
					Return(db.IrrigationCommand{
						ID:       1,
						Uuid:     commandUUID,
						Action:   "START",
						Status:   "PENDING",
						DeviceID: device.ID,
						UserID:   user.ID,
					}, nil)

				store.EXPECT().
					UpdateIrrigationCommandStatus(gomock.Any(), gomock.Any()).
					Times(0)

				expectedPayload, err := json.Marshal(dto.IrrigationCommandPayload{
					CommandID: commandUUID.String(),
					Action:    "START",
					DurationSeconds: func() *int32 {
						v := int32(300)
						return &v
					}(),
				})
				require.NoError(t, err)

				mqttClient.EXPECT().
					TopicPrefix().
					Return("ground-guard")

				mqttClient.EXPECT().
					Publish(
						"ground-guard/devices/"+device.DeviceUid+"/commands",
						expectedPayload,
					).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)

				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var got dto.CreateIrrigationCommandResponse
				require.NoError(t, json.Unmarshal(data, &got))
				require.Equal(t, commandUUID, got.CommandID)
				require.Equal(t, "PENDING", got.Status)
			},
		},
		{
			name: "NotFound",
			body: gin.H{
				"device_id": "" + device.Uuid.String(),
				"action":    "START",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient) {
				store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Times(1).Return(db.Device{}, sql.ErrNoRows)
				store.EXPECT().CreateIrrigationCommand(gomock.Any(), gomock.Any()).Times(0)
				mqttClient.EXPECT().Publish(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "Forbidden",
			body: gin.H{
				"device_id": "" + device.Uuid.String(),
				"action":    "START",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				otherUser, _ := randomUser(t)
				otherUser.ID = user.ID + 1
				otherUser.Uuid = util.RandomUuid()
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, otherUser.Username, otherUser.ID, otherUser.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient) {
				store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Times(1).Return(device, nil)
				store.EXPECT().CreateIrrigationCommand(gomock.Any(), gomock.Any()).Times(0)
				mqttClient.EXPECT().Publish(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "ConflictInactive",
			body: gin.H{
				"device_id": "" + device.Uuid.String(),
				"action":    "START",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient) {
				offlineDevice := device
				offlineDevice.Status = "inativo"
				store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Times(1).Return(offlineDevice, nil)
				store.EXPECT().CreateIrrigationCommand(gomock.Any(), gomock.Any()).Times(0)
				mqttClient.EXPECT().Publish(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "MQTTFailure",
			body: gin.H{
				"device_id": "" + device.Uuid.String(),
				"action":    "START",
				"duration":  300,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				middleware.AddAuthorization(t, request, tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore, mqttClient *mockmqtt.MockClient) {
				store.EXPECT().
					GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).
					Return(device, nil)

				store.EXPECT().
					ExistsPendingIrrigationCommand(gomock.Any(), device.ID).
					Return(false, nil)

				store.EXPECT().
					ExistsActiveIrrigationAction(gomock.Any(), device.ID).
					Return(false, nil)

				store.EXPECT().
					CreateIrrigationCommand(
						gomock.Any(),
						gomock.Eq(db.CreateIrrigationCommandParams{
							DeviceID:        device.ID,
							UserID:          user.ID,
							Action:          "START",
							DurationSeconds: sql.NullInt32{Int32: 300, Valid: true},
						}),
					).
					Return(db.IrrigationCommand{
						ID:       2,
						Uuid:     commandUUID,
						Action:   "START",
						Status:   "PENDING",
						DeviceID: device.ID,
						UserID:   user.ID,
					}, nil)

				store.EXPECT().
					UpdateIrrigationCommandStatus(
						gomock.Any(),
						gomock.Eq(db.UpdateIrrigationCommandStatusParams{
							Uuid:   commandUUID,
							Status: "FAILED",
							ErrorMessage: sql.NullString{
								String: "mqtt publish failed",
								Valid:  true,
							},
						}),
					).
					Return(db.IrrigationCommand{}, nil)

				mqttClient.EXPECT().
					TopicPrefix().
					Return("ground-guard")

				mqttClient.EXPECT().
					Publish(gomock.Any(), gomock.Any()).
					Return(errors.New("mqtt publish failed"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			mqttClient := mockmqtt.NewMockClient(ctrl)
			tc.buildStubs(store, mqttClient)

			server, err := NewServer(util.Config{
				TokenSymmetricKey:   util.RandomString(32),
				AccessTokenDuration: time.Minute,
			}, store, mqttClient)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/api/v1/irrigation/commands", bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func randomIrrigationCommandDevice(userID int64) db.Device {
	return db.Device{
		ID:              util.RandomInt(1, 1000),
		Uuid:            util.RandomUuid(),
		DeviceUid:       util.RandomString(10),
		Name:            util.RandomString(6),
		FirmwareVersion: util.RandomString(8),
		Status:          "ativo",
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
	}
}

func randomIrrigationHistoryAction(userID int64) db.IrrigationAction {
	now := time.Now().UTC()
	return db.IrrigationAction{
		ID:              util.RandomInt(1, 1000),
		Uuid:            util.RandomUuid(),
		DeviceID:        util.RandomInt(1, 1000),
		UserID:          userID,
		StartedAt:       now,
		FinishedAt:      sql.NullTime{Time: now.Add(30 * time.Minute), Valid: true},
		DurationSeconds: sql.NullInt32{Int32: 1800, Valid: true},
		Status:          "FINALIZADO",
		TriggerType:     "MANUAL",
		WaterVolumeMl:   sql.NullInt32{Int32: 600, Valid: true},
		ErrorMessage:    sql.NullString{},
		CreatedAt:       now,
	}
}

func startOfISOWeekForTest(now time.Time) time.Time {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := now.AddDate(0, 0, -(weekday - 1))
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
}
