package service_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type publishedMessage struct {
	topic   string
	payload []byte
}

type fakeMqtt struct {
	published []publishedMessage
	failWith  error
}

func (f *fakeMqtt) Publish(topic string, payload []byte) error {
	if f.failWith != nil {
		return f.failWith
	}
	f.published = append(f.published, publishedMessage{topic, payload})
	return nil
}
func (f *fakeMqtt) Subscribe(string, client.MessageHandler) error { return nil }
func (f *fakeMqtt) Close()                                         {}
func (f *fakeMqtt) TopicPrefix() string                            { return "ground-guard" }

type firmwareFixture struct {
	store    *mockdb.MockStore
	mqtt     *fakeMqtt
	notifier *fakeNotifier
	svc      service.FirmwareService
	dir      string
	key      *ecdsa.PrivateKey
}

func newFirmwareFixture(t *testing.T) firmwareFixture {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	fakeClient := &fakeMqtt{}
	notifier := &fakeNotifier{}
	dir := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return firmwareFixture{
		store:    store,
		mqtt:     fakeClient,
		notifier: notifier,
		svc:      service.NewFirmwareService(store, fakeClient, websocket.NewHub(), notifier, dir, key),
		dir:      dir,
		key:      key,
	}
}

func TestPublishReleaseRefusesWithoutSigningKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CreateFirmwareRelease(gomock.Any(), gomock.Any()).Times(0)
	svc := service.NewFirmwareService(store, &fakeMqtt{}, websocket.NewHub(), &fakeNotifier{}, t.TempDir(), nil)

	_, err := svc.PublishRelease(context.Background(),
		dto.PublishFirmwareForm{Version: "1.1.0", Build: "1"}, bytes.NewReader(espImage(64)))
	require.ErrorIs(t, err, service.ErrFirmwareSigningDisabled)
}

func espImage(size int) []byte {
	img := bytes.Repeat([]byte{0xAB}, size)
	img[0] = 0xE9
	return img
}

func ownedController(version string) db.ControllerDevice {
	return db.ControllerDevice{
		ID:                  10,
		Uuid:                uuid.New(),
		ControllerDeviceUid: "AABBCCDDEEFF",
		Name:                "Jardim",
		FirmwareVersion:     version,
		IsOnline:            true,
		UserID:              sql.NullInt64{Int64: 1, Valid: true},
	}
}

func TestPublishReleaseStoresImageAndNotifiesOutdatedOwners(t *testing.T) {
	f := newFirmwareFixture(t)
	img := espImage(4096)
	sum := sha256.Sum256(img)

	manual := ownedController("1.0.0")
	auto := ownedController("1.0.0")
	auto.ID, auto.AutoUpdate = 11, true
	current := ownedController("1.1.0")
	current.ID = 12

	f.store.EXPECT().GetLatestFirmwareRelease(gomock.Any()).Return(db.FirmwareRelease{Version: "1.0.0"}, nil)
	f.store.EXPECT().
		CreateFirmwareRelease(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateFirmwareReleaseParams) (db.FirmwareRelease, error) {
			require.Equal(t, "1.1.0", arg.Version)
			require.Equal(t, int64(len(img)), arg.FileSize)
			require.Equal(t, hex.EncodeToString(sum[:]), arg.Sha256)
			sig, err := base64.StdEncoding.DecodeString(arg.Signature)
			require.NoError(t, err)
			require.True(t, ecdsa.VerifyASN1(&f.key.PublicKey, sum[:], sig), "release must be signed over the image digest")
			stored, err := os.ReadFile(arg.FilePath)
			require.NoError(t, err)
			require.Equal(t, img, stored)
			return db.FirmwareRelease{Version: arg.Version}, nil
		})
	f.store.EXPECT().ListOwnedControllersNotNotified(gomock.Any(), "1.1.0").
		Return([]db.ControllerDevice{manual, auto, current}, nil)
	f.store.EXPECT().MarkControllerUpdateNotified(gomock.Any(), gomock.Any()).Times(2).Return(nil)

	_, err := f.svc.PublishRelease(context.Background(),
		dto.PublishFirmwareForm{Version: "1.1.0", Build: "2"}, bytes.NewReader(img))
	require.NoError(t, err)
	require.Equal(t, 1, f.notifier.calls, "only the outdated controller without auto-update is notified")
	require.Equal(t, service.NotificationTypeFirmwareUpdate, f.notifier.notifType)
}

