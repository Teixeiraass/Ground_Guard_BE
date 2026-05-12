package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestGetDeviceAPI(t *testing.T) {
	user, _ := randomUser(t)
	device := randomDevice(user.ID)

	testCases := []struct {
		name string
		deviceUuid string
		setupAuth func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			deviceUuid: device.Uuid.String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Times(1).Return(device, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchDevice(t, recorder.Body, device)
			},
		},
		{
			name:      "NotFound",
			deviceUuid: device.Uuid.String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// build stubs
				store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Times(1).Return(db.Device{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				//check response
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			deviceUuid: device.Uuid.String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// build stubs
				store.EXPECT().GetDevice(gomock.Any(), gomock.Eq(device.Uuid)).Times(1).Return(db.Device{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				//check response
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "InvalidUUID",
			deviceUuid: "invalidUUID",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, user.ID, user.Uuid, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// build stubs
				store.EXPECT().GetDevice(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				//check response
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			//start test server and send request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/devices/%s", tc.deviceUuid)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func randomDevice(userID int64) db.Device {
	return db.Device{
		ID: util.RandomInt(1, 1000),
		Uuid: util.RandomUuid(),
		DeviceUid: util.RandomString(10),
		Name: util.RandomString(6),
		FirmwareVersion: util.RandomString(8),
		Status: util.RandomStatus(),
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
	}
}

func requireBodyMatchDevice(t *testing.T, body *bytes.Buffer, device db.Device) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gotDevice db.Device
	err = json.Unmarshal(data, &gotDevice)
	require.NoError(t, err)
	require.Equal(t, device, gotDevice)
}