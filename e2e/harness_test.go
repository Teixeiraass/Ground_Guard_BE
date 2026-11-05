//go:build e2e

// Package e2e runs the real API in-process against a real Postgres database
// and a real MQTT broker, and drives it over HTTP, WebSocket and MQTT the way
// the app and the ESP32 firmware do. Run with `make e2e`.
package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/handler"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/push"
	"github.com/Teixeiraass/ground_guard_be/util"
	mqttgo "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	apiBase     string
	wsBase      string
	store       db.Store
	cfg         util.Config
	topicPrefix string
)

func TestMain(m *testing.M) {
	var err error
	cfg, err = util.LoadConfig("..")
	if err != nil {
		log.Fatal("cannot load config: ", err)
	}
	// The suite creates and deletes data freely: never point it at a real DB.
	if !strings.Contains(cfg.DBSource, "_e2e") {
		log.Fatal("refusing to run: DB_SOURCE must point to a *_e2e database (use `make e2e`)")
	}
	if !cfg.MQTTEnabled {
		log.Fatal("e2e needs MQTT_ENABLED=true and a running broker")
	}

	fwDir, err := os.MkdirTemp("", "gg-e2e-firmware-")
	if err != nil {
		log.Fatal(err)
	}
	cfg.FirmwareStorageDir = fwDir
	cfg.SMTPHost = ""
	cfg.TurnstileSecretKey = ""
	// Real production protections stay on. Each simulated client gets its own
	// IP through X-Forwarded-For, trusted only from the local "proxy".
	cfg.RateLimitEnabled = true
	cfg.TrustedProxies = "127.0.0.1,::1"
	cfg.FirmwareSigningKeyFile = filepath.Join("..", "keys", "firmware_signing_dev.pem")
	topicPrefix = cfg.MQTTTopicPrefix

	conn, err := sql.Open(cfg.DBDriver, cfg.DBSource)
	if err != nil {
		log.Fatal(err)
	}
	store = db.NewStore(conn)

	mqttClient, err := client.NewPahoClient(cfg)
	if err != nil {
		log.Fatal("cannot connect to mqtt broker: ", err)
	}

	server, err := handler.NewServer(cfg, store, mqttClient, push.NewNoopSender())
	if err != nil {
		log.Fatal("cannot create server: ", err)
	}
	ts := httptest.NewServer(server.HTTPHandler())
	apiBase = ts.URL + "/api/v1"
	wsBase = "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1"

	code := m.Run()

	ts.Close()
	mqttClient.Close()
	conn.Close()
	os.RemoveAll(fwDir)
	os.Exit(code)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// randomDeviceUID mimics a MAC-derived ESP32 UID: 12 uppercase hex chars.
func randomDeviceUID() string {
	return strings.ToUpper(randomHex(6))
}

// ---------- HTTP ----------

type apiClient struct {
	t     *testing.T
	token string
	ip    string
}

// randomIP gives each simulated client its own address (TEST-NET-2 range).
func randomIP() string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return fmt.Sprintf("198.51.%d.%d", b[0], b[1])
}

func anonymous(t *testing.T) apiClient { return apiClient{t: t, ip: randomIP()} }

func (c apiClient) do(method, path string, body any) (int, []byte) {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, apiBase+path, reader)
	require.NoError(c.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.send(req)
}

func (c apiClient) send(req *http.Request) (int, []byte) {
	c.t.Helper()
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.ip != "" {
		req.Header.Set("X-Forwarded-For", c.ip)
	}
	rsp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err)
	defer rsp.Body.Close()
	raw, err := io.ReadAll(rsp.Body)
	require.NoError(c.t, err)
	return rsp.StatusCode, raw
}

// expect asserts the status and decodes the JSON body into out (if non-nil).
func (c apiClient) expect(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()
	status, raw := c.do(method, path, body)
	require.Equalf(c.t, wantStatus, status, "%s %s -> %s", method, path, raw)
	if out != nil && len(raw) > 0 {
		require.NoError(c.t, json.Unmarshal(raw, out), "body: %s", raw)
	}
}

type obj = map[string]any

// ---------- users ----------

type testUser struct {
	ID        int64
	UUID      string
	Email     string
	Password  string
	SessionID string
	Refresh   string
	ip        string
	api       apiClient
}

func newUser(t *testing.T) *testUser {
	t.Helper()
	suffix := randomHex(5)
	u := &testUser{Email: "e2e_" + suffix + "@example.com", Password: "S3nha-forte-" + suffix, ip: randomIP()}
	apiClient{t: t, ip: u.ip}.expect(http.MethodPost, "/users", obj{
		"username":  "e2e" + suffix,
		"full_name": "E2E " + suffix,
		"email":     u.Email,
		"password":  u.Password,
	}, http.StatusCreated, nil)
	u.login(t)

	dbUser, err := store.GetUserByEmail(context.Background(), u.Email)
	require.NoError(t, err)
	u.ID = dbUser.ID
	u.UUID = dbUser.Uuid.String()
	return u
}

