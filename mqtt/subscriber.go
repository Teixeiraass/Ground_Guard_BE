package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/util"
)

type TelemetrySubscriber struct {
	client Client
	store  db.Store
}

func NewTelemetrySubscriber(client Client, store db.Store) *TelemetrySubscriber {
	return &TelemetrySubscriber{
		client: client,
		store:  store,
	}
}

func (s *TelemetrySubscriber) Start() error {
	topic := DeviceTelemetryWildcard(s.client.TopicPrefix())
	return s.client.Subscribe(topic, s.handleMessage)
}

func (s *TelemetrySubscriber) handleMessage(topic string, payload []byte) {
	if err := s.HandleTelemetry(context.Background(), topic, payload); err != nil {
		fmt.Printf("mqtt telemetry error on topic %s: %v\n", topic, err)
	}
}

func (s *TelemetrySubscriber) HandleTelemetry(ctx context.Context, topic string, payload []byte) error {
	deviceUID, ok := DeviceUIDFromTelemetryTopic(s.client.TopicPrefix(), topic)
	if !ok {
		return fmt.Errorf("invalid telemetry topic: %s", topic)
	}

	var telemetry TelemetryPayload
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return fmt.Errorf("invalid telemetry payload: %w", err)
	}

	status := telemetry.Status
	if status == "" {
		status = DeviceStatusOnline
	}

	_, err := s.store.UpdateDeviceTelemetryByUID(ctx, db.UpdateDeviceTelemetryByUIDParams{
		DeviceUid: deviceUID,
		LastSeen:  util.ToNullTime(time.Now()),
		Status:    status,
		IpAddress: util.ToInet(telemetry.IPAddress),
		WifiSsid:  util.ToNullString(telemetry.WifiSSID),
	})
	if err != nil {
		return fmt.Errorf("update device telemetry: %w", err)
	}

	return nil
}

// PublishCommand sends a command to a device via MQTT.
func PublishCommand(client Client, deviceUID string, payload []byte) error {
	topic := DeviceCommandTopic(client.TopicPrefix(), deviceUID)
	return client.Publish(topic, payload)
}