func TestPublishReleaseRejectsInvalidUploads(t *testing.T) {
	cases := []struct {
		name    string
		version string
		latest  string
		file    []byte
		wantErr error
	}{
		{"bad version", "1.1.0-beta", "", espImage(10), service.ErrInvalidFirmwareVersion},
		{"path traversal", "../1", "", espImage(10), service.ErrInvalidFirmwareVersion},
		{"not newer", "1.0.0", "1.0.0", espImage(10), service.ErrFirmwareVersionNotNewer},
		{"older", "0.9.9", "1.0.0", espImage(10), service.ErrFirmwareVersionNotNewer},
		{"not esp32 image", "1.1.0", "1.0.0", []byte("hello"), service.ErrInvalidFirmwareImage},
		{"too large", "1.1.0", "1.0.0", espImage(int(service.MaxFirmwareSize) + 1), service.ErrFirmwareTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFirmwareFixture(t)
			if tc.wantErr != service.ErrInvalidFirmwareVersion {
				if tc.latest == "" {
					f.store.EXPECT().GetLatestFirmwareRelease(gomock.Any()).Return(db.FirmwareRelease{}, sql.ErrNoRows)
				} else {
					f.store.EXPECT().GetLatestFirmwareRelease(gomock.Any()).Return(db.FirmwareRelease{Version: tc.latest}, nil)
				}
			}
			f.store.EXPECT().CreateFirmwareRelease(gomock.Any(), gomock.Any()).Times(0)

			_, err := f.svc.PublishRelease(context.Background(),
				dto.PublishFirmwareForm{Version: tc.version, Build: "1"}, bytes.NewReader(tc.file))
			require.ErrorIs(t, err, tc.wantErr)

			entries, _ := os.ReadDir(f.dir)
			require.Empty(t, entries, "rejected uploads must not leave files behind")
		})
	}
}

func TestRequestUpdateSendsCommand(t *testing.T) {
	f := newFirmwareFixture(t)
	c := ownedController("1.0.0")

	f.store.EXPECT().GetControllerDevice(gomock.Any(), c.Uuid).Return(c, nil)
	f.store.EXPECT().GetLatestFirmwareRelease(gomock.Any()).AnyTimes().Return(db.FirmwareRelease{Version: "1.1.0"}, nil)
	f.store.EXPECT().CountIrrigatingDevicesByController(gomock.Any(), gomock.Any()).AnyTimes().Return(int64(0), nil)
	started := c
	started.OtaStatus = sql.NullString{String: "PENDING", Valid: true}
	started.OtaTargetVersion = sql.NullString{String: "1.1.0", Valid: true}
	started.OtaRequestedAt = sql.NullTime{Time: time.Now(), Valid: true}
	f.store.EXPECT().
		StartControllerOta(gomock.Any(), db.StartControllerOtaParams{
			ID:            c.ID,
			TargetVersion: sql.NullString{String: "1.1.0", Valid: true},
		}).
		Return(started, nil)

	rsp, err := f.svc.RequestUpdate(context.Background(), 1, c.Uuid)
	require.NoError(t, err)
	require.Equal(t, "PENDING", *rsp.OtaStatus)
	require.Equal(t, "IN_PROGRESS", *rsp.BlockedReason)

	require.Len(t, f.mqtt.published, 1)
	require.Equal(t, "ground-guard/devices/AABBCCDDEEFF/commands", f.mqtt.published[0].topic)
	var cmd mqtt.OtaUpdateCommand
	require.NoError(t, json.Unmarshal(f.mqtt.published[0].payload, &cmd))
	require.Equal(t, mqtt.OtaUpdateAction, cmd.Action)
	require.Equal(t, "1.1.0", cmd.Data.Version)
	_, err = uuid.Parse(cmd.CommandID)
	require.NoError(t, err)
}

