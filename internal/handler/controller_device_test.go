package handler

import (
	"bytes"
	"context"
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
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCreateControllerDeviceAPIRequiresAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE"}, nil)
	store.EXPECT().CreateControllerDevice(gomock.Any(), gomock.Any()).Times(0)
	user, _ := randomUser(t)

	server := newTestServer(t, store)
	body, err := json.Marshal(dto.CreateControllerDeviceRequest{DeviceUid: "AABBCCDDEEFF", Name: "x", FirmwareVersion: "1.0.0"})
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPost, "/api/v1/controller-devices", bytes.NewReader(body))
	require.NoError(t, err)
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	// Anyone creating a controller with an arbitrary UID would own it.
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestCreateControllerDeviceAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE", IsAdmin: true}, nil)
	user, _ := randomUser(t)
	requestBody := dto.CreateControllerDeviceRequest{
		DeviceUid:       "CONTROLLER-001",
		Name:            "Minha controladora",
		FirmwareVersion: "1.0.0",
	}
	created := db.ControllerDevice{
		Uuid:                util.RandomUuid(),
		ControllerDeviceUid: requestBody.DeviceUid,
		Name:                requestBody.Name,
		FirmwareVersion:     requestBody.FirmwareVersion,
		UserID:              nullInt64(user.ID),
	}
	store.EXPECT().CreateControllerDevice(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.CreateControllerDeviceParams) (db.ControllerDevice, error) {
			require.Equal(t, requestBody.DeviceUid, arg.ControllerDeviceUid)
			require.Equal(t, user.ID, arg.UserID.Int64)
			require.True(t, arg.UserID.Valid)
			return created, nil
		},
	)

	server := newTestServer(t, store)
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/controller-devices", bytes.NewReader(body))
	middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	var response dto.ControllerDeviceResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, created.Uuid, response.Uuid)
	require.Equal(t, created.ControllerDeviceUid, response.ControllerDeviceUid)
}

func TestRegisterControllerDeviceAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE"}, nil)
	created := db.ControllerDevice{
		Uuid:                util.RandomUuid(),
		ControllerDeviceUid: "AABBCCDDEEFF",
		Name:                "ground_guard_AABBCCDDEEFF",
		FirmwareVersion:     "1.0.0",
	}
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "AABBCCDDEEFF").Return(db.ControllerDevice{}, sql.ErrNoRows)
	store.EXPECT().CreateControllerDevice(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.CreateControllerDeviceParams) (db.ControllerDevice, error) {
			require.Equal(t, "AABBCCDDEEFF", arg.ControllerDeviceUid)
			require.Equal(t, created.Name, arg.Name)
			require.False(t, arg.UserID.Valid)
			return created, nil
		},
	)

	server := newTestServer(t, store)
	body, err := json.Marshal(dto.RegisterControllerDeviceRequest{
		DeviceUid:       "aabbccddeeff",
		FirmwareVersion: "1.0.0",
		FirmwareBuild:   "1",
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/controller-devices/register", bytes.NewReader(body)))

	require.Equal(t, http.StatusOK, recorder.Code)
	var response dto.ControllerDeviceResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, created.Name, response.Name)
	require.Equal(t, created.ControllerDeviceUid, response.ControllerDeviceUid)
	require.NotContains(t, recorder.Body.String(), "wifi_password")
}

