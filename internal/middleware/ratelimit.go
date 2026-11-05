package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter is an in-memory token bucket per key (IP, email, user...).
// It is per process: behind several API replicas, move it to Redis.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
	idleTTL  time.Duration
	disabled bool
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter allows `burst` requests at once, refilled at one request
// per `every`. Example: NewRateLimiter(12*time.Second, 5) ≈ 5 per minute.
func NewRateLimiter(every time.Duration, burst int) *RateLimiter {
	l := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Every(every),
		burst:    burst,
		idleTTL:  time.Duration(burst+1) * every,
	}
	if l.idleTTL < 10*time.Minute {
		l.idleTTL = 10 * time.Minute
	}
	go l.cleanup()
	return l
}

// Disabled returns a limiter that always allows; used when rate limiting is
// switched off by configuration.
func Disabled() *RateLimiter {
	return &RateLimiter{disabled: true}
}

func (l *RateLimiter) get(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.visitors[key] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// Allow consumes one request for key and returns whether it is allowed and,
// if not, how long the caller should wait.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	if l.disabled {
		return true, 0
	}
	reservation := l.get(key).Reserve()
	if !reservation.OK() {
		return false, time.Minute
	}
	delay := reservation.Delay()
	if delay == 0 {
		return true, 0
	}
	reservation.Cancel()
	return false, delay
}

// Blocked reports whether key has no budget left, without consuming any.
// Paired with Hit, it limits only failed attempts (e.g. wrong passwords),
// so a legitimate user logging in often is never throttled.
func (l *RateLimiter) Blocked(key string) (bool, time.Duration) {
	if l.disabled {
		return false, 0
	}
	lim := l.get(key)
	if lim.Tokens() >= 1 {
		return false, 0
	}
	r := lim.Reserve()
	delay := r.Delay()
	r.Cancel()
	return true, delay
}

// Hit consumes one unit of budget for key (a failed attempt).
func (l *RateLimiter) Hit(key string) {
	if l.disabled {
		return
	}
	l.get(key).Allow()
}

func (l *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.evictIdle(time.Now())
	}
}

// evictIdle forgets keys unused for idleTTL so memory stays bounded. idleTTL
// covers a full refill, so evicting never hands a fresh burst to an
// attacker still inside their penalty window.
func (l *RateLimiter) evictIdle(now time.Time) int {
	cutoff := now.Add(-l.idleTTL)
	l.mu.Lock()
	defer l.mu.Unlock()
	evicted := 0
	for key, v := range l.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(l.visitors, key)
			evicted++
		}
	}
	return evicted
}

// TooManyRequests writes a 429 with Retry-After.
func TooManyRequests(ctx *gin.Context, wait time.Duration) {
	ctx.Header("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
	ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": "too many requests, try again later",
		"code":  "RATE_LIMITED",
	})
}

// LimitByIP rate limits a route group by client IP. ClientIP honours
// X-Forwarded-For only from the trusted proxies configured on the engine.
func LimitByIP(scope string, l *RateLimiter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ok, wait := l.Allow(scope + "|ip|" + ctx.ClientIP()); !ok {
			TooManyRequests(ctx, wait)
			return
		}
		ctx.Next()
	}
}

// LimitByUser rate limits by the authenticated user (falls back to IP).
func LimitByUser(scope string, l *RateLimiter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := scope + "|ip|" + ctx.ClientIP()
		if payload, ok := ctx.Get(AuthorizationPayloadKey); ok {
			if p, ok := payload.(*token.Payload); ok {
				key = scope + "|user|" + strconv.FormatInt(p.UserID, 10)
			}
		}
		if ok, wait := l.Allow(key); !ok {
			TooManyRequests(ctx, wait)
			return
		}
		ctx.Next()
	}
}
