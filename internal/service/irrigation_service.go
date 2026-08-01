package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/mqtt/publisher"
	"github.com/google/uuid"
)

type IrrigationService interface {
	CreateIrrigationCommand(ctx context.Context, req dto.CreateIrrigationCommandRequest, userID int64) (*db.IrrigationCommand, error)
	GetIrrigationCommand(ctx context.Context, commandUUID uuid.UUID) (*db.IrrigationCommand, error)
	CreateIrrigationPreference(ctx context.Context, req dto.CreateIrrigationPreferenceRequest) (*db.IrrigationPreference, error)
	GetIrrigationPreference(ctx context.Context, preferenceUUID uuid.UUID) (*db.IrrigationPreference, error)
	GetIrrigationPreferenceByDevice(ctx context.Context, deviceUUID uuid.UUID) (*db.IrrigationPreference, error)
}

type irrigationService struct {
	store      db.Store
	mqttClient client.Client
}

func NewIrrigationService(store db.Store, mqttClient client.Client) IrrigationService {
	return &irrigationService{
		store:      store,
		mqttClient: mqttClient,
	}
}

func (s *irrigationService) CreateIrrigationCommand(ctx context.Context, req dto.CreateIrrigationCommandRequest, userID int64) (*db.IrrigationCommand, error) {
	deviceUUID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	if !device.UserID.Valid || device.UserID.Int64 != userID {
		return nil, errors.New("device doesn't belong to authenticated user")
	}

	if !strings.EqualFold(strings.TrimSpace(device.Status), "ativo") {
		return nil, errors.New("device is inactive, cannot send command")
	}

	pending, err := s.store.ExistsPendingIrrigationCommand(ctx, device.ID)
	if err != nil {
		return nil, err
	}

	if pending {
		return nil, errors.New("device already has a pending command")
	}

	active, err := s.store.ExistsActiveIrrigationAction(ctx, device.ID)
	if err != nil {
		return nil, err
	}

	switch req.Action {
	case "START":
		if active {
			return nil, errors.New("device is already irrigating")
		}
	case "STOP":
		if !active {
			return nil, errors.New("device is not irrigating")
		}
	}

	var durationSeconds sql.NullInt32
	if req.Duration != nil {
		durationSeconds = sql.NullInt32{Int32: *req.Duration, Valid: true}
	}

	command, err := s.store.CreateIrrigationCommand(ctx, db.CreateIrrigationCommandParams{
		DeviceID:        device.ID,
		UserID:          userID,
		Action:          req.Action,
		DurationSeconds: durationSeconds,
	})
	if err != nil {
		return nil, err
	}

	payload := dto.IrrigationCommandPayload{
		CommandID: command.Uuid.String(),
		Action:    command.Action,
	}
	if durationSeconds.Valid {
		payload.DurationSeconds = &durationSeconds.Int32
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := publisher.PublishCommand(s.mqttClient, device.DeviceUid, data); err != nil {
		_, updateErr := s.store.UpdateIrrigationCommandStatus(context.Background(), db.UpdateIrrigationCommandStatusParams{
			Uuid:   command.Uuid,
			Status: "FAILED",
			ErrorMessage: sql.NullString{
				String: err.Error(),
				Valid:  true,
			},
		})
		if updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}

	return &command, nil
}

func (s *irrigationService) GetIrrigationCommand(ctx context.Context, commandUUID uuid.UUID) (*db.IrrigationCommand, error) {
	command, err := s.store.GetIrrigationCommand(ctx, commandUUID)
	if err != nil {
		return nil, err
	}
	return &command, nil
}

func (s *irrigationService) CreateIrrigationPreference(ctx context.Context, req dto.CreateIrrigationPreferenceRequest) (*db.IrrigationPreference, error) {
	deviceUUID, err := uuid.Parse(req.DeviceUUID)
	if err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	var startHour sql.NullTime
	if req.StartHour != nil {
		t, err := time.Parse("15:04", *req.StartHour)
		if err != nil {
			return nil, err
		}
		startHour = sql.NullTime{Time: t, Valid: true}
	}

	var endHour sql.NullTime
	if req.EndHour != nil {
		t, err := time.Parse("15:04", *req.EndHour)
		if err != nil {
			return nil, err
		}
		endHour = sql.NullTime{Time: t, Valid: true}
	}

	arg := db.CreateIrrigationPreferencesParams{
		DeviceID:             device.ID,
		IrrigationMode:       req.IrrigationMode,
		MoistureThreshold:    req.MoistureThreshold,
		DryTimeMinutes:       req.DryTimeMinutes,
		MaxIrrigationsPerDay: req.MaxIrrigationsPerDay,
		StartHour:            startHour,
		EndHour:              endHour,
	}

	preference, err := s.store.CreateIrrigationPreferences(ctx, arg)
	if err != nil {
		return nil, err
	}

	return &preference, nil
}

func (s *irrigationService) GetIrrigationPreference(ctx context.Context, preferenceUUID uuid.UUID) (*db.IrrigationPreference, error) {
	preference, err := s.store.GetIrrigationPreference(ctx, preferenceUUID)
	if err != nil {
		return nil, err
	}
	return &preference, nil
}

func (s *irrigationService) GetIrrigationPreferenceByDevice(ctx context.Context, deviceUUID uuid.UUID) (*db.IrrigationPreference, error) {
	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	preference, err := s.store.GetIrrigationPreferenceByDevice(ctx, device.ID)
	if err != nil {
		return nil, err
	}
	return &preference, nil
}
