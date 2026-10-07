package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	mockmqtt "github.com/Teixeiraass/ground_guard_be/mqtt/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestClaimPublishesControllerConfigurationWithoutSecrets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	controller := db.ControllerDevice{
		ID:                  7,
		Uuid:                uuid.New(),
		ControllerDeviceUid: "246F28F074DC",
		UserID:              sql.NullInt64{Int64: 42, Valid: true},
	}
	command := db.ControllerDeviceConfigurationCommand{
		Uuid:               uuid.New(),
		ControllerDeviceID: 7,
		Action:             mqtt.ConfigureControllerAction,
		Timezone:           "America/Sao_Paulo",
		Status:             "PENDING",
	}

	store.EXPECT().ClaimControllerDeviceByUID(gomock.Any(), gomock.Any()).Return(controller, nil)
	store.EXPECT().GetLatestControllerDeviceConfigurationCommand(gomock.Any(), int64(7)).Return(db.ControllerDeviceConfigurationCommand{}, sql.ErrNoRows)
	store.EXPECT().CreateControllerDeviceConfigurationCommand(gomock.Any(), gomock.Any()).Return(command, nil)
	mqttClient.EXPECT().TopicPrefix().Return("ground-guard")
	mqttClient.EXPECT().Publish("ground-guard/devices/246F28F074DC/commands", gomock.Any()).DoAndReturn(
		func(_ string, payload []byte) error {
			var message mqtt.ControllerConfigurationCommand
			require.NoError(t, json.Unmarshal(payload, &message))
			require.Equal(t, command.Uuid.String(), message.CommandID)
			require.Equal(t, mqtt.ConfigureControllerAction, message.Action)
			require.Equal(t, controller.Uuid.String(), message.Data.ControllerUUID)
			require.Equal(t, "America/Sao_Paulo", message.Data.Timezone)
			require.NotContains(t, string(payload), "password")
			require.NotContains(t, string(payload), "jwt")
			require.NotContains(t, string(payload), "user_id")
			return nil
		},
	)
	sentCommand := command
	sentCommand.Status = "SENT"
	store.EXPECT().MarkControllerDeviceConfigurationAttempt(gomock.Any(), gomock.Any()).Return(sentCommand, nil)

	sut := service.NewControllerDeviceService(store, mqttClient)
	got, configuration, err := sut.ClaimControllerDeviceByUID(context.Background(), controller.ControllerDeviceUid, 42)

	require.NoError(t, err)
	require.Equal(t, controller.Uuid, got.Uuid)
	require.Equal(t, "SENT", configuration.Status)
	require.Equal(t, command.Uuid, configuration.CommandID)
}

func TestClaimKeepsPendingWhenMQTTPublishFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	controller := db.ControllerDevice{ID: 8, Uuid: uuid.New(), ControllerDeviceUid: "246F28F074DC"}
	command := db.ControllerDeviceConfigurationCommand{Uuid: uuid.New(), ControllerDeviceID: 8, Action: mqtt.ConfigureControllerAction, Timezone: "America/Sao_Paulo"}

	store.EXPECT().ClaimControllerDeviceByUID(gomock.Any(), gomock.Any()).Return(controller, nil)
	store.EXPECT().GetLatestControllerDeviceConfigurationCommand(gomock.Any(), int64(8)).Return(db.ControllerDeviceConfigurationCommand{}, sql.ErrNoRows)
	store.EXPECT().CreateControllerDeviceConfigurationCommand(gomock.Any(), gomock.Any()).Return(command, nil)
	mqttClient.EXPECT().TopicPrefix().Return("ground-guard")
	mqttClient.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(errors.New("broker unavailable"))
	store.EXPECT().MarkControllerDeviceConfigurationAttempt(gomock.Any(), gomock.Any()).Return(command, nil)

	sut := service.NewControllerDeviceService(store, mqttClient)
	_, configuration, err := sut.ClaimControllerDeviceByUID(context.Background(), controller.ControllerDeviceUid, 42)

	require.NoError(t, err)
	require.Equal(t, "PENDING", configuration.Status)
}

func TestClaimReusesActiveConfigurationCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	mqttClient := mockmqtt.NewMockClient(ctrl)
	controller := db.ControllerDevice{ID: 9, Uuid: uuid.New(), ControllerDeviceUid: "246F28F074DC"}
	command := db.ControllerDeviceConfigurationCommand{Uuid: uuid.New(), ControllerDeviceID: 9, Status: "SENT"}

	store.EXPECT().ClaimControllerDeviceByUID(gomock.Any(), gomock.Any()).Return(controller, nil)
	store.EXPECT().GetLatestControllerDeviceConfigurationCommand(gomock.Any(), int64(9)).Return(command, nil)

	sut := service.NewControllerDeviceService(store, mqttClient)
	_, configuration, err := sut.ClaimControllerDeviceByUID(context.Background(), controller.ControllerDeviceUid, 42)

	require.NoError(t, err)
	require.Equal(t, command.Uuid, configuration.CommandID)
	require.Equal(t, "SENT", configuration.Status)
}
