//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// A user signs up, verifies the email, changes and resets the password,
// renews and revokes sessions, and finally deletes the account.
func TestE2E_AccountLifecycle(t *testing.T) {
	ctx := context.Background()
	u := newUser(t)

	var me obj
	u.api.expect(http.MethodGet, "/users/me", nil, http.StatusOK, &me)
	require.Equal(t, u.Email, me["email"])
	require.Equal(t, false, me["email_verified"])

	t.Run("duplicate signup is rejected", func(t *testing.T) {
		status, _ := anonymous(t).do(http.MethodPost, "/users", obj{
			"username": "e2edup" + randomHex(4), "full_name": "Dup",
			"email": u.Email, "password": "whatever123",
		})
		require.GreaterOrEqual(t, status, 400)
	})

	t.Run("login does not reveal which emails exist", func(t *testing.T) {
		wrongPw, _ := anonymous(t).do(http.MethodPost, "/users/login", obj{"email": u.Email, "password": "wrong-password"})
		unknown, _ := anonymous(t).do(http.MethodPost, "/users/login", obj{"email": "nobody_" + randomHex(4) + "@example.com", "password": "wrong-password"})
		require.Equal(t, http.StatusUnauthorized, wrongPw)
		require.Equal(t, wrongPw, unknown)
	})

	t.Run("email verification by link token", func(t *testing.T) {
		raw := randomHex(32)
		_, err := store.SetEmailVerificationToken(ctx, db.SetEmailVerificationTokenParams{
			Uuid:                       uuid.MustParse(u.UUID),
			EmailVerificationToken:     sql.NullString{String: util.HashToken(raw), Valid: true},
			EmailVerificationExpiresAt: sql.NullTime{Time: time.Now().Add(time.Hour), Valid: true},
		})
		require.NoError(t, err)

		status, body := anonymous(t).do(http.MethodPost, "/users/verify-email", obj{"token": raw})
		require.Lessf(t, status, 300, "verify: %s", body)
		u.api.expect(http.MethodGet, "/users/me", nil, http.StatusOK, &me)
		require.Equal(t, true, me["email_verified"])

		// A link can't be replayed.
		status, _ = anonymous(t).do(http.MethodPost, "/users/verify-email", obj{"token": raw})
		require.GreaterOrEqual(t, status, 400)
	})

	t.Run("change password", func(t *testing.T) {
		status, _ := u.api.do(http.MethodPut, "/users/password", obj{"current_password": "wrong", "new_password": "nova-senha-123"})
		require.Equal(t, http.StatusBadRequest, status)

		// The revocation check tolerates 1s of DB/API clock skew, so let the
		// current token age past that before changing the password.
		time.Sleep(1100 * time.Millisecond)
		newPw := "Nova-senha-" + randomHex(3)
		u.api.expect(http.MethodPut, "/users/password", obj{"current_password": u.Password, "new_password": newPw}, http.StatusNoContent, nil)
		status, _ = anonymous(t).do(http.MethodPost, "/users/login", obj{"email": u.Email, "password": u.Password})
		require.Equal(t, http.StatusUnauthorized, status, "old password must stop working")
		status, _ = u.api.do(http.MethodGet, "/users/me", nil)
		require.Equal(t, http.StatusUnauthorized, status, "tokens issued before the change are revoked")
		u.Password = newPw
		u.login(t)
	})

	t.Run("refresh token works until logout", func(t *testing.T) {
		anonymous(t).expect(http.MethodPost, "/tokens/refresh", obj{"refresh_token": u.Refresh}, http.StatusOK, nil)
		u.api.expect(http.MethodPost, "/users/logout", obj{"session_id": u.SessionID}, http.StatusOK, nil)
		status, _ := anonymous(t).do(http.MethodPost, "/tokens/refresh", obj{"refresh_token": u.Refresh})
		require.Equal(t, http.StatusUnauthorized, status, "revoked session must not mint new tokens")
		u.login(t)
	})

	t.Run("password reset via emailed token", func(t *testing.T) {
		// Same answer for known and unknown emails (no enumeration).
		anonymous(t).expect(http.MethodPost, "/password-reset/request", obj{"email": u.Email}, http.StatusNoContent, nil)
		anonymous(t).expect(http.MethodPost, "/password-reset/request", obj{"email": "ghost_" + randomHex(4) + "@example.com"}, http.StatusNoContent, nil)

		raw := randomHex(32)
		_, err := store.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
			UserID: u.ID, TokenHash: util.HashToken(raw), ExpiresAt: time.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		resetPw := "Reset-" + randomHex(4)
		anonymous(t).expect(http.MethodPost, "/password-reset/confirm", obj{"token": raw, "new_password": resetPw}, http.StatusNoContent, nil)
		status, _ := anonymous(t).do(http.MethodPost, "/password-reset/confirm", obj{"token": raw, "new_password": "another-123"})
		require.GreaterOrEqual(t, status, 400, "reset tokens are single use")

		u.Password = resetPw
		u.login(t)
	})

	t.Run("deleted account is locked out", func(t *testing.T) {
		u.api.expect(http.MethodDelete, "/users/me", obj{"password": u.Password}, http.StatusNoContent, nil)
		status, _ := u.api.do(http.MethodGet, "/users/me", nil)
		require.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, status)
		status, _ = anonymous(t).do(http.MethodPost, "/users/login", obj{"email": u.Email, "password": u.Password})
		require.GreaterOrEqual(t, status, 400)
	})
}