func TestRequestUpdateGuards(t *testing.T) {
	cases := []struct {
		name       string
		controller func() db.ControllerDevice
		latest     string
		irrigating int64
		wantErr    error
	}{
		{"not owned", func() db.ControllerDevice {
			c := ownedController("1.0.0")
			c.UserID = sql.NullInt64{Int64: 99, Valid: true}
			return c
		}, "1.1.0", 0, service.ErrControllerNotOwned},
		{"already latest", func() db.ControllerDevice { return ownedController("1.1.0") }, "1.1.0", 0, service.ErrNoFirmwareUpdate},
		{"offline", func() db.ControllerDevice {
			c := ownedController("1.0.0")
			c.IsOnline = false
			return c
		}, "1.1.0", 0, service.ErrControllerOffline},
		{"irrigating", func() db.ControllerDevice { return ownedController("1.0.0") }, "1.1.0", 1, service.ErrControllerIrrigating},
		{"in progress", func() db.ControllerDevice {
			c := ownedController("1.0.0")
			c.OtaStatus = sql.NullString{String: "DOWNLOADING", Valid: true}
			c.OtaRequestedAt = sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true}
			return c
		}, "1.1.0", 0, service.ErrOtaInProgress},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFirmwareFixture(t)
			c := tc.controller()
			f.store.EXPECT().GetControllerDevice(gomock.Any(), c.Uuid).Return(c, nil)
			f.store.EXPECT().GetLatestFirmwareRelease(gomock.Any()).AnyTimes().Return(db.FirmwareRelease{Version: tc.latest}, nil)
			f.store.EXPECT().CountIrrigatingDevicesByController(gomock.Any(), gomock.Any()).AnyTimes().Return(tc.irrigating, nil)
			f.store.EXPECT().StartControllerOta(gomock.Any(), gomock.Any()).Times(0)

			_, err := f.svc.RequestUpdate(context.Background(), 1, c.Uuid)
			require.ErrorIs(t, err, tc.wantErr)
			require.Empty(t, f.mqtt.published)
		})
	}
}

func TestRequestUpdateMarksFailedWhenPublishFails(t *testing.T) {
	f := newFirmwareFixture(t)
	f.mqtt.failWith = errors.New("broker down")
	c := ownedController("1.0.0")

	f.store.EXPECT().GetControllerDevice(gomock.Any(), c.Uuid).Return(c, nil)
	f.store.EXPECT().GetLatestFirmwareRelease(gomock.Any()).AnyTimes().Return(db.FirmwareRelease{Version: "1.1.0"}, nil)
	f.store.EXPECT().CountIrrigatingDevicesByController(gomock.Any(), gomock.Any()).AnyTimes().Return(int64(0), nil)
	f.store.EXPECT().StartControllerOta(gomock.Any(), gomock.Any()).Return(c, nil)
	f.store.EXPECT().
		UpdateControllerOtaStatus(gomock.Any(), db.UpdateControllerOtaStatusParams{
			ID:        c.ID,
			OtaStatus: sql.NullString{String: "FAILED", Valid: true},
			OtaError:  sql.NullString{String: "publish_failed", Valid: true},
		}).
		Return(c, nil)

	_, err := f.svc.RequestUpdate(context.Background(), 1, c.Uuid)
	require.Error(t, err)
}

func updatingController(status string) db.ControllerDevice {
	c := ownedController("1.0.0")
	c.OtaStatus = sql.NullString{String: status, Valid: true}
	c.OtaTargetVersion = sql.NullString{String: "1.1.0", Valid: true}
	c.OtaRequestedAt = sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true}
	return c
}

func TestHandleOtaEventSuccessRecordsVersionAndNotifies(t *testing.T) {
	f := newFirmwareFixture(t)
	c := updatingController("DOWNLOADING")

	f.store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "AABBCCDDEEFF").Return(c, nil)
	f.store.EXPECT().UpdateControllerOtaStatus(gomock.Any(), gomock.Any()).Return(c, nil)
	f.store.EXPECT().
		UpdateControllerDeviceFirmware(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdateControllerDeviceFirmwareParams) (db.ControllerDevice, error) {
			require.Equal(t, "1.1.0", arg.FirmwareVersion)
			updated := c
			updated.FirmwareVersion = arg.FirmwareVersion
			return updated, nil
		})

	err := f.svc.HandleOtaEvent(context.Background(), "AABBCCDDEEFF",
		mqtt.OtaEventPayload{OtaStatus: "SUCCESS", Version: "1.1.0"})
	require.NoError(t, err)
	require.Equal(t, 1, f.notifier.calls)
	require.Contains(t, f.notifier.body, "1.1.0")
}

func TestHandleOtaEventIgnoresUnrequestedEvents(t *testing.T) {
	cases := []struct {
		name       string
		controller db.ControllerDevice
		event      mqtt.OtaEventPayload
	}{
		{"success without any request", ownedController("1.0.0"), mqtt.OtaEventPayload{OtaStatus: "SUCCESS", Version: "9.9.9"}},
		{"success for another version", updatingController("DOWNLOADING"), mqtt.OtaEventPayload{OtaStatus: "SUCCESS", Version: "9.9.9"}},
		{"duplicate success", updatingController("SUCCESS"), mqtt.OtaEventPayload{OtaStatus: "SUCCESS", Version: "1.1.0"}},
		{"downloading when idle", ownedController("1.0.0"), mqtt.OtaEventPayload{OtaStatus: "DOWNLOADING", Version: "1.1.0"}},
		{"failed without any request", ownedController("1.0.0"), mqtt.OtaEventPayload{OtaStatus: "FAILED", Error: "x"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFirmwareFixture(t)
			f.store.EXPECT().GetControllerDeviceByUID(gomock.Any(), gomock.Any()).Return(tc.controller, nil)
			f.store.EXPECT().UpdateControllerOtaStatus(gomock.Any(), gomock.Any()).Times(0)
			f.store.EXPECT().UpdateControllerDeviceFirmware(gomock.Any(), gomock.Any()).Times(0)

			require.NoError(t, f.svc.HandleOtaEvent(context.Background(), "AABBCCDDEEFF", tc.event))
			require.Zero(t, f.notifier.calls)
		})
	}
}

