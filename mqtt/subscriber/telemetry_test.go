package subscriber_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/mqtt/subscriber"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestHandleTelemetry(t *testing.T) {
	deviceUID := "ESP32-TEST-001"
	topic := mqtt.DeviceTelemetryTopic("ground-guard", deviceUID)

	payload, err := json.Marshal(mqtt.TelemetryPayload{
		Status:    "ATIVO",
		IPAddress: "192.168.0.50",
		WifiSSID:  "GroundGuard-WiFi",
	})
	require.NoError(t, err)

	testCases := []struct {
		name       string
		topic      string
		payload    []byte
		buildStubs func(store *mockdb.MockStore)
		wantErr    bool
	}{
		{
			name:    "OK",
			topic:   topic,
			payload: payload,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateDeviceTelemetryByUID(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(_ context.Context, arg db.UpdateDeviceTelemetryByUIDParams) (db.Device, error) {
						require.Equal(t, deviceUID, arg.DeviceUid)
						require.Equal(t, "ATIVO", arg.Status)
						require.True(t, arg.LastSeen.Valid)
						require.True(t, arg.IpAddress.Valid)
						require.Equal(t, "GroundGuard-WiFi", arg.WifiSsid.String)
						return db.Device{}, nil
					})
			},
		},
		{
			name:    "InvalidTopic",
			topic:   "invalid/topic",
			payload: payload,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateDeviceTelemetryByUID(gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: true,
		},
		{
			name:    "InvalidJSON",
			topic:   topic,
			payload: []byte("{invalid"),
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateDeviceTelemetryByUID(gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: true,
		},
		{
			name:    "DefaultStatusWhenEmpty",
			topic:   topic,
			payload: []byte(`{"ip_address":"10.0.0.1"}`),
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateDeviceTelemetryByUID(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(_ context.Context, arg db.UpdateDeviceTelemetryByUIDParams) (db.Device, error) {
						require.Equal(t, "ATIVO", arg.Status)
						return db.Device{}, nil
					})
			},
		},
		{
			name:    "StoreError",
			topic:   topic,
			payload: payload,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateDeviceTelemetryByUID(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Device{}, sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)
			err := sub.HandleTelemetry(context.Background(), tc.topic, tc.payload)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestTelemetrySubscriberStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)

	require.NoError(t, sub.Start())
}

func TestTelemetryPayloadRoundTrip(t *testing.T) {
	original := mqtt.TelemetryPayload{
		Status:    util.RandomStatus(),
		IPAddress: "192.168.1.10",
		WifiSSID:  util.RandomString(8),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded mqtt.TelemetryPayload
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, original, decoded)
	require.WithinDuration(t, time.Now(), time.Now(), time.Second)
}
