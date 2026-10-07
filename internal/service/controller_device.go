package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/mqtt/publisher"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	ErrInvalidControllerDeviceUID = errors.New("device_uid must contain exactly 12 hexadecimal characters")
	ErrControllerDeviceConflict   = errors.New("controller device is already linked to another user")
)

var controllerDeviceUIDPattern = regexp.MustCompile(`^[0-9A-Fa-f]{12}$`)

type ControllerDeviceService interface {
	CreateControllerDevice(ctx context.Context, req dto.CreateControllerDeviceRequest, userID int64) (*db.ControllerDevice, error)
	RegisterControllerDevice(ctx context.Context, req dto.RegisterControllerDeviceRequest) (*db.ControllerDevice, error)
	GetControllerDevice(ctx context.Context, controllerDeviceUUID uuid.UUID, userID int64) (*db.ControllerDevice, error)
	ListControllerDevices(ctx context.Context, userID int64, limit int32, offset int32) ([]db.ControllerDevice, error)
	GetControllerDeviceByUID(ctx context.Context, uid string) (*db.ControllerDevice, error)
	ClaimControllerDeviceByUID(ctx context.Context, uid string, userID int64) (*db.ControllerDevice, *dto.ControllerDeviceConfigurationResponse, error)
	UnlinkControllerDeviceFromUser(ctx context.Context, controllerDeviceUUID uuid.UUID, userID int64) (*db.ControllerDevice, error)
	UpdateNameControllerDevice(ctx context.Context, controllerDeviceUUID uuid.UUID, name string) (*db.ControllerDevice, error)
}

type controllerDeviceService struct {
	store      db.Store
	mqttClient client.Client
}

func NewControllerDeviceService(store db.Store, mqttClients ...client.Client) ControllerDeviceService {
	var mqttClient client.Client = client.NewNoopClient()
	if len(mqttClients) > 0 && mqttClients[0] != nil {
		mqttClient = mqttClients[0]
	}
	return &controllerDeviceService{store: store, mqttClient: mqttClient}
}

func (s *controllerDeviceService) createControllerDevice(ctx context.Context, req dto.CreateControllerDeviceRequest, userID sql.NullInt64) (*db.ControllerDevice, error) {
	arg := db.CreateControllerDeviceParams{
		ControllerDeviceUid: req.DeviceUid,
		Name:                req.Name,
		FirmwareVersion:     req.FirmwareVersion,
		FirmwareBuild:       util.ToNullString(req.FirmwareBuild),
		IpAddress:           util.ToInet(req.IpAddress),
		WifiSsid:            util.ToNullString(req.WifiSsid),
		LastSeen: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		UserID: userID,
	}

	controllerDevice, err := s.store.CreateControllerDevice(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			return nil, errors.New("unique_violation")
		}
		return nil, err
	}

	return &controllerDevice, nil
}

func (s *controllerDeviceService) CreateControllerDevice(ctx context.Context, req dto.CreateControllerDeviceRequest, userID int64) (*db.ControllerDevice, error) {
	uid := strings.ToUpper(strings.TrimSpace(req.DeviceUid))
	if controllerDeviceUIDPattern.MatchString(uid) {
		req.DeviceUid = uid
	}
	return s.createControllerDevice(ctx, req, sql.NullInt64{Int64: userID, Valid: true})
}

func normalizeControllerDeviceUID(uid string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(uid))
	if !controllerDeviceUIDPattern.MatchString(normalized) {
		return "", ErrInvalidControllerDeviceUID
	}
	return normalized, nil
}

func (s *controllerDeviceService) RegisterControllerDevice(ctx context.Context, req dto.RegisterControllerDeviceRequest) (*db.ControllerDevice, error) {
	uid, err := normalizeControllerDeviceUID(req.DeviceUid)
	if err != nil {
		return nil, err
	}
	existing, err := s.store.GetControllerDeviceByUID(ctx, uid)
	if err == nil {
		return &existing, nil
	}
	if err == sql.ErrNoRows {
		name := req.Name
		if name == "" {
			name = fmt.Sprintf("ground_guard_%s", uid)
		}
		controllerDevice, createErr := s.createControllerDevice(ctx, dto.CreateControllerDeviceRequest{
			DeviceUid:       uid,
			Name:            name,
			FirmwareVersion: req.FirmwareVersion,
			FirmwareBuild:   req.FirmwareBuild,
		}, sql.NullInt64{})
		if createErr != nil && createErr.Error() == "unique_violation" {
			existing, getErr := s.store.GetControllerDeviceByUID(ctx, uid)
			if getErr != nil {
				return nil, getErr
			}
			return &existing, nil
		}
		return controllerDevice, createErr
	}
	return nil, err
}

func (s *controllerDeviceService) GetControllerDevice(ctx context.Context, deviceUUID uuid.UUID, userID int64) (*db.ControllerDevice, error) {
	controllerDevice, err := s.store.GetControllerDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	if controllerDevice.UserID.Int64 != userID {
		return nil, errors.New("device doesn't belong to authenticated user")
	}

	return &controllerDevice, nil
}

func (s *controllerDeviceService) ListControllerDevices(ctx context.Context, userID int64, limit int32, offset int32) ([]db.ControllerDevice, error) {
	arg := db.ListControllerDevicesParams{
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
		Limit:  limit,
		Offset: offset,
	}

	return s.store.ListControllerDevices(ctx, arg)
}

