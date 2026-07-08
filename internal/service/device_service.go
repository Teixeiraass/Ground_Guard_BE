package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type DeviceService interface {
	CreateDevice(ctx context.Context, req dto.CreateDeviceRequest) (*db.Device, error)
	RegisterDevice(ctx context.Context, req dto.CreateDeviceRequest) (*db.Device, error)
	GetDevice(ctx context.Context, deviceUUID uuid.UUID, userID int64) (*db.Device, error)
	ListDevices(ctx context.Context, userID int64, limit int32, offset int32) ([]db.Device, error)
	GetDeviceByUID(ctx context.Context, uid string) (*db.Device, error)
	LinkDeviceToUserByQrToken(ctx context.Context, qrToken string, userID int64) (*db.Device, error)
	UnlinkDeviceFromUser(ctx context.Context, deviceUUID uuid.UUID, userID int64) (*db.Device, error)
	UpdateNameDevice(ctx context.Context, deviceUUID uuid.UUID, name string) (*db.Device, error)
}

type deviceService struct {
	store db.Store
}

func NewDeviceService(store db.Store) DeviceService {
	return &deviceService{store: store}
}

func (s *deviceService) createDevice(ctx context.Context, req dto.CreateDeviceRequest) (*db.Device, error) {
	qrToken, err :=util.GenerateQRToken(12)
	if err != nil {
		return nil, err
	}

	qrFileName, err := util.GenerateQRCodeImage(qrToken)
	if err != nil {
		return nil, err
	}

	arg := db.CreateDeviceParams{
		DeviceUid:       req.DeviceUid,
		Name:            req.Name,
		FirmwareVersion: req.FirmwareVersion,
		FirmwareBuild:   util.ToNullString(req.FirmwareBuild),
		IpAddress:       util.ToInet(req.IpAddress),
		WifiSsid:        util.ToNullString(req.WifiSsid),
		Status:          req.Status,
		LastSeen: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		QrToken:     qrToken,
		QrCodeFile:  util.ToNullString(qrFileName),
	}

	device, err := s.store.CreateDevice(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			return nil, errors.New("unique_violation")
		}
		return nil, err
	}

	return &device, nil
}

func (s *deviceService) CreateDevice(ctx context.Context, req dto.CreateDeviceRequest) (*db.Device, error) {
	return s.createDevice(ctx, req)
}

func (s *deviceService) RegisterDevice(ctx context.Context, req dto.CreateDeviceRequest) (*db.Device, error) {
	_, err := s.store.GetDeviceByUID(ctx, req.DeviceUid)

	if err == sql.ErrNoRows {
		return s.createDevice(ctx, req)
	}

	if err != nil {
		return nil, err
	}

	device, err := s.store.UpdateDeviceRegistration(ctx, db.UpdateDeviceRegistrationParams{
		DeviceUid:       req.DeviceUid,
		FirmwareVersion: req.FirmwareVersion,
		FirmwareBuild:   util.ToNullString(req.FirmwareBuild),
		IpAddress:       util.ToInet(req.IpAddress),
		WifiSsid:        util.ToNullString(req.WifiSsid),
		Status:          req.Status,
	})

	if err != nil {
		return nil, err
	}

	return &device, nil
}

func (s *deviceService) GetDevice(ctx context.Context, deviceUUID uuid.UUID, userID int64) (*db.Device, error) {
	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	if device.UserID.Int64 != userID {
		return nil, errors.New("device doesn't belong to authenticated user")
	}

	return &device, nil
}

func (s *deviceService) ListDevices(ctx context.Context, userID int64, limit int32, offset int32) ([]db.Device, error) {
	arg := db.ListDevicesParams{
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
		Limit:  limit,
		Offset: offset,
	}

	return s.store.ListDevices(ctx, arg)
}

func (s *deviceService) GetDeviceByUID(ctx context.Context, uid string) (*db.Device, error) {
	device, err := s.store.GetDeviceByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (s *deviceService) LinkDeviceToUserByQrToken(ctx context.Context, qrToken string, userID int64) (*db.Device, error) {
	arg := db.LinkDeviceToUserByQrTokenParams{
		QrToken: qrToken,
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
	}

	device, err := s.store.LinkDeviceToUserByQrToken(ctx, arg)
	if err != nil {
		return nil, err
	}

	return &device, nil
}

func (s *deviceService) UnlinkDeviceFromUser(ctx context.Context, deviceUUID uuid.UUID, userID int64) (*db.Device, error) {
	arg := db.UnlinkDeviceFromUserParams{
		Uuid: deviceUUID,
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
	}

	device, err := s.store.UnlinkDeviceFromUser(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (s *deviceService) UpdateNameDevice(ctx context.Context, deviceUUID uuid.UUID, name string) (*db.Device, error) {
	arg := db.UpdateNameDeviceParams{
		Uuid: deviceUUID,
		Name: name,
	}

	device, err := s.store.UpdateNameDevice(ctx, arg)
	if err != nil {
		return nil, err
	}

	return &device, nil
}
