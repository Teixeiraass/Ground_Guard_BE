package subscriber_test

import (
	"context"
	"encoding/json"
	"testing"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	mockmqtt "github.com/Teixeiraass/ground_guard_be/mqtt/mock"
	"github.com/Teixeiraass/ground_guard_be/mqtt/subscriber"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHandleControllerConfigurationAck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	commandID := uuid.New()
	store.EXPECT().GetControllerDeviceConfigurationCommand(gomock.Any(), commandID).Return(db.ControllerDeviceConfigurationCommand{
		Uuid:               commandID,
		ControllerDeviceID: 3,
		Status:             "SENT",
	}, nil)
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "246F28F074DC").Return(db.ControllerDevice{
		ID:                  3,
		ControllerDeviceUid: "246F28F074DC",
	}, nil)
	store.EXPECT().UpdateControllerDeviceConfigurationStatus(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.UpdateControllerDeviceConfigurationStatusParams) (db.ControllerDeviceConfigurationCommand, error) {
			require.Equal(t, "ACKNOWLEDGED", arg.Status)
			return db.ControllerDeviceConfigurationCommand{Uuid: commandID, Status: arg.Status}, nil
		},
	)

	sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)
	payload, err := json.Marshal(map[string]string{"command_id": commandID.String(), "status": "SUCCESS"})
	require.NoError(t, err)
	err = sub.HandleEvent(context.Background(), mqtt.DeviceEventTopic("ground-guard", "246F28F074DC"), payload)
	require.NoError(t, err)
}

func TestHandleControllerConfigurationAckRejectsOtherDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	commandID := uuid.New()
	store.EXPECT().GetControllerDeviceConfigurationCommand(gomock.Any(), commandID).Return(db.ControllerDeviceConfigurationCommand{
		Uuid:               commandID,
		ControllerDeviceID: 3,
		Status:             "SENT",
	}, nil)
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "OTHER246F28F").Return(db.ControllerDevice{
		ID:                  4,
		ControllerDeviceUid: "OTHER246F28F",
	}, nil)

	sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)
	payload := []byte(`{"command_id":"` + commandID.String() + `","status":"SUCCESS"}`)
	err := sub.HandleEvent(context.Background(), mqtt.DeviceEventTopic("ground-guard", "OTHER246F28F"), payload)
	require.ErrorContains(t, err, "mismatch")
}

func TestHandleControllerConfigurationFailedAck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	commandID := uuid.New()
	store.EXPECT().GetControllerDeviceConfigurationCommand(gomock.Any(), commandID).Return(db.ControllerDeviceConfigurationCommand{
		Uuid:               commandID,
		ControllerDeviceID: 3,
		Status:             "SENT",
	}, nil)
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "246F28F074DC").Return(db.ControllerDevice{
		ID: 3,
	}, nil)
	store.EXPECT().UpdateControllerDeviceConfigurationStatus(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.UpdateControllerDeviceConfigurationStatusParams) (db.ControllerDeviceConfigurationCommand, error) {
			require.Equal(t, "FAILED", arg.Status)
			require.Equal(t, "invalid_configuration", arg.ErrorMessage.String)
			return db.ControllerDeviceConfigurationCommand{}, nil
		},
	)

	sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)
	err := sub.HandleEvent(context.Background(), mqtt.DeviceEventTopic("ground-guard", "246F28F074DC"),
		[]byte(`{"command_id":"`+commandID.String()+`","status":"FAILED","error":"invalid_configuration"}`))
	require.NoError(t, err)
}

func TestHandleControllerConfigurationDuplicateAckIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	commandID := uuid.New()
	store.EXPECT().GetControllerDeviceConfigurationCommand(gomock.Any(), commandID).Return(db.ControllerDeviceConfigurationCommand{
		Uuid:               commandID,
		ControllerDeviceID: 3,
		Status:             "ACKNOWLEDGED",
	}, nil)
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "246F28F074DC").Return(db.ControllerDevice{ID: 3}, nil)

	sub := subscriber.NewTelemetrySubscriber(client.NewNoopClient(), store, nil)
	err := sub.HandleEvent(context.Background(), mqtt.DeviceEventTopic("ground-guard", "246F28F074DC"),
		[]byte(`{"command_id":"`+commandID.String()+`","status":"SUCCESS"}`))
	require.NoError(t, err)
}

func TestConfigurationTopicAndPayload(t *testing.T) {
	require.Equal(t, "ground-guard/devices/246F28F074DC/commands",
		mqtt.DeviceCommandTopic("ground-guard", "246F28F074DC"))
	payload, err := mqtt.NewControllerConfigurationCommand(uuid.New().String(), uuid.New().String(), "America/Sao_Paulo")
	require.NoError(t, err)
	var command mqtt.ControllerConfigurationCommand
	require.NoError(t, json.Unmarshal(payload, &command))
	require.Equal(t, mqtt.ConfigureControllerAction, command.Action)
	require.NotContains(t, string(payload), "wifi_password")
	require.NotContains(t, string(payload), "jwt")
	require.NotContains(t, string(payload), "user_id")
}

func TestRetryPendingControllerConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	commandID := uuid.New()
	controller := db.ControllerDevice{ID: 11, Uuid: uuid.New(), ControllerDeviceUid: "246F28F074DC"}
	command := db.ControllerDeviceConfigurationCommand{
		Uuid:               commandID,
		ControllerDeviceID: controller.ID,
		Action:             mqtt.ConfigureControllerAction,
		Timezone:           "America/Sao_Paulo",
		Status:             "PENDING",
	}
	store.EXPECT().GetControllerDeviceByUID(gomock.Any(), controller.ControllerDeviceUid).Return(controller, nil)
	store.EXPECT().GetLatestControllerDeviceConfigurationCommand(gomock.Any(), controller.ID).Return(command, nil)
	mqttClient.EXPECT().TopicPrefix().Return("ground-guard")
	mqttClient.EXPECT().Publish("ground-guard/devices/246F28F074DC/commands", gomock.Any()).Return(nil)
	sent := command
	sent.Status = "SENT"
	store.EXPECT().MarkControllerDeviceConfigurationAttempt(gomock.Any(), gomock.Any()).Return(sent, nil)

	sub := subscriber.NewTelemetrySubscriber(mqttClient, store, nil)
	require.NoError(t, sub.RetryPendingControllerConfiguration(context.Background(), controller.ControllerDeviceUid))
}
