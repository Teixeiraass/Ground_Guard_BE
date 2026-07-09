package subscriber

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/google/uuid"
)

type TelemetrySubscriber struct {
	client client.Client
	store  db.Store
	hub    *websocket.Hub
}

type IrrigationEventPayload struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Action    string `json:"action,omitempty"`
}

const (
	IrrigationActionActive   = "ATIVO"
	IrrigationActionFinished = "FINALIZADO"
)

func NewTelemetrySubscriber(
	c client.Client,
	store db.Store,
	hub *websocket.Hub,
) *TelemetrySubscriber {

	return &TelemetrySubscriber{
		client: c,
		store:  store,
		hub:    hub,
	}
}

type DeviceStatePayload struct {
	IsIrrigating bool   `json:"is_irrigating"`
	IsOnline     bool   `json:"is_online"`
}

func (s *TelemetrySubscriber) Start() error {
	if err := s.client.Subscribe(
		mqtt.DeviceTelemetryWildcard(s.client.TopicPrefix()),
		s.handleTelemetryMessage,
	); err != nil {
		return err
	}

	if err := s.client.Subscribe(
		mqtt.DeviceEventWildcard(s.client.TopicPrefix()),
		s.handleEventMessage,
	); err != nil {
		return err
	}

	if err := s.client.Subscribe(
		mqtt.DeviceStateWildcard(s.client.TopicPrefix()),
		s.handleStateMessage,
	); err != nil {
		return err
	}

	return nil
}

func (s *TelemetrySubscriber) handleTelemetryMessage(topic string, payload []byte) {
	if err := s.HandleTelemetry(context.Background(), topic, payload); err != nil {
		fmt.Printf("mqtt telemetry error on topic %s: %v\n", topic, err)
	}
}

func (s *TelemetrySubscriber) handleEventMessage(topic string, payload []byte) {
	if err := s.HandleEvent(context.Background(), topic, payload); err != nil {
		fmt.Printf("mqtt event error on topic %s: %v\n", topic, err)
	}
}

func (s *TelemetrySubscriber) handleStateMessage(topic string, payload []byte) {
	if err := s.HandleState(context.Background(), topic, payload); err != nil {
		fmt.Printf("mqtt state error on topic %s: %v\n", topic, err)
	}
}

func (s *TelemetrySubscriber) HandleEvent(ctx context.Context, topic string, payload []byte) error {
	deviceUID, ok := mqtt.DeviceUIDFromEventTopic(s.client.TopicPrefix(), topic)
	if !ok {
		return fmt.Errorf("invalid event topic: %s", topic)
	}

	var event IrrigationEventPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("invalid event payload: %w", err)
	}

	switch event.Status {
	case "SUCCESS", "FAILED":
	default:
		return fmt.Errorf("invalid status %s", event.Status)
	}

	commandUUID, err := uuid.Parse(event.CommandID)
	if err != nil {
		return fmt.Errorf("invalid command id: %w", err)
	}

	command, err := s.store.GetIrrigationCommand(ctx, commandUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("command not found")
		}
		return err
	}

	if command.Status != "PENDING" {
		return nil
	}

	device, err := s.store.GetDeviceByUID(ctx, deviceUID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	if command.DeviceID != device.ID {
		return fmt.Errorf("device mismatch")
	}

	command, err = s.store.UpdateIrrigationCommandStatus(ctx, db.UpdateIrrigationCommandStatusParams{
		Uuid:   command.Uuid,
		Status: event.Status,
		ErrorMessage: sql.NullString{
			String: event.Error,
			Valid:  event.Error != "",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update command: %w", err)
	}

	if event.Status != "SUCCESS" {
		return nil
	}

	switch command.Action {
	case "START":
		action, err := s.store.CreateIrrigationAction(ctx, db.CreateIrrigationActionParams{
			DeviceID:        device.ID,
			UserID:          command.UserID,
			DurationSeconds: command.DurationSeconds,
			Status:          IrrigationActionActive,
			TriggerType:     "MANUAL",
		})
		if err != nil {
			return fmt.Errorf("failed to create irrigation action: %w", err)
		}

		_, err = s.store.LinkIrrigationCommandAction(ctx, db.LinkIrrigationCommandActionParams{
			ID: command.ID,
			IrrigationActionID: sql.NullInt64{
				Int64: action.ID,
				Valid: true,
			},
		})
		if err != nil {
			return err
		}
	case "STOP":
		action, err := s.store.GetActiveIrrigationActionByDevice(ctx, device.ID)
		if err != nil {
			return err
		}

		_, err = s.store.UpdateIrrigationAction(ctx, db.UpdateIrrigationActionParams{
			Uuid: action.Uuid,
			FinishedAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
			DurationSeconds: sql.NullInt32{
				Int32: int32(time.Since(action.StartedAt).Seconds()),
				Valid: true,
			},
			Status: IrrigationActionFinished,
		})
		if err != nil {
			return err
		}

		_, err = s.store.LinkIrrigationCommandAction(ctx, db.LinkIrrigationCommandActionParams{
			ID: command.ID,
			IrrigationActionID: sql.NullInt64{
				Int64: action.ID,
				Valid: true,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TelemetrySubscriber) HandleState(ctx context.Context, topic string, payload []byte) error {

	deviceUID, ok := mqtt.DeviceUIDFromStateTopic(s.client.TopicPrefix(), topic)
	if !ok {
		return fmt.Errorf("invalid state topic")
	}

	var state DeviceStatePayload

	if err := json.Unmarshal(payload, &state); err != nil {
		return err
	}

	_, err := s.store.UpdateDeviceState(ctx, db.UpdateDeviceStateParams{
		DeviceUid:    deviceUID,
		IsIrrigating: state.IsIrrigating,
		IsOnline:     state.IsOnline,
	})
	if err != nil {
		return err
	}

	device, err := s.store.GetDeviceByUID(ctx, deviceUID)
	if err != nil {
		return err
	}

	s.hub.BroadcastToUser(device.UserID.Int64, map[string]any{
		"type":          "device_state",
		"device_uid":    deviceUID,
		"is_irrigating": state.IsIrrigating,
		"is_online":     state.IsOnline,
	})

	return nil
}

func (s *TelemetrySubscriber) HandleTelemetry(ctx context.Context, topic string, payload []byte) error {
	deviceUID, ok := mqtt.DeviceUIDFromTelemetryTopic(s.client.TopicPrefix(), topic)
	if !ok {
		return fmt.Errorf("invalid telemetry topic: %s", topic)
	}

	var telemetry mqtt.TelemetryPayload
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return fmt.Errorf("invalid telemetry payload: %w", err)
	}

	status := telemetry.Status
	if status == "" {
		status = mqtt.DeviceStatusOnline
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