func (u *testUser) login(t *testing.T) {
	t.Helper()
	var rsp obj
	apiClient{t: t, ip: u.ip}.expect(http.MethodPost, "/users/login",
		obj{"email": u.Email, "password": u.Password}, http.StatusOK, &rsp)
	u.api = apiClient{t: t, token: rsp["access_token"].(string), ip: u.ip}
	u.SessionID = rsp["session_id"].(string)
	u.Refresh = rsp["refresh_token"].(string)
}

func makeAdmin(t *testing.T, u *testUser) {
	t.Helper()
	_, err := testDB(t).Exec("UPDATE users SET is_admin = true WHERE id = $1", u.ID)
	require.NoError(t, err)
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open(cfg.DBDriver, cfg.DBSource)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// ---------- MQTT fake firmware ----------

type fakeFirmware struct {
	t        *testing.T
	uid      string
	client   mqttgo.Client
	commands chan obj
}

// newFakeFirmware connects to the broker as the ESP32 with this UID would
// and records every command the API sends it.
func newFakeFirmware(t *testing.T, uid string) *fakeFirmware {
	t.Helper()
	f := &fakeFirmware{t: t, uid: uid, commands: make(chan obj, 16)}

	opts := mqttgo.NewClientOptions().
		AddBroker(cfg.MQTTBrokerURL).
		// The real firmware uses its UID as client id; the broker ACL relies on it.
		SetClientID(uid).
		SetConnectTimeout(5 * time.Second)
	f.client = mqttgo.NewClient(opts)
	tok := f.client.Connect()
	require.True(t, tok.WaitTimeout(5*time.Second), "fake firmware connect timeout")
	require.NoError(t, tok.Error())

	sub := f.client.Subscribe(f.topic("commands"), 1, func(_ mqttgo.Client, msg mqttgo.Message) {
		var cmd obj
		if json.Unmarshal(msg.Payload(), &cmd) == nil {
			f.commands <- cmd
		}
	})
	require.True(t, sub.WaitTimeout(5*time.Second))
	require.NoError(t, sub.Error())

	t.Cleanup(func() { f.client.Disconnect(100) })
	return f
}

func (f *fakeFirmware) topic(suffix string) string {
	return fmt.Sprintf("%s/devices/%s/%s", topicPrefix, f.uid, suffix)
}

func (f *fakeFirmware) publish(suffix string, payload any) {
	f.t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(f.t, err)
	tok := f.client.Publish(f.topic(suffix), 1, false, raw)
	require.True(f.t, tok.WaitTimeout(5*time.Second))
	require.NoError(f.t, tok.Error())
}

// awaitCommand returns the next command with the given action.
func (f *fakeFirmware) awaitCommand(action string) obj {
	f.t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case cmd := <-f.commands:
			if cmd["action"] == action {
				return cmd
			}
		case <-deadline:
			f.t.Fatalf("no %s command received by fake firmware %s", action, f.uid)
			return nil
		}
	}
}

// eventually polls until cond holds, for the async MQTT -> DB -> API paths.
func eventually(t *testing.T, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("condition not met in time: %s", msg)
}

// ---------- shared setup ----------

type claimedController struct {
	UID      string
	UUID     string
	firmware *fakeFirmware
}

// claimController registers a controller the way the firmware does on boot,
// then claims it as the user the way the app does after BLE provisioning.
func claimController(t *testing.T, owner *testUser) claimedController {
	t.Helper()
	uid := randomDeviceUID()
	fw := newFakeFirmware(t, uid)

	anonymous(t).expect(http.MethodPost, "/controller-devices/register", obj{
		"device_uid":       uid,
		"firmware_version": "1.0.0",
		"firmware_build":   "1",
	}, http.StatusOK, nil)

	var claimed obj
	owner.api.expect(http.MethodPut, "/controller-devices/claim/"+uid, nil, http.StatusOK, &claimed)

	cmd := fw.awaitCommand("CONFIGURE_CONTROLLER")
	fw.publish("events", obj{"command_id": cmd["command_id"], "status": "SUCCESS"})

	// Online state comes from the firmware's retained state message.
	fw.publish("state", obj{"status": "ONLINE", "is_online": true, "is_irrigating": false})
	eventually(t, "controller online", func() bool {
		c, err := store.GetControllerDeviceByUID(context.Background(), uid)
		return err == nil && c.IsOnline
	})

	return claimedController{UID: uid, UUID: claimed["uuid"].(string), firmware: fw}
}
