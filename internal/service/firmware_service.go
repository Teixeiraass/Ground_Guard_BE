package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/google/uuid"
)

const (
	NotificationTypeFirmwareUpdate = "FIRMWARE_UPDATE"

	// The controller's OTA partition (min_spiffs.csv) holds 1.9 MB.
	MaxFirmwareSize int64 = 0x1E0000

	// ESP32 application images always start with this magic byte.
	esp32ImageMagic byte = 0xE9

	otaTimeout            = 20 * time.Minute
	autoUpdateRetryAfter  = 24 * time.Hour
	autoUpdateWindowStart = 2 // 02:00 local time
	autoUpdateWindowEnd   = 5 // until 05:00
	autoUpdateTimezone    = "America/Sao_Paulo"
)

var (
	ErrInvalidFirmwareVersion  = errors.New("version must look like 1.2.3")
	ErrFirmwareVersionNotNewer = errors.New("version must be newer than the latest published release")
	ErrInvalidFirmwareImage    = errors.New("file is not an ESP32 firmware image")
	ErrFirmwareTooLarge        = errors.New("firmware is larger than the controller OTA partition")
	ErrNoFirmwareRelease       = errors.New("no firmware release published")
	ErrNoFirmwareUpdate        = errors.New("controller is already on the latest firmware")
	ErrControllerOffline       = errors.New("controller is offline")
	ErrControllerIrrigating    = errors.New("controller is irrigating")
	ErrOtaInProgress           = errors.New("an update is already in progress")
	ErrControllerNotOwned      = errors.New("controller doesn't belong to authenticated user")
	ErrFirmwareSigningDisabled = errors.New("firmware signing key is not configured (FIRMWARE_SIGNING_KEY_FILE)")
)

type FirmwareService interface {
	PublishRelease(ctx context.Context, form dto.PublishFirmwareForm, file io.Reader) (db.FirmwareRelease, error)
	ListReleases(ctx context.Context, limit, offset int32) ([]db.FirmwareRelease, error)
	LatestRelease(ctx context.Context) (db.FirmwareRelease, error)
	GetRelease(ctx context.Context, releaseUUID uuid.UUID) (db.FirmwareRelease, error)
	GetControllerFirmware(ctx context.Context, userID int64, controllerUUID uuid.UUID) (dto.ControllerFirmwareResponse, error)
	RequestUpdate(ctx context.Context, userID int64, controllerUUID uuid.UUID) (dto.ControllerFirmwareResponse, error)
	SetAutoUpdate(ctx context.Context, userID int64, controllerUUID uuid.UUID, enabled bool) (dto.ControllerFirmwareResponse, error)
	HandleOtaEvent(ctx context.Context, deviceUID string, event mqtt.OtaEventPayload) error
	RunMaintenance(ctx context.Context, now time.Time) error
}

type firmwareService struct {
	store         db.Store
	mqttClient    client.Client
	hub           *websocket.Hub
	notifications NotificationService
	storageDir    string
	location      *time.Location
	signingKey    *ecdsa.PrivateKey
}

// NewFirmwareService: without a signing key nothing can be published, since
// controllers reject unsigned images.
func NewFirmwareService(
	store db.Store,
	mqttClient client.Client,
	hub *websocket.Hub,
	notifications NotificationService,
	storageDir string,
	signingKey *ecdsa.PrivateKey,
) FirmwareService {
	if mqttClient == nil {
		mqttClient = client.NewNoopClient()
	}
	location, err := time.LoadLocation(autoUpdateTimezone)
	if err != nil {
		location = time.FixedZone("BRT", -3*60*60)
	}
	return &firmwareService{
		store:         store,
		mqttClient:    mqttClient,
		hub:           hub,
		notifications: notifications,
		storageDir:    storageDir,
		location:      location,
		signingKey:    signingKey,
	}
}