func (s *controllerDeviceService) GetControllerDeviceByUID(ctx context.Context, uid string) (*db.ControllerDevice, error) {
	normalizedUID, err := normalizeControllerDeviceUID(uid)
	if err != nil {
		return nil, err
	}
	controllerDevice, err := s.store.GetControllerDeviceByUID(ctx, normalizedUID)
	if err != nil {
		return nil, err
	}
	return &controllerDevice, nil
}

func (s *controllerDeviceService) ClaimControllerDeviceByUID(ctx context.Context, uid string, userID int64) (*db.ControllerDevice, *dto.ControllerDeviceConfigurationResponse, error) {
	normalizedUID, err := normalizeControllerDeviceUID(uid)
	if err != nil {
		return nil, nil, err
	}
	controllerDevice, err := s.store.ClaimControllerDeviceByUID(ctx, db.ClaimControllerDeviceByUIDParams{
		ControllerDeviceUid: normalizedUID,
		UserID:              sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			existing, getErr := s.store.GetControllerDeviceByUID(ctx, normalizedUID)
			if getErr == sql.ErrNoRows {
				return nil, nil, getErr
			}
			if getErr != nil {
				return nil, nil, getErr
			}
			if existing.UserID.Valid && existing.UserID.Int64 != userID {
				return nil, nil, ErrControllerDeviceConflict
			}
			controllerDevice = existing
		} else {
			return nil, nil, err
		}
	}

	configuration, err := s.ensureControllerConfiguration(ctx, controllerDevice)
	if err != nil {
		return nil, nil, err
	}
	return &controllerDevice, configuration, nil
}

func (s *controllerDeviceService) ensureControllerConfiguration(ctx context.Context, controllerDevice db.ControllerDevice) (*dto.ControllerDeviceConfigurationResponse, error) {
	command, err := s.store.GetLatestControllerDeviceConfigurationCommand(ctx, controllerDevice.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && command.Status != "FAILED" {
		return &dto.ControllerDeviceConfigurationResponse{Status: command.Status, CommandID: command.Uuid}, nil
	}

	command, err = s.store.CreateControllerDeviceConfigurationCommand(ctx, db.CreateControllerDeviceConfigurationCommandParams{
		ControllerDeviceID: controllerDevice.ID,
		Action:             mqtt.ConfigureControllerAction,
		Timezone:           "America/Sao_Paulo",
	})
	if err != nil {
		if strings.Contains(err.Error(), "unique_violation") {
			command, getErr := s.store.GetLatestControllerDeviceConfigurationCommand(ctx, controllerDevice.ID)
			if getErr != nil {
				return nil, getErr
			}
			return &dto.ControllerDeviceConfigurationResponse{Status: command.Status, CommandID: command.Uuid}, nil
		}
		return nil, err
	}

	status := "PENDING"
	var publishErr error
	if _, noop := s.mqttClient.(*client.NoopClient); noop {
		publishErr = errors.New("mqtt client is disabled")
	} else {
		payload, marshalErr := mqtt.NewControllerConfigurationCommand(command.Uuid.String(), controllerDevice.Uuid.String(), command.Timezone)
		if marshalErr != nil {
			publishErr = marshalErr
		} else {
			publishErr = publisher.PublishCommand(s.mqttClient, controllerDevice.ControllerDeviceUid, payload)
		}
	}
	if publishErr == nil {
		status = "SENT"
	} else {
		_, updateErr := s.store.MarkControllerDeviceConfigurationAttempt(ctx, db.MarkControllerDeviceConfigurationAttemptParams{
			Uuid:          command.Uuid,
			NextAttemptAt: sql.NullTime{Time: time.Now().Add(time.Minute), Valid: true},
			Status:        status,
			ErrorMessage:  sql.NullString{String: publishErr.Error(), Valid: true},
		})
		if updateErr != nil {
			return nil, updateErr
		}
		return &dto.ControllerDeviceConfigurationResponse{Status: status, CommandID: command.Uuid}, nil
	}

	command, err = s.store.MarkControllerDeviceConfigurationAttempt(ctx, db.MarkControllerDeviceConfigurationAttemptParams{
		Uuid:          command.Uuid,
		NextAttemptAt: sql.NullTime{Time: time.Now().Add(time.Minute), Valid: true},
		Status:        status,
	})
	if err != nil {
		return nil, err
	}
	return &dto.ControllerDeviceConfigurationResponse{Status: command.Status, CommandID: command.Uuid}, nil
}

func (s *controllerDeviceService) UnlinkControllerDeviceFromUser(ctx context.Context, deviceUUID uuid.UUID, userID int64) (*db.ControllerDevice, error) {
	arg := db.UnlinkControllerDeviceFromUserParams{
		Uuid: deviceUUID,
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
	}

	controllerDevice, err := s.store.UnlinkControllerDeviceFromUser(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &controllerDevice, nil
}

func (s *controllerDeviceService) UpdateNameControllerDevice(ctx context.Context, deviceUUID uuid.UUID, name string) (*db.ControllerDevice, error) {
	arg := db.UpdateNameControllerDeviceParams{
		Uuid: deviceUUID,
		Name: name,
	}

	controllerDevice, err := s.store.UpdateNameControllerDevice(ctx, arg)
	if err != nil {
		return nil, err
	}

	return &controllerDevice, nil
}