func TestHandleOtaEventFailureNotifies(t *testing.T) {
	f := newFirmwareFixture(t)
	c := updatingController("DOWNLOADING")

	f.store.EXPECT().GetControllerDeviceByUID(gomock.Any(), "AABBCCDDEEFF").Return(c, nil)
	f.store.EXPECT().
		UpdateControllerOtaStatus(gomock.Any(), db.UpdateControllerOtaStatusParams{
			ID:        c.ID,
			OtaStatus: sql.NullString{String: "FAILED", Valid: true},
			OtaError:  sql.NullString{String: "sha256_mismatch", Valid: true},
		}).
		Return(c, nil)

	err := f.svc.HandleOtaEvent(context.Background(), "AABBCCDDEEFF",
		mqtt.OtaEventPayload{OtaStatus: "FAILED", Version: "1.1.0", Error: "sha256_mismatch"})
	require.NoError(t, err)
	require.Equal(t, 1, f.notifier.calls)
}

func TestHandleOtaEventRejectsUnknownStatus(t *testing.T) {
	f := newFirmwareFixture(t)
	f.store.EXPECT().GetControllerDeviceByUID(gomock.Any(), gomock.Any()).Return(ownedController("1.0.0"), nil)

	err := f.svc.HandleOtaEvent(context.Background(), "AABBCCDDEEFF", mqtt.OtaEventPayload{OtaStatus: "HACKED"})
	require.Error(t, err)
}

func saoPaulo(t *testing.T, hour int) time.Time {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	return time.Date(2026, 11, 5, hour, 30, 0, 0, loc)
}

func TestRunMaintenanceOutsideWindowOnlyTimesOut(t *testing.T) {
	f := newFirmwareFixture(t)
	stale := ownedController("1.0.0")
	f.store.EXPECT().FailStaleControllerOta(gomock.Any(), gomock.Any()).Return([]db.ControllerDevice{stale}, nil)
	f.store.EXPECT().ListAutoUpdateCandidates(gomock.Any()).Times(0)

	require.NoError(t, f.svc.RunMaintenance(context.Background(), saoPaulo(t, 14)))
	require.Equal(t, 1, f.notifier.calls, "owner hears about the timed-out update")
}

func TestRunMaintenanceStartsAutoUpdatesOvernight(t *testing.T) {
	f := newFirmwareFixture(t)
	now := saoPaulo(t, 3)

	outdated := ownedController("1.0.0")
	upToDate := ownedController("1.1.0")
	upToDate.ID = 11
	failedRecently := ownedController("1.0.0")
	failedRecently.ID = 12
	failedRecently.OtaStatus = sql.NullString{String: "FAILED", Valid: true}
	failedRecently.OtaTargetVersion = sql.NullString{String: "1.1.0", Valid: true}
	failedRecently.OtaUpdatedAt = sql.NullTime{Time: now.Add(-time.Hour), Valid: true}

	f.store.EXPECT().FailStaleControllerOta(gomock.Any(), gomock.Any()).Return(nil, nil)
	f.store.EXPECT().GetLatestFirmwareRelease(gomock.Any()).Return(db.FirmwareRelease{Version: "1.1.0"}, nil)
	f.store.EXPECT().ListAutoUpdateCandidates(gomock.Any()).
		Return([]db.ControllerDevice{outdated, upToDate, failedRecently}, nil)
	f.store.EXPECT().CountIrrigatingDevicesByController(gomock.Any(), gomock.Any()).Return(int64(0), nil)
	f.store.EXPECT().
		StartControllerOta(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.StartControllerOtaParams) (db.ControllerDevice, error) {
			require.Equal(t, outdated.ID, arg.ID)
			return outdated, nil
		})

	require.NoError(t, f.svc.RunMaintenance(context.Background(), now))
	require.Len(t, f.mqtt.published, 1)
}
