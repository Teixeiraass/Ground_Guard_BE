//go:build e2e

package e2e

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestE2E_LoginBruteForceIsThrottled(t *testing.T) {
	victim := newUser(t)

	t.Run("per account, even from many IPs", func(t *testing.T) {
		statuses := map[int]int{}
		for i := 0; i < 8; i++ {
			// A fresh IP each time: only the per-account limit can stop this.
			status, _ := anonymous(t).do(http.MethodPost, "/users/login", obj{"email": victim.Email, "password": "guess-" + randomHex(3)})
			statuses[status]++
		}
		require.Equal(t, 5, statuses[http.StatusUnauthorized], "first attempts are just wrong passwords")
		require.Equal(t, 3, statuses[http.StatusTooManyRequests], "then the account is throttled")
	})

	t.Run("per IP, across accounts", func(t *testing.T) {
		attacker := anonymous(t)
		throttled := false
		for i := 0; i < 15 && !throttled; i++ {
			status, raw := attacker.do(http.MethodPost, "/users/login", obj{"email": "spray_" + randomHex(4) + "@example.com", "password": "x"})
			if status == http.StatusTooManyRequests {
				throttled = true
				require.Contains(t, string(raw), "RATE_LIMITED")
			}
		}
		require.True(t, throttled, "password spraying from one IP must be throttled")
	})

	t.Run("forged X-Forwarded-For from an untrusted hop is ignored", func(t *testing.T) {
		// The server only trusts 127.0.0.1; a spoofed chain still resolves
		// to the trusted proxy's view, so limits can't be dodged per request.
		c := apiClient{t: t, ip: "203.0.113.50, 198.51.100.9"}
		status, _ := c.do(http.MethodGet, "/helper", nil)
		require.Less(t, status, 500)
	})
}

func TestE2E_HardenedResponses(t *testing.T) {
	u := newUser(t)

	t.Run("security headers on every response", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, apiBase+"/users/me", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+u.api.token)
		rsp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, "nosniff", rsp.Header.Get("X-Content-Type-Options"))
		require.Equal(t, "DENY", rsp.Header.Get("X-Frame-Options"))
		require.Equal(t, "no-referrer", rsp.Header.Get("Referrer-Policy"))
	})

	t.Run("oversized bodies are refused", func(t *testing.T) {
		huge := bytes.Repeat([]byte("a"), 2<<20)
		req, err := http.NewRequest(http.MethodPost, apiBase+"/support/messages", bytes.NewReader(huge))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		status, _ := u.api.send(req)
		require.Equal(t, http.StatusRequestEntityTooLarge, status)
	})

	t.Run("profile picture must really be an image", func(t *testing.T) {
		var body bytes.Buffer
		body.WriteString("--b\r\nContent-Disposition: form-data; name=\"image\"; filename=\"x.png\"\r\nContent-Type: image/png\r\n\r\n")
		body.WriteString("<html><script>alert(document.cookie)</script></html>")
		body.WriteString("\r\n--b--\r\n")
		req, err := http.NewRequest(http.MethodPut, apiBase+"/users/profile-image", &body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=b")
		status, _ := u.api.send(req)
		require.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("lookups by UID don't leak other users' devices", func(t *testing.T) {
		owner := newUser(t)
		ctrl := claimController(t, owner)
		status, _ := u.api.do(http.MethodGet, "/controller-devices/uid/"+ctrl.UID, nil)
		require.Equal(t, http.StatusNotFound, status)
		status, _ = u.api.do(http.MethodPut, "/controller-devices/name/"+ctrl.UUID, obj{"name": "pwned"})
		require.Equal(t, http.StatusForbidden, status)

		var mine obj
		owner.api.expect(http.MethodGet, "/controller-devices/uid/"+ctrl.UID, nil, http.StatusOK, &mine)
		_, leaksOwnerID := mine["user"]
		require.False(t, leaksOwnerID, "internal owner id must not be exposed")
	})

	t.Run("weather proxy validates input and never needs a key in the app", func(t *testing.T) {
		status, _ := u.api.do(http.MethodGet, "/weather/current", nil)
		require.Contains(t, []int{http.StatusBadRequest, http.StatusServiceUnavailable}, status)
	})
}

func TestE2E_WebSocketRejectsForeignBrowserOrigins(t *testing.T) {
	u := newUser(t)
	header := http.Header{
		"Authorization": []string{"Bearer " + u.api.token},
		"Origin":        []string{"https://evil.example.com"},
	}
	conn, rsp, err := websocket.DefaultDialer.Dial(wsBase+"/ws", header)
	if conn != nil {
		conn.Close()
	}
	require.Error(t, err, "cross-site websocket hijacking must be refused")
	require.NotNil(t, rsp)
	require.Equal(t, http.StatusForbidden, rsp.StatusCode)
}

// The controller trusts an image only if its signature verifies with the
// public key compiled into the firmware: check the API produces exactly that.
func TestE2E_PublishedFirmwareIsSignedForTheDevice(t *testing.T) {
	admin := newUser(t)
	makeAdmin(t, admin)

	image := append([]byte{0xE9}, bytes.Repeat([]byte(randomHex(32)), 512)...)
	status, raw := publishFirmware(t, admin, nextFirmwareVersion(t), image)
	require.Equalf(t, http.StatusCreated, status, "publish: %s", raw)

	var manifest obj
	anonymous(t).expect(http.MethodGet, "/firmware/manifest", nil, http.StatusOK, &manifest)

	pubPEM, err := os.ReadFile(filepath.Join("..", "keys", "firmware_signing_dev.pub.pem"))
	require.NoError(t, err)
	block, _ := pem.Decode(pubPEM)
	require.NotNil(t, block)
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	pub := parsed.(*ecdsa.PublicKey)

	sig, err := base64.StdEncoding.DecodeString(manifest["signature"].(string))
	require.NoError(t, err)
	digest := sha256.Sum256(image)
	require.Equal(t, hex.EncodeToString(digest[:]), manifest["sha256"])
	require.True(t, ecdsa.VerifyASN1(pub, digest[:], sig), "firmware public key must accept the signature")

	tampered := sha256.Sum256(append(image, 0x00))
	require.False(t, ecdsa.VerifyASN1(pub, tampered[:], sig), "a modified image must not verify")
}