func TestRegisterControllerDeviceKeepsUnchangedExistingDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE"}, nil)
	existing := db.ControllerDevice{
		Uuid:                util.RandomUuid(),
		ControllerDeviceUid: "AABBCCDDEEFF",
		Name:                "existing",
		FirmwareVersion:     "1.0.0",
		FirmwareBuild:       sql.NullString{String: "2", Valid: true},
	}
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), existing.ControllerDeviceUid).Return(existing, nil)
	store.EXPECT().UpdateControllerDeviceFirmware(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	body, err := json.Marshal(dto.RegisterControllerDeviceRequest{
		DeviceUid:       existing.ControllerDeviceUid,
		FirmwareVersion: "1.0.0",
		FirmwareBuild:   "2",
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/controller-devices/register", bytes.NewReader(body)))

	require.Equal(t, http.StatusOK, recorder.Code)
	var response dto.ControllerDeviceResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, existing.Name, response.Name)
}

func TestRegisterControllerDeviceUpdatesFirmwareVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE"}, nil)
	existing := db.ControllerDevice{
		ID:                  7,
		Uuid:                util.RandomUuid(),
		ControllerDeviceUid: "AABBCCDDEEFF",
		Name:                "existing",
		FirmwareVersion:     "0.9.0",
	}
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), existing.ControllerDeviceUid).Return(existing, nil)
	store.EXPECT().
		UpdateControllerDeviceFirmware(gomock.Any(), db.UpdateControllerDeviceFirmwareParams{
			ID:              7,
			FirmwareVersion: "1.0.0",
			FirmwareBuild:   sql.NullString{String: "2", Valid: true},
		}).
		DoAndReturn(func(_ context.Context, arg db.UpdateControllerDeviceFirmwareParams) (db.ControllerDevice, error) {
			updated := existing
			updated.FirmwareVersion = arg.FirmwareVersion
			updated.FirmwareBuild = arg.FirmwareBuild
			return updated, nil
		})

	server := newTestServer(t, store)
	body, err := json.Marshal(dto.RegisterControllerDeviceRequest{
		DeviceUid:       existing.ControllerDeviceUid,
		FirmwareVersion: "1.0.0",
		FirmwareBuild:   "2",
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/controller-devices/register", bytes.NewReader(body)))

	require.Equal(t, http.StatusOK, recorder.Code)
	var response dto.ControllerDeviceResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "1.0.0", response.FirmwareVersion)
	require.Equal(t, existing.Name, response.Name)
}

func TestClaimControllerDeviceByUIDAPI(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		claimed        db.ControllerDevice
		queryError     error
		existing       db.ControllerDevice
		existingError  error
		expectedStatus int
	}{
		{
			name:   "claims unowned device",
			userID: 10,
			claimed: db.ControllerDevice{
				ID:                  1,
				Uuid:                util.RandomUuid(),
				ControllerDeviceUid: "AABBCCDDEEFF",
				UserID:              nullInt64(10),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "is idempotent for owner",
			userID: 10,
			claimed: db.ControllerDevice{
				ID:                  1,
				Uuid:                util.RandomUuid(),
				ControllerDeviceUid: "AABBCCDDEEFF",
				UserID:              nullInt64(10),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "rejects another owner",
			userID:     10,
			queryError: sql.ErrNoRows,
			existing: db.ControllerDevice{
				ControllerDeviceUid: "AABBCCDDEEFF",
				UserID:              nullInt64(20),
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE"}, nil)
			if test.queryError == nil {
				store.EXPECT().ClaimControllerDeviceByUID(gomock.Any(), gomock.Any()).Return(test.claimed, nil)
				store.EXPECT().GetLatestControllerDeviceConfigurationCommand(gomock.Any(), int64(1)).Return(db.ControllerDeviceConfigurationCommand{}, sql.ErrNoRows)
				store.EXPECT().CreateControllerDeviceConfigurationCommand(gomock.Any(), gomock.Any()).Return(db.ControllerDeviceConfigurationCommand{
					Uuid:               util.RandomUuid(),
					ControllerDeviceID: 1,
					Action:             "CONFIGURE_CONTROLLER",
					Timezone:           "America/Sao_Paulo",
					Status:             "PENDING",
				}, nil)
				store.EXPECT().MarkControllerDeviceConfigurationAttempt(gomock.Any(), gomock.Any()).Return(db.ControllerDeviceConfigurationCommand{}, nil)
			} else {
				store.EXPECT().ClaimControllerDeviceByUID(gomock.Any(), gomock.Any()).Return(db.ControllerDevice{}, test.queryError)
				store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "AABBCCDDEEFF").Return(test.existing, test.existingError)
			}

			server := newTestServer(t, store)
			request := httptest.NewRequest(http.MethodPut, "/api/v1/controller-devices/claim/AABBCCDDEEFF", nil)
			middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, "user", test.userID, util.RandomUuid(), time.Minute)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)

			require.Equal(t, test.expectedStatus, recorder.Code)
		})
	}
}

func TestControllerDeviceEndpointsValidateUIDAndAuthentication(t *testing.T) {
	t.Run("invalid uid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE"}, nil)
		server := newTestServer(t, store)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/controller-devices/claim/not-a-uid", nil)
		middleware.AddAuthorization(t, request, server.tokenMaker, middleware.AuthorizationTypeBearer, "user", 1, util.RandomUuid(), time.Minute)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("missing authentication", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().GetUser(gomock.Any(), gomock.Any()).AnyTimes().Return(db.User{Status: "ACTIVE"}, nil)
		server := newTestServer(t, store)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/controller-devices/claim/AABBCCDDEEFF", nil)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
	})
}

func nullInt64(value int64) (result sql.NullInt64) {
	result.Int64 = value
	result.Valid = true
	return result
}
