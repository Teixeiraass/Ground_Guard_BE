package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRateLimiterBurstThenBlocks(t *testing.T) {
	l := NewRateLimiter(time.Hour, 3)
	for i := 0; i < 3; i++ {
		ok, _ := l.Allow("k")
		require.True(t, ok, "request %d within burst", i)
	}
	ok, wait := l.Allow("k")
	require.False(t, ok)
	require.Greater(t, wait, time.Duration(0))

	ok, _ = l.Allow("other-key")
	require.True(t, ok, "keys are independent")
}

func TestRateLimiterRefills(t *testing.T) {
	l := NewRateLimiter(20*time.Millisecond, 1)
	ok, _ := l.Allow("k")
	require.True(t, ok)
	ok, _ = l.Allow("k")
	require.False(t, ok)
	time.Sleep(30 * time.Millisecond)
	ok, _ = l.Allow("k")
	require.True(t, ok)
}

func TestDisabledLimiterAlwaysAllows(t *testing.T) {
	l := Disabled()
	for i := 0; i < 1000; i++ {
		ok, _ := l.Allow("k")
		require.True(t, ok)
	}
}

func newEngine(handlers ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(handlers...)
	r.POST("/*path", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})
	return r
}

func TestLimitByIPReturns429WithRetryAfter(t *testing.T) {
	r := newEngine(LimitByIP("test", NewRateLimiter(time.Hour, 2)))
	codes := []int{}
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.RemoteAddr = "203.0.113.7:1234"
		r.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
		if rec.Code == http.StatusTooManyRequests {
			require.NotEmpty(t, rec.Header().Get("Retry-After"))
			require.Contains(t, rec.Body.String(), "RATE_LIMITED")
		}
	}
	require.Equal(t, []int{200, 200, 429}, codes)
}

func TestLimitByIPIgnoresSpoofedForwardedFor(t *testing.T) {
	// No trusted proxies: X-Forwarded-For must not create fresh buckets.
	r := newEngine(LimitByIP("test", NewRateLimiter(time.Hour, 1)))
	require.NoError(t, r.SetTrustedProxies(nil))

	var last int
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.RemoteAddr = "203.0.113.7:1234"
		req.Header.Set("X-Forwarded-For", "10.0.0."+string(rune('1'+i)))
		r.ServeHTTP(rec, req)
		last = rec.Code
	}
	require.Equal(t, http.StatusTooManyRequests, last)
}

func TestSecurityHeaders(t *testing.T) {
	r := newEngine(SecurityHeaders())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", nil))

	h := rec.Header()
	require.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", h.Get("X-Frame-Options"))
	require.Equal(t, "no-referrer", h.Get("Referrer-Policy"))
	require.Empty(t, h.Get("Strict-Transport-Security"), "HSTS only over HTTPS")

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	r.ServeHTTP(rec, req)
	require.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
}

func TestLimitBodyPerRoute(t *testing.T) {
	r := newEngine(LimitBody())
	post := func(path string, size int64) int {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(make([]byte, size))))
		return rec.Code
	}

	require.Equal(t, http.StatusOK, post("/api/v1/users", defaultMaxBody))
	require.Equal(t, http.StatusRequestEntityTooLarge, post("/api/v1/users", defaultMaxBody+1))
	require.Equal(t, http.StatusOK, post("/api/v1/admin/firmware", 2<<20))
	require.Equal(t, http.StatusRequestEntityTooLarge, post("/api/v1/admin/firmware", firmwareMaxBody+1))
	require.Equal(t, http.StatusOK, post("/api/v1/users/profile-image", 5<<20))
}

func TestLimitBodyCatchesChunkedBodiesWithoutLength(t *testing.T) {
	r := newEngine(LimitBody())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", io.NopCloser(strings.NewReader(strings.Repeat("a", int(defaultMaxBody)+10))))
	req.ContentLength = -1
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestBlockedAndHitOnlyCountFailures(t *testing.T) {
	l := NewRateLimiter(time.Hour, 2)
	for i := 0; i < 10; i++ {
		blocked, _ := l.Blocked("acct")
		require.False(t, blocked, "checking alone never consumes budget")
	}
	l.Hit("acct")
	blocked, _ := l.Blocked("acct")
	require.False(t, blocked)
	l.Hit("acct")
	blocked, wait := l.Blocked("acct")
	require.True(t, blocked)
	require.Greater(t, wait, time.Duration(0))
}

func TestEvictIdleKeepsRecentAndPenalisedKeys(t *testing.T) {
	l := NewRateLimiter(time.Minute, 5)
	// idleTTL must cover a full refill (burst+1 intervals), never less than 10 min.
	require.Equal(t, 10*time.Minute, l.idleTTL)
	long := NewRateLimiter(time.Hour, 3)
	require.Equal(t, 4*time.Hour, long.idleTTL)

	l.Allow("old")
	l.Allow("recent")
	l.mu.Lock()
	l.visitors["old"].lastSeen = time.Now().Add(-11 * time.Minute)
	l.mu.Unlock()

	require.Equal(t, 1, l.evictIdle(time.Now()))
	l.mu.Lock()
	_, oldKept := l.visitors["old"]
	_, recentKept := l.visitors["recent"]
	l.mu.Unlock()
	require.False(t, oldKept)
	require.True(t, recentKept)
}