// The firmware registers, the user claims it, a soil sensor reports over
// MQTT, and the readings flow into devices, soil health and irrigation.
// A second user must not be able to touch any of it.
func TestE2E_SensorTelemetrySoilHealthAndOwnership(t *testing.T) {
	ctx := context.Background()
	owner := newUser(t)
	intruder := newUser(t)
	ctrl := claimController(t, owner)

	t.Run("controller can't be claimed twice", func(t *testing.T) {
		status, _ := intruder.api.do(http.MethodPut, "/controller-devices/claim/"+ctrl.UID, nil)
		require.GreaterOrEqual(t, status, 400)
	})

	sensorUID := randomDeviceUID()
	sensor := newFakeFirmware(t, sensorUID)
	anonymous(t).expect(http.MethodPost, "/devices/register", obj{
		"device_uid": sensorUID, "name": "Horta E2E", "status": "ATIVO",
	}, http.StatusOK, nil)
	dbDevice, err := store.GetDeviceByUID(ctx, sensorUID)
	require.NoError(t, err)

	t.Run("only the controller owner can link a sensor to it", func(t *testing.T) {
		status, _ := intruder.api.do(http.MethodPut, "/devices/link/"+dbDevice.QrToken+"?controller_uuid="+ctrl.UUID, nil)
		require.GreaterOrEqual(t, status, 400)
	})
	var linked obj
	owner.api.expect(http.MethodPut, "/devices/link/"+dbDevice.QrToken+"?controller_uuid="+ctrl.UUID, nil, http.StatusOK, &linked)
	deviceUUID := linked["uuid"].(string)

	listDevices := func(u *testUser) []obj {
		var out []obj
		u.api.expect(http.MethodGet, "/devices?page_id=1&page_size=10&controller_uuid="+ctrl.UUID, nil, http.StatusOK, &out)
		return out
	}

	t.Run("telemetry from the new firmware reaches the API", func(t *testing.T) {
		sensor.publish("telemetry", obj{"soil_moisture": 55, "soil_raw": 2100, "sensor_ok": true, "rssi": -60})
		eventually(t, "moisture visible in device list", func() bool {
			d := listDevices(owner)
			return len(d) == 1 && d[0]["soil_moisture"] == float64(55) && d[0]["sensor_ok"] == true
		})

		var health obj
		owner.api.expect(http.MethodGet, "/statistics/soil-health", nil, http.StatusOK, &health)
		require.Equal(t, "EXCELLENT", health["status"])
		require.Equal(t, float64(55), health["avg_moisture"])
	})

	t.Run("a faulty probe is flagged and its reading ignored", func(t *testing.T) {
		sensor.publish("telemetry", obj{"soil_raw": 4095, "sensor_ok": false, "rssi": -61})
		eventually(t, "sensor marked as faulty", func() bool {
			var health obj
			owner.api.expect(http.MethodGet, "/statistics/soil-health", nil, http.StatusOK, &health)
			sensors := health["sensors"].([]any)
			return len(sensors) == 1 && sensors[0].(obj)["state"] == "FAULT"
		})
		require.Equal(t, float64(55), listDevices(owner)[0]["soil_moisture"], "last good reading kept")
	})

	t.Run("intruder sees and changes nothing", func(t *testing.T) {
		status, raw := intruder.api.do(http.MethodGet, "/devices?page_id=1&page_size=10&controller_uuid="+ctrl.UUID, nil)
		if status == http.StatusOK {
			require.JSONEq(t, "[]", string(raw), "another user's sensors must not be listed")
		}
		status, _ = intruder.api.do(http.MethodPut, "/devices/name/"+deviceUUID, obj{"name": "pwned"})
		require.Equal(t, http.StatusForbidden, status)
		status, _ = intruder.api.do(http.MethodPut, "/devices/"+deviceUUID+"/location", obj{"latitude": 1, "longitude": 1})
		require.Equal(t, http.StatusForbidden, status)
		status, _ = intruder.api.do(http.MethodPost, "/irrigation/commands", obj{"device_id": deviceUUID, "action": "START", "duration": 30})
		require.GreaterOrEqual(t, status, 400)

		var health obj
		intruder.api.expect(http.MethodGet, "/statistics/soil-health", nil, http.StatusOK, &health)
		require.Equal(t, "NO_DEVICES", health["status"])
	})

	t.Run("owner places the sensor on the map", func(t *testing.T) {
		owner.api.expect(http.MethodPut, "/devices/"+deviceUUID+"/location", obj{"latitude": -23.55, "longitude": -46.63}, http.StatusOK, nil)
		require.Equal(t, -23.55, listDevices(owner)[0]["latitude"])
		status, _ := owner.api.do(http.MethodPut, "/devices/"+deviceUUID+"/location", obj{"latitude": 91, "longitude": 0})
		require.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("manual irrigation round-trips through the firmware", func(t *testing.T) {
		var cmdRsp obj
		owner.api.expect(http.MethodPost, "/irrigation/commands", obj{"device_id": deviceUUID, "action": "START", "duration": 30}, http.StatusCreated, &cmdRsp)
		commandID := cmdRsp["command_id"].(string)

		cmd := sensor.awaitCommand("START")
		require.Equal(t, commandID, cmd["command_id"])
		sensor.publish("events", obj{"command_id": commandID, "status": "SUCCESS"})

		eventually(t, "command acknowledged", func() bool {
			var c obj
			owner.api.expect(http.MethodGet, "/irrigation/command/"+commandID, nil, http.StatusOK, &c)
			return c["status"] == "SUCCESS"
		})
	})
}

func publishFirmware(t *testing.T, admin *testUser, version string, image []byte) (int, []byte) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "firmware.bin")
	require.NoError(t, err)
	_, err = part.Write(image)
	require.NoError(t, err)
	require.NoError(t, w.WriteField("version", version))
	require.NoError(t, w.WriteField("build", "1"))
	require.NoError(t, w.WriteField("release_notes", "E2E release"))
	require.NoError(t, w.Close())

	req, err := http.NewRequest(http.MethodPost, apiBase+"/admin/firmware", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return admin.api.send(req)
}

func nextFirmwareVersion(t *testing.T) string {
	latest, err := store.GetLatestFirmwareRelease(context.Background())
	if err != nil {
		return "2.0.0"
	}
	major, err := strconv.Atoi(strings.Split(latest.Version, ".")[0])
	require.NoError(t, err)
	return fmt.Sprintf("%d.0.0", major+1)
}

// Admin publishes a release; the owner is notified, triggers the update and
// the (fake) controller downloads it, verifies the hash and reports back.
func TestE2E_FirmwareOtaUpdate(t *testing.T) {
	admin := newUser(t)
	makeAdmin(t, admin)
	owner := newUser(t)
	intruder := newUser(t)
	ctrl := claimController(t, owner)

	image := append([]byte{0xE9}, bytes.Repeat([]byte(randomHex(64)), 1024)...)
	version := nextFirmwareVersion(t)

	t.Run("only admins publish", func(t *testing.T) {
		status, _ := publishFirmware(t, owner, version, image)
		require.Equal(t, http.StatusForbidden, status)
	})

	status, raw := publishFirmware(t, admin, version, image)
	require.Equalf(t, http.StatusCreated, status, "publish: %s", raw)

	t.Run("controller downloads exactly the published image", func(t *testing.T) {
		var manifest obj
		anonymous(t).expect(http.MethodGet, "/firmware/manifest", nil, http.StatusOK, &manifest)
		require.Equal(t, version, manifest["version"])
		sum := sha256.Sum256(image)
		require.Equal(t, hex.EncodeToString(sum[:]), manifest["sha256"])

		rsp, err := http.Get(strings.TrimSuffix(apiBase, "/api/v1") + manifest["path"].(string))
		require.NoError(t, err)
		defer rsp.Body.Close()
		downloaded, err := io.ReadAll(rsp.Body)
		require.NoError(t, err)
		require.Equal(t, image, downloaded)
	})

	t.Run("owner is told an update exists", func(t *testing.T) {
		var fw obj
		owner.api.expect(http.MethodGet, "/controller-devices/"+ctrl.UUID+"/firmware", nil, http.StatusOK, &fw)
		require.Equal(t, true, fw["update_available"])
		require.Equal(t, version, fw["latest_version"])

		var notifications []obj
		owner.api.expect(http.MethodGet, "/notifications", nil, http.StatusOK, &notifications)
		found := false
		for _, n := range notifications {
			found = found || n["type"] == "FIRMWARE_UPDATE"
		}
		require.True(t, found, "owner should get a FIRMWARE_UPDATE notification")
	})

	t.Run("intruder can't see or trigger it", func(t *testing.T) {
		status, _ := intruder.api.do(http.MethodGet, "/controller-devices/"+ctrl.UUID+"/firmware", nil)
		require.Equal(t, http.StatusForbidden, status)
		status, _ = intruder.api.do(http.MethodPost, "/controller-devices/"+ctrl.UUID+"/firmware/update", nil)
		require.Equal(t, http.StatusForbidden, status)
	})

	t.Run("update runs end to end", func(t *testing.T) {
		owner.api.expect(http.MethodPost, "/controller-devices/"+ctrl.UUID+"/firmware/update", nil, http.StatusAccepted, nil)
		cmd := ctrl.firmware.awaitCommand("OTA_UPDATE")
		require.Equal(t, version, cmd["data"].(obj)["version"])

		status, _ := owner.api.do(http.MethodPost, "/controller-devices/"+ctrl.UUID+"/firmware/update", nil)
		require.Equal(t, http.StatusConflict, status, "no double update while one is running")

		ctrl.firmware.publish("events", obj{"ota_status": "DOWNLOADING", "version": version})

		// The unauthenticated register endpoint must not be able to change
		// an owned controller's version (that could suppress updates).
		anonymous(t).expect(http.MethodPost, "/controller-devices/register", obj{
			"device_uid": ctrl.UID, "firmware_version": "999.0.0", "firmware_build": "1",
		}, http.StatusOK, nil)

		// After the reboot the firmware reports its version in telemetry
		// and confirms the update over MQTT.
		ctrl.firmware.publish("telemetry", obj{"status": "ONLINE", "firmware_version": version, "firmware_build": "1"})
		ctrl.firmware.publish("events", obj{"ota_status": "SUCCESS", "version": version})

		eventually(t, "update reported as successful", func() bool {
			var fw obj
			owner.api.expect(http.MethodGet, "/controller-devices/"+ctrl.UUID+"/firmware", nil, http.StatusOK, &fw)
			return fw["ota_status"] == "SUCCESS" && fw["current_version"] == version && fw["update_available"] == false
		})
	})

	t.Run("auto-update preference persists", func(t *testing.T) {
		var fw obj
		owner.api.expect(http.MethodPut, "/controller-devices/"+ctrl.UUID+"/firmware/auto-update", obj{"enabled": true}, http.StatusOK, &fw)
		require.Equal(t, true, fw["auto_update"])
	})
}

func dialWS(t *testing.T, u *testUser) *websocket.Conn {
	t.Helper()
	header := http.Header{"Authorization": []string{"Bearer " + u.api.token}}
	conn, rsp, err := websocket.DefaultDialer.Dial(wsBase+"/ws", header)
	if rsp != nil {
		rsp.Body.Close()
	}
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// awaitWSEvent reads frames until one has the given type.
func awaitWSEvent(t *testing.T, conn *websocket.Conn, eventType string) obj {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	require.NoError(t, conn.SetReadDeadline(deadline))
	for {
		_, raw, err := conn.ReadMessage()
		require.NoErrorf(t, err, "waiting for websocket event %s", eventType)
		var ev obj
		if json.Unmarshal(raw, &ev) == nil && ev["type"] == eventType {
			return ev
		}
	}
}

// A user writes to support; an admin answers; the reply arrives live over
// the websocket and as a notification.
func TestE2E_SupportChatLiveReply(t *testing.T) {
	user := newUser(t)
	other := newUser(t)
	admin := newUser(t)
	makeAdmin(t, admin)

	userWS := dialWS(t, user)

	user.api.expect(http.MethodPost, "/support/messages", obj{"body": "Meu sensor não aparece"}, http.StatusCreated, nil)
	status, _ := user.api.do(http.MethodPost, "/support/messages", obj{"body": "   "})
	require.Equal(t, http.StatusBadRequest, status)
	status, _ = user.api.do(http.MethodGet, "/admin/support/conversations", nil)
	require.Equal(t, http.StatusForbidden, status)

	var conversations []obj
	admin.api.expect(http.MethodGet, "/admin/support/conversations?page_size=100", nil, http.StatusOK, &conversations)
	var mine obj
	for _, c := range conversations {
		if c["user_uuid"] == user.UUID {
			mine = c
		}
	}
	require.NotNil(t, mine, "conversation listed for admin")
	require.Equal(t, float64(1), mine["unread_count"])

	admin.api.expect(http.MethodPost, "/admin/support/conversations/"+user.UUID+"/messages",
		obj{"body": "Reinicie a controladora, por favor."}, http.StatusCreated, nil)

	ev := awaitWSEvent(t, userWS, "support_message")
	require.Equal(t, "Reinicie a controladora, por favor.", ev["body"])

	var messages []obj
	user.api.expect(http.MethodGet, "/support/messages", nil, http.StatusOK, &messages)
	require.Len(t, messages, 2)
	require.Equal(t, "SUPPORT", messages[0]["sender"])

	// Conversations are private to their owner.
	var otherMessages []obj
	other.api.expect(http.MethodGet, "/support/messages", nil, http.StatusOK, &otherMessages)
	require.Empty(t, otherMessages)
}

// Every protected route must reject anonymous and forged tokens, admin
// routes must reject normal users, and a banned user is locked out at once.
func TestE2E_AccessControl(t *testing.T) {
	someUUID := uuid.NewString()
	protected := []struct{ method, path string }{
		{http.MethodGet, "/users/me"},
		{http.MethodPut, "/users/name"},
		{http.MethodPut, "/users/password"},
		{http.MethodDelete, "/users/me"},
		{http.MethodGet, "/devices?page_id=1&page_size=5"},
		{http.MethodPut, "/devices/name/" + someUUID},
		{http.MethodPut, "/devices/" + someUUID + "/location"},
		{http.MethodGet, "/controller-devices?page_id=1&page_size=5"},
		{http.MethodGet, "/controller-devices/" + someUUID + "/firmware"},
		{http.MethodPost, "/controller-devices/" + someUUID + "/firmware/update"},
		{http.MethodPost, "/irrigation/commands"},
		{http.MethodGet, "/statistics/soil-health"},
		{http.MethodGet, "/notifications"},
		{http.MethodGet, "/support/messages"},
		{http.MethodGet, "/firmware/latest"},
		{http.MethodGet, "/ws"},
	}

	for _, r := range protected {
		t.Run("anonymous "+r.method+" "+r.path, func(t *testing.T) {
			status, _ := anonymous(t).do(r.method, r.path, nil)
			require.Equal(t, http.StatusUnauthorized, status)
		})
		t.Run("forged token "+r.method+" "+r.path, func(t *testing.T) {
			status, _ := apiClient{t: t, token: "v2.local.forged-" + randomHex(16)}.do(r.method, r.path, nil)
			require.Equal(t, http.StatusUnauthorized, status)
		})
	}

	user := newUser(t)
	admin := newUser(t)
	makeAdmin(t, admin)

	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/admin/users?page_id=1&page_size=5"},
		{http.MethodPut, "/admin/users/" + someUUID + "/ban"},
		{http.MethodGet, "/admin/support/conversations"},
		{http.MethodPost, "/admin/firmware"},
		{http.MethodGet, "/admin/firmware"},
	} {
		t.Run("non-admin "+r.method+" "+r.path, func(t *testing.T) {
			status, _ := user.api.do(r.method, r.path, nil)
			require.Equal(t, http.StatusForbidden, status)
		})
	}

	t.Run("banned user is locked out immediately", func(t *testing.T) {
		victim := newUser(t)
		admin.api.expect(http.MethodPut, "/admin/users/"+victim.UUID+"/ban", obj{"reason": "e2e"}, http.StatusOK, nil)
		status, _ := victim.api.do(http.MethodGet, "/users/me", nil)
		require.Equal(t, http.StatusForbidden, status)
		status, _ = anonymous(t).do(http.MethodPost, "/users/login", obj{"email": victim.Email, "password": victim.Password})
		require.GreaterOrEqual(t, status, 400)
	})
}
