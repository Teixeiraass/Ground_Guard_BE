package mqtt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/google/uuid"
)

type TelemetrySubscriber struct {
	client Client
	store  db.Store
}

type IrrigationEventPayload struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Action    string `json:"action,omitempty"`
}

const (
    IrrigationActionActive = "ATIVO"
    IrrigationActionFinished = "FINALIZADO"
)

func NewTelemetrySubscriber(client Client, store db.Store) *TelemetrySubscriber {
	return &TelemetrySubscriber{
		client: client,
		store:  store,
	}
}

func (s *TelemetrySubscriber) Start() error {
    if err := s.client.Subscribe(
        DeviceTelemetryWildcard(s.client.TopicPrefix()),
        s.handleTelemetryMessage,
    ); err != nil {
        return err
    }

    if err := s.client.Subscribe(
        DeviceEventWildcard(s.client.TopicPrefix()),
        s.handleEventMessage,
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


func (s *TelemetrySubscriber) HandleEvent(ctx context.Context, topic string, payload []byte) error {
	deviceUID, ok := DeviceUIDFromEventTopic(s.client.TopicPrefix(), topic)
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
			DeviceID: device.ID,
			UserID: command.UserID,
			DurationSeconds: command.DurationSeconds,
			Status: IrrigationActionActive,
			TriggerType: "MANUAL",
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
				Time: time.Now(),
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