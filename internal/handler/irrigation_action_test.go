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
