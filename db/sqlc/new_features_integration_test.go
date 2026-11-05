package db

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

// These run against the real Postgres schema: they check SQL semantics
// (COALESCE, atomic UPDATE ... RETURNING, CTEs, case-insensitive lookups)
// that mocks can't.

func ownedDevice(t *testing.T) (User, ControllerDevice, Device) {
	t.Helper()
	ctrl := createRandomControllerDevice(t)
	device := createRandomDevice(t)
	_, err := testDB.Exec("UPDATE devices SET controller_device_id = $1 WHERE id = $2", ctrl.ID, device.ID)
	require.NoError(t, err)
	user, err := testQueries.GetUserByID(context.Background(), ctrl.UserID.Int64)
	require.NoError(t, err)
	device.ControllerDeviceID = sql.NullInt64{Int64: ctrl.ID, Valid: true}
	return user, ctrl, device
}

func TestIntegrationTelemetryKeepsLastGoodMoisture(t *testing.T) {
	ctx := context.Background()
	device := createRandomDevice(t)
	now := sql.NullTime{Time: time.Now(), Valid: true}

	updated, err := testQueries.UpdateDeviceTelemetryByUID(ctx, UpdateDeviceTelemetryByUIDParams{
		DeviceUid:    device.DeviceUid,
		LastSeen:     now,
		Status:       "ATIVO",
		SoilMoisture: sql.NullInt32{Int32: 57, Valid: true},
		SoilSensorOk: sql.NullBool{Bool: true, Valid: true},
		SoilRaw:      sql.NullInt32{Int32: 2000, Valid: true},
		WifiRssi:     sql.NullInt32{Int32: -60, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, int32(57), updated.SoilMoisture.Int32)

	// Faulty probe: no moisture -> the stored value must survive.
	updated, err = testQueries.UpdateDeviceTelemetryByUID(ctx, UpdateDeviceTelemetryByUIDParams{
		DeviceUid:    device.DeviceUid,
		LastSeen:     now,
		Status:       "ATIVO",
		SoilSensorOk: sql.NullBool{Bool: false, Valid: true},
		SoilRaw:      sql.NullInt32{Int32: 4095, Valid: true},
	})
	require.NoError(t, err)
	require.True(t, updated.SoilMoisture.Valid)
	require.Equal(t, int32(57), updated.SoilMoisture.Int32)
	require.False(t, updated.SoilSensorOk.Bool)
	require.Equal(t, int32(4095), updated.SoilRaw.Int32)
	require.False(t, updated.WifiRssi.Valid, "fields absent from the payload are cleared, not stale")
}

func TestIntegrationDeviceLocationSetAndClear(t *testing.T) {
	ctx := context.Background()
	device := createRandomDevice(t)

	located, err := testQueries.UpdateDeviceLocation(ctx, UpdateDeviceLocationParams{
		Uuid:      device.Uuid,
		Latitude:  sql.NullFloat64{Float64: -23.5505, Valid: true},
		Longitude: sql.NullFloat64{Float64: -46.6333, Valid: true},
	})
	require.NoError(t, err)
	require.InDelta(t, -23.5505, located.Latitude.Float64, 1e-9)

	cleared, err := testQueries.UpdateDeviceLocation(ctx, UpdateDeviceLocationParams{Uuid: device.Uuid})
	require.NoError(t, err)
	require.False(t, cleared.Latitude.Valid)
	require.False(t, cleared.Longitude.Valid)
}

func TestIntegrationDeviceOwnershipLookup(t *testing.T) {
	user, _, device := ownedDevice(t)
	owner, err := testQueries.GetDeviceUserID(context.Background(), device.Uuid)
	require.NoError(t, err)
	require.Equal(t, user.ID, owner.Int64)

	orphan := createRandomDevice(t)
	_, err = testQueries.GetDeviceUserID(context.Background(), orphan.Uuid)
	require.ErrorIs(t, err, sql.ErrNoRows, "a device without controller has no owner")
}

func TestIntegrationEmailLookupIsCaseInsensitive(t *testing.T) {
	user := createRandomUser(t)
	found, err := testQueries.GetUserByEmail(context.Background(), strings.ToUpper(user.Email))
	require.NoError(t, err)
	require.Equal(t, user.ID, found.ID)
}

func TestIntegrationPasswordResetTokenIsSingleUseUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	hash := util.HashToken(util.RandomString(32))
	_, err := testQueries.CreatePasswordResetToken(ctx, CreatePasswordResetTokenParams{
		UserID: user.ID, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	// Ten requests race with the same token: exactly one may consume it.
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := testQueries.ConsumePasswordResetToken(ctx, hash); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, wins)
}

func TestIntegrationExpiredResetTokenCannotBeConsumed(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	hash := util.HashToken(util.RandomString(32))
	_, err := testQueries.CreatePasswordResetToken(ctx, CreatePasswordResetTokenParams{
		UserID: user.ID, TokenHash: hash, ExpiresAt: time.Now().Add(-time.Minute),
	})
	require.NoError(t, err)
	_, err = testQueries.ConsumePasswordResetToken(ctx, hash)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestIntegrationSupportConversationsCountUnread(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	for _, sender := range []string{"USER", "USER", "SUPPORT"} {
		_, err := testQueries.CreateSupportMessage(ctx, CreateSupportMessageParams{UserID: user.ID, Sender: sender, Body: "msg " + sender})
		require.NoError(t, err)
	}

	find := func() ListSupportConversationsRow {
		rows, err := testQueries.ListSupportConversations(ctx, ListSupportConversationsParams{Limit: 1000, Offset: 0})
		require.NoError(t, err)
		for _, r := range rows {
			if r.UserUuid == user.Uuid {
				return r
			}
		}
		t.Fatal("conversation not listed")
		return ListSupportConversationsRow{}
	}

	row := find()
	require.Equal(t, int64(2), row.UnreadCount)
	require.Equal(t, "SUPPORT", row.LastSender, "last message is the newest one")

	require.NoError(t, testQueries.MarkSupportMessagesRead(ctx, MarkSupportMessagesReadParams{UserID: user.ID, Sender: "USER"}))
	require.Equal(t, int64(0), find().UnreadCount)
}

func TestIntegrationStaleOtaIsFailedOnlyWhenOld(t *testing.T) {
	ctx := context.Background()
	fresh := createRandomControllerDevice(t)
	stale := createRandomControllerDevice(t)
	for _, c := range []ControllerDevice{fresh, stale} {
		_, err := testQueries.StartControllerOta(ctx, StartControllerOtaParams{ID: c.ID, TargetVersion: sql.NullString{String: "9.0.0", Valid: true}})
		require.NoError(t, err)
	}
	_, err := testDB.Exec("UPDATE controller_devices SET ota_requested_at = now() - interval '1 hour' WHERE id = $1", stale.ID)
	require.NoError(t, err)

	failed, err := testQueries.FailStaleControllerOta(ctx, sql.NullTime{Time: time.Now().Add(-20 * time.Minute), Valid: true})
	require.NoError(t, err)

	ids := map[int64]bool{}
	for _, c := range failed {
		ids[c.ID] = true
	}
	require.True(t, ids[stale.ID])
	require.False(t, ids[fresh.ID])

	reloaded, err := testQueries.GetControllerDeviceByID(ctx, stale.ID)
	require.NoError(t, err)
	require.Equal(t, "FAILED", reloaded.OtaStatus.String)
	require.Equal(t, "timeout", reloaded.OtaError.String)
}

func TestIntegrationReleaseNotificationIsOncePerVersion(t *testing.T) {
	ctx := context.Background()
	ctrl := createRandomControllerDevice(t)
	version := "77." + util.RandomString(3)

	isListed := func() bool {
		rows, err := testQueries.ListOwnedControllersNotNotified(ctx, version)
		require.NoError(t, err)
		for _, r := range rows {
			if r.ID == ctrl.ID {
				return true
			}
		}
		return false
	}

	require.True(t, isListed())
	require.NoError(t, testQueries.MarkControllerUpdateNotified(ctx, MarkControllerUpdateNotifiedParams{
		ID: ctrl.ID, Version: sql.NullString{String: version, Valid: true},
	}))
	require.False(t, isListed(), "already notified about this version")
}

func TestIntegrationIrrigatingDevicesCount(t *testing.T) {
	_, ctrl, device := ownedDevice(t)
	count, err := testQueries.CountIrrigatingDevicesByController(context.Background(), sql.NullInt64{Int64: ctrl.ID, Valid: true})
	require.NoError(t, err)
	require.Zero(t, count)

	_, err = testDB.Exec("UPDATE devices SET is_irrigating = true WHERE id = $1", device.ID)
	require.NoError(t, err)
	count, err = testQueries.CountIrrigatingDevicesByController(context.Background(), sql.NullInt64{Int64: ctrl.ID, Valid: true})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}
