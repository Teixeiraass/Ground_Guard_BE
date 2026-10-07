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
				store.EXPECT().GetControllerDeviceByUID(gomock.Any(), deviceUID).Return(db.ControllerDevice{}, sql.ErrNoRows)
				store.EXPECT().
					UpdateDeviceTelemetryByUID(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(_ context.Context, arg db.UpdateDeviceTelemetryByUIDParams) (db.Device, error) {
						require.Equal(t, deviceUID, arg.DeviceUid)
						require.Equal(t, "ATIVO", arg.Status)
						require.True(t, arg.LastSeen.Valid)
						return db.Device{ID: 1}, nil
					})
				store.EXPECT().GetDeviceUserIDByUID(gomock.Any(), deviceUID).Return(sql.NullInt64{}, nil)
				store.EXPECT().GetLatestDeviceSensorHistory(gomock.Any(), gomock.Any()).Times(1).Return(db.DeviceSensorHistory{}, sql.ErrNoRows)
				store.EXPECT().CreateDeviceSensorHistory(gomock.Any(), gomock.Any()).Times(1).Return(db.DeviceSensorHistory{}, nil)
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
						return db.Device{ID: 1}, nil
					})
				store.EXPECT().GetControllerDeviceByUID(gomock.Any(), deviceUID).Return(db.ControllerDevice{}, sql.ErrNoRows)
				store.EXPECT().GetDeviceUserIDByUID(gomock.Any(), deviceUID).Return(sql.NullInt64{}, nil)
				store.EXPECT().GetLatestDeviceSensorHistory(gomock.Any(), gomock.Any()).Times(1).Return(db.DeviceSensorHistory{}, sql.ErrNoRows)
				store.EXPECT().CreateDeviceSensorHistory(gomock.Any(), gomock.Any()).Times(1).Return(db.DeviceSensorHistory{}, nil)
			},
		},
		{
			name:    "StoreError",
			topic:   topic,
			payload: payload,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetControllerDeviceByUID(gomock.Any(), deviceUID).Return(db.ControllerDevice{}, sql.ErrNoRows)
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

func TestHandleControllerTelemetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	deviceUID := "246F28F074DC"
	controller := db.ControllerDevice{
		ID:                  10,
		Uuid:                util.RandomUuid(),
		ControllerDeviceUid: deviceUID,
		UserID:              sql.NullInt64{Int64: 42, Valid: true},
	}
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), deviceUID).Return(controller, nil).Times(2)
	store.EXPECT().UpdateControllerDeviceTelemetryByUID(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.UpdateControllerDeviceTelemetryByUIDParams) (db.ControllerDevice, error) {
			require.Equal(t, deviceUID, arg.ControllerDeviceUid)
			require.Equal(t, "192.168.15.5", arg.IpAddress.IPNet.IP.String())
			require.Equal(t, "GroundGuard-WiFi", arg.WifiSsid.String)
			return controller, nil
		},
	)
	store.EXPECT().GetLatestControllerDeviceConfigurationCommand(gomock.Any(), int64(10)).Return(db.ControllerDeviceConfigurationCommand{}, sql.ErrNoRows)

	sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)
	err := sub.HandleTelemetry(
		context.Background(),
		mqtt.DeviceTelemetryTopic("ground-guard", deviceUID),
		[]byte(`{"status":"ONLINE","ip_address":"192.168.15.5","wifi_ssid":"GroundGuard-WiFi","provisioning_required":false}`),
	)
	require.NoError(t, err)
}

func TestHandleControllerTelemetryWithoutUserDoesNotRequireDeviceLookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	deviceUID := "246F28F074DC"
	controller := db.ControllerDevice{
		ID:                  10,
		ControllerDeviceUid: deviceUID,
	}
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), deviceUID).Return(controller, nil).Times(2)
	store.EXPECT().UpdateControllerDeviceTelemetryByUID(gomock.Any(), gomock.Any()).Return(controller, nil)
	store.EXPECT().GetLatestControllerDeviceConfigurationCommand(gomock.Any(), int64(10)).Return(db.ControllerDeviceConfigurationCommand{}, sql.ErrNoRows)

	sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)
	err := sub.HandleTelemetry(
		context.Background(),
		mqtt.DeviceTelemetryTopic("ground-guard", deviceUID),
		[]byte(`{"status":"ONLINE","ip_address":"192.168.15.5","wifi_ssid":"GroundGuard-WiFi"}`),
	)
	require.NoError(t, err)
}
