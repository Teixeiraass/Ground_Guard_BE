package routes

import (
	"log"
	"strings"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Options struct {
	TrustedProxies   string
	RateLimitEnabled bool
}

// limits holds the per-route rate limiters for the router being built.
type limits struct {
	login, register, oauth, refresh  gin.HandlerFunc
	resetRequest, resetConfirm       gin.HandlerFunc
	verifyEmail, verifyResend        gin.HandlerFunc
	deviceRegister, firmwareDownload gin.HandlerFunc
	supportSend                      gin.HandlerFunc
}

func newLimits(enabled bool) limits {
	mk := func(every time.Duration, burst int) *middleware.RateLimiter {
		if !enabled {
			return middleware.Disabled()
		}
		return middleware.NewRateLimiter(every, burst)
	}
	return limits{
		login:            middleware.LimitByIP("login", mk(6*time.Second, 10)),
		register:         middleware.LimitByIP("register", mk(10*time.Minute, 5)),
		oauth:            middleware.LimitByIP("oauth", mk(3*time.Second, 20)),
		refresh:          middleware.LimitByIP("refresh", mk(2*time.Second, 30)),
		resetRequest:     middleware.LimitByIP("reset-request", mk(3*time.Minute, 5)),
		resetConfirm:     middleware.LimitByIP("reset-confirm", mk(90*time.Second, 10)),
		verifyEmail:      middleware.LimitByIP("verify-email", mk(90*time.Second, 10)),
		verifyResend:     middleware.LimitByUser("verify-resend", mk(20*time.Minute, 3)),
		deviceRegister:   middleware.LimitByIP("device-register", mk(2*time.Minute, 30)),
		firmwareDownload: middleware.LimitByIP("firmware", mk(2*time.Minute, 30)),
		supportSend:      middleware.LimitByUser("support-send", mk(6*time.Second, 10)),
	}
}

var rl limits

func Setup(tokenMaker token.Maker, store db.Store, handlers Handlers, opts Options) *gin.Engine {
	router := gin.Default()

	// Without this gin trusts X-Forwarded-For from anyone, so rate limits
	// and audit logs could be bypassed with a forged header.
	var proxies []string
	for _, p := range strings.Split(opts.TrustedProxies, ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies = append(proxies, p)
		}
	}
	if err := router.SetTrustedProxies(proxies); err != nil {
		log.Printf("invalid TRUSTED_PROXIES %q: %v", opts.TrustedProxies, err)
		_ = router.SetTrustedProxies(nil)
	}

	router.Use(middleware.SecurityHeaders(), middleware.LimitBody())

	rl = newLimits(opts.RateLimitEnabled)
	globalLimit := middleware.Disabled()
	if opts.RateLimitEnabled {
		globalLimit = middleware.NewRateLimiter(200*time.Millisecond, 300)
	}
	router.Use(middleware.LimitByIP("global", globalLimit))

	router.Static("/uploads/profile", "./uploads/profile")
	router.Static("/uploads/qrcodes", "./uploads/qrcodes")

	// API docs only outside production (GIN_MODE=release).
	if gin.Mode() != gin.ReleaseMode {
		registerSwaggerRoutes(router)
	}
	registerV1Routes(router, tokenMaker, store, handlers)

	return router
}

func registerSwaggerRoutes(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