func (s *firmwareService) PublishRelease(ctx context.Context, form dto.PublishFirmwareForm, file io.Reader) (db.FirmwareRelease, error) {
	if s.signingKey == nil {
		return db.FirmwareRelease{}, ErrFirmwareSigningDisabled
	}
	version := strings.TrimSpace(form.Version)
	if !util.IsValidVersion(version) {
		return db.FirmwareRelease{}, ErrInvalidFirmwareVersion
	}

	latest, err := s.store.GetLatestFirmwareRelease(ctx)
	if err == nil && util.CompareVersions(version, latest.Version) <= 0 {
		return db.FirmwareRelease{}, ErrFirmwareVersionNotNewer
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return db.FirmwareRelease{}, err
	}

	path, size, sum, err := s.storeImage(version, file)
	if err != nil {
		return db.FirmwareRelease{}, err
	}

	digest, err := hex.DecodeString(sum)
	if err != nil {
		os.Remove(path)
		return db.FirmwareRelease{}, err
	}
	signature, err := util.SignFirmwareDigest(s.signingKey, digest)
	if err != nil {
		os.Remove(path)
		return db.FirmwareRelease{}, err
	}

	release, err := s.store.CreateFirmwareRelease(ctx, db.CreateFirmwareReleaseParams{
		Version:      version,
		Build:        strings.TrimSpace(form.Build),
		FilePath:     path,
		FileSize:     size,
		Sha256:       sum,
		ReleaseNotes: strings.TrimSpace(form.ReleaseNotes),
		Signature:    signature,
	})
	if err != nil {
		os.Remove(path)
		return db.FirmwareRelease{}, err
	}

	if err := s.notifyNewRelease(ctx, release); err != nil {
		log.Printf("firmware: failed to notify release %s: %v", release.Version, err)
	}
	return release, nil
}

// storeImage streams the upload to disk while hashing it, so a large file
// is never held in memory. The version was validated, so it's path-safe.
func (s *firmwareService) storeImage(version string, file io.Reader) (string, int64, string, error) {
	if err := os.MkdirAll(s.storageDir, 0o750); err != nil {
		return "", 0, "", err
	}

	tmp, err := os.CreateTemp(s.storageDir, "upload-*.bin")
	if err != nil {
		return "", 0, "", err
	}
	defer os.Remove(tmp.Name())

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(file, MaxFirmwareSize+1))
	closeErr := tmp.Close()
	if err != nil {
		return "", 0, "", err
	}
	if closeErr != nil {
		return "", 0, "", closeErr
	}
	if size > MaxFirmwareSize {
		return "", 0, "", ErrFirmwareTooLarge
	}

	header := make([]byte, 1)
	f, err := os.Open(tmp.Name())
	if err != nil {
		return "", 0, "", err
	}
	_, readErr := io.ReadFull(f, header)
	f.Close()
	if readErr != nil || header[0] != esp32ImageMagic {
		return "", 0, "", ErrInvalidFirmwareImage
	}

	path := filepath.Join(s.storageDir, fmt.Sprintf("controller-%s.bin", version))
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", 0, "", err
	}
	return path, size, hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *firmwareService) ListReleases(ctx context.Context, limit, offset int32) ([]db.FirmwareRelease, error) {
	return s.store.ListFirmwareReleases(ctx, db.ListFirmwareReleasesParams{Limit: limit, Offset: offset})
}

func (s *firmwareService) LatestRelease(ctx context.Context) (db.FirmwareRelease, error) {
	release, err := s.store.GetLatestFirmwareRelease(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return db.FirmwareRelease{}, ErrNoFirmwareRelease
	}
	return release, err
}

func (s *firmwareService) GetRelease(ctx context.Context, releaseUUID uuid.UUID) (db.FirmwareRelease, error) {
	return s.store.GetFirmwareRelease(ctx, releaseUUID)
}

func (s *firmwareService) ownedController(ctx context.Context, userID int64, controllerUUID uuid.UUID) (db.ControllerDevice, error) {
	controller, err := s.store.GetControllerDevice(ctx, controllerUUID)
	if err != nil {
		return db.ControllerDevice{}, err
	}
	if !controller.UserID.Valid || controller.UserID.Int64 != userID {
		return db.ControllerDevice{}, ErrControllerNotOwned
	}
	return controller, nil
}

