package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/push"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func TestGetAppVersionAPI(t *testing.T) {
	store := mockdb.NewMockStore(nil)
	config := util.Config{
		TokenSymmetricKey: util.RandomString(32),
		MinimumAppVersion: "1.2.0",
		LatestAppVersion:  "1.4.0",
		AndroidStoreUrl:   "https://play.google.com/store/apps/details?id=com.groundguard.app",
		IosStoreUrl:       "https://apps.apple.com/app/id000000000",
	}
	server, err := NewServer(config, store, client.NewNoopClient(), push.NewNoopSender())
	require.NoError(t, err)

	request, err := http.NewRequest(http.MethodGet, "/api/v1/app-version", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response dto.AppVersionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, config.MinimumAppVersion, response.MinimumVersion)
	require.Equal(t, config.LatestAppVersion, response.LatestVersion)
	require.Equal(t, config.AndroidStoreUrl, response.AndroidStoreUrl)
	require.Equal(t, config.IosStoreUrl, response.IosStoreUrl)
}

// Esse endpoint é público (sem auth) — confirma que não exige Bearer token.
func TestGetAppVersionAPIDoesNotRequireAuth(t *testing.T) {
	store := mockdb.NewMockStore(nil)
	server := newTestServer(t, store)

	request, err := http.NewRequest(http.MethodGet, "/api/v1/app-version", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
}
