package subscriber

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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

const maxControllerConfigurationAttempts int32 = 5

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
	Status       string `json:"status"`
}

func (s *TelemetrySubscriber) retryControllerConfiguration(ctx context.Context, deviceUID string) error {
	controller, err := s.store.GetControllerDeviceByUID(ctx, deviceUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	command, err := s.store.GetLatestControllerDeviceConfigurationCommand(ctx, controller.ID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if command.Status != "PENDING" {
		return nil
	}

	payload, err := mqtt.NewControllerConfigurationCommand(
		command.Uuid.String(),
		controller.Uuid.String(),
		command.Timezone,
	)
	if err != nil {
		return err
	}

	status := "SENT"
	var publishErr error
	if _, noop := s.client.(*client.NoopClient); noop {
		publishErr = fmt.Errorf("mqtt client is disabled")
		status = "PENDING"
	} else {
		publishErr = s.client.Publish(mqtt.DeviceCommandTopic(s.client.TopicPrefix(), deviceUID), payload)
		if publishErr != nil {
			status = "PENDING"
		}
	}
	if command.Attempts+1 >= maxControllerConfigurationAttempts && publishErr != nil {
		status = "FAILED"
	}
	nextAttempt := time.Now().Add(time.Duration(1<<minInt32(command.Attempts, 5)) * time.Second)
	_, err = s.store.MarkControllerDeviceConfigurationAttempt(ctx, db.MarkControllerDeviceConfigurationAttemptParams{
		Uuid:          command.Uuid,
		NextAttemptAt: util.ToNullTime(nextAttempt),
		Status:        status,
		ErrorMessage: sql.NullString{
			String: func() string {
				if publishErr != nil {
					return publishErr.Error()
				}
				return ""
			}(),
			Valid: publishErr != nil,
		},
	})
	return err
}

func minInt32(value, max int32) int32 {
	if value < max {
		return value
	}
	return max
}

func (s *TelemetrySubscriber) RetryPendingControllerConfiguration(ctx context.Context, deviceUID string) error {
	return s.retryControllerConfiguration(ctx, deviceUID)
}

func (s *TelemetrySubscriber) RetryAllPendingControllerConfigurations(ctx context.Context) error {
	commands, err := s.store.GetPendingControllerDeviceConfigurationCommands(ctx, maxControllerConfigurationAttempts)
	if err != nil {
		return err
	}
	for _, command := range commands {
		controller, err := s.store.GetControllerDeviceByID(ctx, command.ControllerDeviceID)
		if err != nil {
			return err
		}
		if err := s.retryControllerConfiguration(ctx, controller.ControllerDeviceUid); err != nil {
			return err
		}
	}
	return nil
}

func (s *TelemetrySubscriber) Start() error {
	if reconnectClient, ok := s.client.(client.ReconnectAwareClient); ok {
		reconnectClient.SetOnConnect(func() {
			if err := s.RetryAllPendingControllerConfigurations(context.Background()); err != nil {
				fmt.Printf("mqtt controller configuration retry error: %v\n", err)
			}
		})
	}
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
	log.Printf("mqtt telemetry received topic=%s payload_bytes=%d", topic, len(payload))
	if err := s.HandleTelemetry(context.Background(), topic, payload); err != nil {
		fmt.Printf("mqtt telemetry error on topic %s: %v\n", topic, err)
		return
	}
	log.Printf("mqtt telemetry processed topic=%s", topic)
}

func (s *TelemetrySubscriber) handleEventMessage(topic string, payload []byte) {
	if err := s.HandleEvent(context.Background(), topic, payload); err != nil {
		fmt.Printf("mqtt event error on topic %s: %v\n", topic, err)
	}
}

func (s *TelemetrySubscriber) handleStateMessage(topic string, payload []byte) {
	log.Printf("mqtt state received topic=%s payload_bytes=%d", topic, len(payload))
	if err := s.HandleState(context.Background(), topic, payload); err != nil {
		fmt.Printf("mqtt state error on topic %s: %v\n", topic, err)
		return
	}
	log.Printf("mqtt state processed topic=%s", topic)
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

	controllerCommand, controllerErr := s.store.GetControllerDeviceConfigurationCommand(ctx, commandUUID)
	if controllerErr == nil {
		controller, err := s.store.GetControllerDeviceByUID(ctx, deviceUID)
		if err != nil {
			return fmt.Errorf("failed to get controller device: %w", err)
		}
		if controllerCommand.ControllerDeviceID != controller.ID {
			return fmt.Errorf("controller device mismatch")
		}
		if controllerCommand.Status == "ACKNOWLEDGED" || controllerCommand.Status == "FAILED" {
			return nil
		}
		status := "ACKNOWLEDGED"
		if event.Status == "FAILED" {
			status = "FAILED"
		}
		_, err = s.store.UpdateControllerDeviceConfigurationStatus(ctx, db.UpdateControllerDeviceConfigurationStatusParams{
			Uuid:   commandUUID,
			Status: status,
			ErrorMessage: sql.NullString{
				String: event.Error,
				Valid:  event.Error != "",
			},
		})
		return err
	}
	if controllerErr != sql.ErrNoRows {
		return controllerErr
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
	isOnline := state.IsOnline
	status := state.Status
	if status != "" {
		switch status {
		case mqtt.DeviceStatusOnline, "ONLINE":
			isOnline = true
			status = mqtt.DeviceStatusOnline
		case mqtt.DeviceStatusOffline, "OFFLINE":
			isOnline = false
			status = mqtt.DeviceStatusOffline
		default:
			return fmt.Errorf("invalid state status %s", status)
		}
	} else if isOnline {
		status = mqtt.DeviceStatusOnline
	} else {
		status = mqtt.DeviceStatusOffline
	}

	controller, controllerErr := s.store.GetControllerDeviceByUID(ctx, deviceUID)
	if controllerErr == nil {
		updatedController, updateErr := s.store.UpdateControllerDeviceState(ctx, db.UpdateControllerDeviceStateParams{
			ControllerDeviceUid: deviceUID,
			IsOnline:            isOnline,
		})
		if updateErr != nil {
			return updateErr
		}
		if isOnline {
			if err := s.retryControllerConfiguration(ctx, deviceUID); err != nil {
				fmt.Printf("mqtt controller configuration retry error on state for %s: %v\n", deviceUID, err)
			}
		}
		if controller.UserID.Valid && s.hub != nil {
			s.hub.BroadcastToUser(controller.UserID.Int64, map[string]any{
				"type":            "controller_device_state",
				"device_uid":      deviceUID,
				"controller_uuid": updatedController.Uuid,
				"status":          status,
				"is_irrigating":   state.IsIrrigating,
				"is_online":       isOnline,
			})
		}
		log.Printf("mqtt state controller updated device_uid=%s status=%s is_online=%t websocket_user_id=%d",
			deviceUID, status, isOnline, controller.UserID.Int64)
		return nil
	}
	if controllerErr != sql.ErrNoRows {
		return controllerErr
	}

	_, err := s.store.UpdateDeviceState(ctx, db.UpdateDeviceStateParams{
		DeviceUid:    deviceUID,
		IsIrrigating: state.IsIrrigating,
		IsOnline:     isOnline,
	})
	if err != nil {
		return err
	}

	userID, err := s.store.GetDeviceUserIDByUID(ctx, deviceUID)
	if err != nil || !userID.Valid {
		return nil
	}

	if s.hub != nil {
		s.hub.BroadcastToUser(userID.Int64, map[string]any{
			"type":          "device_state",
			"device_uid":    deviceUID,
			"is_irrigating": state.IsIrrigating,
			"is_online":     isOnline,
		})
	}
	log.Printf("mqtt state device updated device_uid=%s status=%s is_online=%t",
		deviceUID, status, isOnline)

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

	controller, controllerErr := s.store.GetControllerDeviceByUID(ctx, deviceUID)
	if controllerErr == nil {
		updatedController, err := s.store.UpdateControllerDeviceTelemetryByUID(ctx, db.UpdateControllerDeviceTelemetryByUIDParams{
			ControllerDeviceUid: deviceUID,
			LastSeen:            util.ToNullTime(time.Now()),
			IpAddress:           util.ToInet(telemetry.IPAddress),
			WifiSsid:            util.ToNullString(telemetry.WifiSSID),
		})
		if err != nil {
			return fmt.Errorf("update controller device telemetry: %w", err)
		}
		if err := s.retryControllerConfiguration(ctx, deviceUID); err != nil {
			fmt.Printf("mqtt controller configuration retry error on telemetry for %s: %v\n", deviceUID, err)
		}
		if controller.UserID.Valid && s.hub != nil {
			s.hub.BroadcastToUser(controller.UserID.Int64, map[string]any{
				"type":                  "controller_device_telemetry",
				"device_uid":            deviceUID,
				"controller_uuid":       updatedController.Uuid,
				"status":                status,
				"is_online":             true,
				"ip_address":            telemetry.IPAddress,
				"wifi_ssid":             telemetry.WifiSSID,
				"provisioning_required": telemetry.ProvisioningRequired,
				"last_seen":             updatedController.LastSeen.Time,
			})
		}
		log.Printf("mqtt telemetry controller updated device_uid=%s status=%s ip=%s wifi_ssid=%s websocket_user_id=%d",
			deviceUID, status, telemetry.IPAddress, telemetry.WifiSSID, controller.UserID.Int64)
		return nil
	}
	if controllerErr != sql.ErrNoRows {
		return fmt.Errorf("find controller device by uid: %w", controllerErr)
	}

	device, err := s.store.UpdateDeviceTelemetryByUID(ctx, db.UpdateDeviceTelemetryByUIDParams{
		DeviceUid: deviceUID,
		LastSeen:  util.ToNullTime(time.Now()),
		Status:    status,
		SoilMoisture: sql.NullInt32{
			Int32: telemetry.SoilMoisture,
			Valid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("update device telemetry: %w", err)
	}
	latestHistory, err := s.store.GetLatestDeviceSensorHistory(ctx, device.ID)
	shouldInsertHistory := false
	if err != nil {
		if err == sql.ErrNoRows {
			shouldInsertHistory = true
		} else {
			return fmt.Errorf("failed to get latest sensor history: %w", err)
		}
	} else {
		if time.Since(latestHistory.CreatedAt) >= 5*time.Minute {
			shouldInsertHistory = true
		}
	}

	if shouldInsertHistory {
		_, err = s.store.CreateDeviceSensorHistory(ctx, db.CreateDeviceSensorHistoryParams{
			DeviceID:     device.ID,
			SoilMoisture: telemetry.SoilMoisture,
		})
		if err != nil {
			return fmt.Errorf("failed to create sensor history: %w", err)
		}
	}

	userID, err := s.store.GetDeviceUserIDByUID(ctx, deviceUID)
	if err == nil && userID.Valid && s.hub != nil {
		s.hub.BroadcastToUser(userID.Int64, map[string]any{
			"type":          "device_telemetry",
			"device_uid":    deviceUID,
			"soil_moisture": telemetry.SoilMoisture,
			"wifi_ssid":     telemetry.WifiSSID,
			"ip_address":    telemetry.IPAddress,
		})
	}
	log.Printf("mqtt telemetry device updated device_uid=%s status=%s ip=%s wifi_ssid=%s",
		deviceUID, status, telemetry.IPAddress, telemetry.WifiSSID)

	return nil
}