// latestOrNil hides "no release yet" so callers can treat it as "no update".
func (s *firmwareService) latestOrNil(ctx context.Context) (*db.FirmwareRelease, error) {
	release, err := s.store.GetLatestFirmwareRelease(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func isOtaInProgress(c db.ControllerDevice, now time.Time) bool {
	if !c.OtaStatus.Valid {
		return false
	}
	if c.OtaStatus.String != mqtt.OtaStatusPending && c.OtaStatus.String != mqtt.OtaStatusDownloading {
		return false
	}
	return c.OtaRequestedAt.Valid && now.Sub(c.OtaRequestedAt.Time) < otaTimeout
}

func (s *firmwareService) blockedReason(ctx context.Context, c db.ControllerDevice) (string, error) {
	if isOtaInProgress(c, time.Now()) {
		return "IN_PROGRESS", nil
	}
	if !c.IsOnline {
		return "OFFLINE", nil
	}
	irrigating, err := s.store.CountIrrigatingDevicesByController(ctx, sql.NullInt64{Int64: c.ID, Valid: true})
	if err != nil {
		return "", err
	}
	if irrigating > 0 {
		return "IRRIGATING", nil
	}
	return "", nil
}

func (s *firmwareService) describe(ctx context.Context, c db.ControllerDevice) (dto.ControllerFirmwareResponse, error) {
	rsp := dto.ControllerFirmwareResponse{
		ControllerUuid:   c.Uuid,
		CurrentVersion:   c.FirmwareVersion,
		AutoUpdate:       c.AutoUpdate,
		OtaStatus:        nullStringPtr(c.OtaStatus),
		OtaTargetVersion: nullStringPtr(c.OtaTargetVersion),
		OtaError:         nullStringPtr(c.OtaError),
	}
	if c.OtaUpdatedAt.Valid {
		rsp.OtaUpdatedAt = &c.OtaUpdatedAt.Time
	}

	latest, err := s.latestOrNil(ctx)
	if err != nil {
		return rsp, err
	}
	if latest != nil {
		rsp.LatestVersion = &latest.Version
		rsp.ReleaseNotes = &latest.ReleaseNotes
		rsp.UpdateAvailable = util.CompareVersions(latest.Version, c.FirmwareVersion) > 0
	}

	reason, err := s.blockedReason(ctx, c)
	if err != nil {
		return rsp, err
	}
	if reason != "" {
		rsp.BlockedReason = &reason
	}
	return rsp, nil
}

func (s *firmwareService) GetControllerFirmware(ctx context.Context, userID int64, controllerUUID uuid.UUID) (dto.ControllerFirmwareResponse, error) {
	controller, err := s.ownedController(ctx, userID, controllerUUID)
	if err != nil {
		return dto.ControllerFirmwareResponse{}, err
	}
	return s.describe(ctx, controller)
}

func (s *firmwareService) RequestUpdate(ctx context.Context, userID int64, controllerUUID uuid.UUID) (dto.ControllerFirmwareResponse, error) {
	controller, err := s.ownedController(ctx, userID, controllerUUID)
	if err != nil {
		return dto.ControllerFirmwareResponse{}, err
	}

	latest, err := s.latestOrNil(ctx)
	if err != nil {
		return dto.ControllerFirmwareResponse{}, err
	}
	if latest == nil || util.CompareVersions(latest.Version, controller.FirmwareVersion) <= 0 {
		return dto.ControllerFirmwareResponse{}, ErrNoFirmwareUpdate
	}

	reason, err := s.blockedReason(ctx, controller)
	if err != nil {
		return dto.ControllerFirmwareResponse{}, err
	}
	switch reason {
	case "IN_PROGRESS":
		return dto.ControllerFirmwareResponse{}, ErrOtaInProgress
	case "OFFLINE":
		return dto.ControllerFirmwareResponse{}, ErrControllerOffline
	case "IRRIGATING":
		return dto.ControllerFirmwareResponse{}, ErrControllerIrrigating
	}

	updated, err := s.startUpdate(ctx, controller, *latest)
	if err != nil {
		return dto.ControllerFirmwareResponse{}, err
	}
	return s.describe(ctx, updated)
}

func (s *firmwareService) startUpdate(ctx context.Context, controller db.ControllerDevice, release db.FirmwareRelease) (db.ControllerDevice, error) {
	payload, err := mqtt.NewOtaUpdateCommand(uuid.NewString(), release.Version)
	if err != nil {
		return db.ControllerDevice{}, err
	}

	updated, err := s.store.StartControllerOta(ctx, db.StartControllerOtaParams{
		ID:            controller.ID,
		TargetVersion: sql.NullString{String: release.Version, Valid: true},
	})
	if err != nil {
		return db.ControllerDevice{}, err
	}

	topic := mqtt.DeviceCommandTopic(s.mqttClient.TopicPrefix(), controller.ControllerDeviceUid)
	if err := s.mqttClient.Publish(topic, payload); err != nil {
		failed, markErr := s.setStatus(ctx, updated, mqtt.OtaStatusFailed, "publish_failed")
		if markErr != nil {
			return db.ControllerDevice{}, markErr
		}
		s.broadcast(failed)
		return failed, fmt.Errorf("send update command: %w", err)
	}

	s.broadcast(updated)
	return updated, nil
}

func (s *firmwareService) SetAutoUpdate(ctx context.Context, userID int64, controllerUUID uuid.UUID, enabled bool) (dto.ControllerFirmwareResponse, error) {
	controller, err := s.ownedController(ctx, userID, controllerUUID)
	if err != nil {
		return dto.ControllerFirmwareResponse{}, err
	}
	updated, err := s.store.SetControllerAutoUpdate(ctx, db.SetControllerAutoUpdateParams{
		ID:         controller.ID,
		AutoUpdate: enabled,
	})
	if err != nil {
		return dto.ControllerFirmwareResponse{}, err
	}
	return s.describe(ctx, updated)
}

func (s *firmwareService) setStatus(ctx context.Context, c db.ControllerDevice, status, errMsg string) (db.ControllerDevice, error) {
	return s.store.UpdateControllerOtaStatus(ctx, db.UpdateControllerOtaStatusParams{
		ID:        c.ID,
		OtaStatus: sql.NullString{String: status, Valid: true},
		OtaError:  sql.NullString{String: errMsg, Valid: errMsg != ""},
	})
}

func (s *firmwareService) HandleOtaEvent(ctx context.Context, deviceUID string, event mqtt.OtaEventPayload) error {
	controller, err := s.store.GetControllerDeviceByUID(ctx, deviceUID)
	if err != nil {
		return fmt.Errorf("ota event for unknown controller %s: %w", deviceUID, err)
	}

	errMsg := truncateRunes(strings.TrimSpace(event.Error), 200)

	// Only events about an update we actually requested count; anything
	// else (stale retained message, forged publish) is ignored.
	status := controller.OtaStatus.String
	requested := controller.OtaTargetVersion.Valid
	inFlight := status == mqtt.OtaStatusPending || status == mqtt.OtaStatusDownloading
	switch event.OtaStatus {
	case mqtt.OtaStatusDownloading, mqtt.OtaStatusUpToDate:
		if !inFlight {
			log.Printf("firmware: ignoring %s from %s, no update in progress", event.OtaStatus, deviceUID)
			return nil
		}
	case mqtt.OtaStatusSuccess:
		// Also accepted after a timeout: slow downloads can finish late.
		if !requested || controller.OtaTargetVersion.String != event.Version || status == mqtt.OtaStatusSuccess {
			log.Printf("firmware: ignoring SUCCESS %s from %s, not the requested update", event.Version, deviceUID)
			return nil
		}
	case mqtt.OtaStatusFailed:
		if !requested || status == mqtt.OtaStatusSuccess {
			log.Printf("firmware: ignoring FAILED from %s, no update requested", deviceUID)
			return nil
		}
	}

	switch event.OtaStatus {
	case mqtt.OtaStatusDownloading:
		updated, err := s.setStatus(ctx, controller, mqtt.OtaStatusDownloading, "")
		if err != nil {
			return err
		}
		s.broadcast(updated)

	case mqtt.OtaStatusSuccess, mqtt.OtaStatusUpToDate:
		// Sent after the reboot, once the new image is running and connected.
		updated, err := s.setStatus(ctx, controller, mqtt.OtaStatusSuccess, "")
		if err != nil {
			return err
		}
		if event.Version != "" && event.Version != updated.FirmwareVersion {
			updated, err = s.store.UpdateControllerDeviceFirmware(ctx, db.UpdateControllerDeviceFirmwareParams{
				ID:              updated.ID,
				FirmwareVersion: event.Version,
				FirmwareBuild:   updated.FirmwareBuild,
			})
			if err != nil {
				return err
			}
		}
		s.broadcast(updated)
		if event.OtaStatus == mqtt.OtaStatusSuccess {
			s.notifyOwner(ctx, updated,
				"Controladora atualizada",
				fmt.Sprintf("\"%s\" foi atualizada para a versão %s.", updated.Name, event.Version))
		}

	case mqtt.OtaStatusFailed:
		if errMsg == "" {
			errMsg = "unknown_error"
		}
		alreadyFailed := status == mqtt.OtaStatusFailed
		updated, err := s.setStatus(ctx, controller, mqtt.OtaStatusFailed, errMsg)
		if err != nil {
			return err
		}
		s.broadcast(updated)
		if !alreadyFailed {
			s.notifyFailure(ctx, updated)
		}

	default:
		return fmt.Errorf("invalid ota_status %q", event.OtaStatus)
	}
	return nil
}

func (s *firmwareService) RunMaintenance(ctx context.Context, now time.Time) error {
	stale, err := s.store.FailStaleControllerOta(ctx, sql.NullTime{Time: now.Add(-otaTimeout), Valid: true})
	if err != nil {
		return err
	}
	for _, c := range stale {
		s.broadcast(c)
		s.notifyFailure(ctx, c)
	}

	hour := now.In(s.location).Hour()
	if hour < autoUpdateWindowStart || hour >= autoUpdateWindowEnd {
		return nil
	}

	latest, err := s.latestOrNil(ctx)
	if err != nil || latest == nil {
		return err
	}

	candidates, err := s.store.ListAutoUpdateCandidates(ctx)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if util.CompareVersions(latest.Version, c.FirmwareVersion) <= 0 {
			continue
		}
		// Don't hammer a controller that already failed this version today.
		if c.OtaStatus.String == mqtt.OtaStatusFailed &&
			c.OtaTargetVersion.String == latest.Version &&
			c.OtaUpdatedAt.Valid && now.Sub(c.OtaUpdatedAt.Time) < autoUpdateRetryAfter {
			continue
		}
		reason, err := s.blockedReason(ctx, c)
		if err != nil || reason != "" {
			continue
		}
		if _, err := s.startUpdate(ctx, c, *latest); err != nil {
			log.Printf("firmware: auto-update of controller %s failed to start: %v", c.ControllerDeviceUid, err)
		}
	}
	return nil
}

// notifyNewRelease tells owners of outdated controllers once per version.
// Auto-update owners get the "updated" notice later instead.
func (s *firmwareService) notifyNewRelease(ctx context.Context, release db.FirmwareRelease) error {
	controllers, err := s.store.ListOwnedControllersNotNotified(ctx, release.Version)
	if err != nil {
		return err
	}
	for _, c := range controllers {
		if util.CompareVersions(release.Version, c.FirmwareVersion) <= 0 {
			continue
		}
		if !c.AutoUpdate {
			s.notifyOwner(ctx, c,
				"Nova atualização para sua controladora",
				fmt.Sprintf("A versão %s está disponível para \"%s\". Abra o app para atualizar.", release.Version, c.Name))
		}
		if err := s.store.MarkControllerUpdateNotified(ctx, db.MarkControllerUpdateNotifiedParams{
			ID:      c.ID,
			Version: sql.NullString{String: release.Version, Valid: true},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *firmwareService) notifyFailure(ctx context.Context, c db.ControllerDevice) {
	s.notifyOwner(ctx, c,
		"Falha ao atualizar a controladora",
		fmt.Sprintf("Não foi possível atualizar \"%s\". Ela continua funcionando na versão %s. Tente de novo pelo app.", c.Name, c.FirmwareVersion))
}

func (s *firmwareService) notifyOwner(ctx context.Context, c db.ControllerDevice, title, body string) {
	if !c.UserID.Valid || s.notifications == nil {
		return
	}
	if err := s.notifications.CreateAndDeliver(ctx, c.UserID.Int64, NotificationTypeFirmwareUpdate, title, body); err != nil {
		log.Printf("firmware: failed to notify user %d: %v", c.UserID.Int64, err)
	}
}

func (s *firmwareService) broadcast(c db.ControllerDevice) {
	if s.hub == nil || !c.UserID.Valid {
		return
	}
	s.hub.BroadcastToUser(c.UserID.Int64, map[string]any{
		"type":             "controller_ota",
		"controller_uuid":  c.Uuid.String(),
		"device_uid":       c.ControllerDeviceUid,
		"ota_status":       c.OtaStatus.String,
		"firmware_version": c.FirmwareVersion,
	})
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}
