package handler

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	appoauth "github.com/Teixeiraass/ground_guard_be/internal/oauth"
	"github.com/Teixeiraass/ground_guard_be/internal/routes"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/internal/worker"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/mqtt/subscriber"
	"github.com/Teixeiraass/ground_guard_be/push"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config                  util.Config
	store                   db.Store
	tokenMaker              token.Maker
	oauthService            *appoauth.Service
	mqttClient              client.Client
	router                  *gin.Engine
	Hub                     *websocket.Hub
	WSHandler               *websocket.Handler
	DeviceService           service.DeviceService
	ControllerDeviceService service.ControllerDeviceService
	UserService             service.UserService
	IrrigationService       service.IrrigationService
	ContentService          service.ContentService
	NotificationService     service.NotificationService
	DeviceGroupService      service.DeviceGroupService
	SupportService          service.SupportService
	SoilHealthService       service.SoilHealthService
	FirmwareService         service.FirmwareService
	loginAccountLimiter     *middleware.RateLimiter
	resetAccountLimiter     *middleware.RateLimiter
	weather                 *weatherProxy
}

func NewServer(config util.Config, store db.Store, mqttClient client.Client, pushSender push.Sender) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	hub := websocket.NewHub()
	go hub.Run()

	notificationService := service.NewNotificationService(store, hub, pushSender)
	var signingKey *ecdsa.PrivateKey
	if config.FirmwareSigningKeyFile != "" {
		signingKey, err = util.LoadFirmwareSigningKey(config.FirmwareSigningKeyFile)
		if err != nil {
			return nil, fmt.Errorf("cannot load firmware signing key: %w", err)
		}
	} else {
		log.Println("FIRMWARE_SIGNING_KEY_FILE not set: firmware publishing is disabled")
	}
	firmwareService := service.NewFirmwareService(store, mqttClient, hub, notificationService, config.FirmwareStorageDir, signingKey)

	if config.MQTTEnabled {
		sub := subscriber.NewTelemetrySubscriber(mqttClient, store, hub)
		sub.SetOtaEventHandler(firmwareService)

		if err := sub.Start(); err != nil {
			return nil, err
		}
	}

	wsHandler := websocket.NewHandler(hub, strings.Split(config.AllowedOrigins, ","))

	timeoutWorker := worker.NewIrrigationTimeoutWorker(store)
	timeoutWorker.Start()

	worker.NewFirmwareUpdateWorker(firmwareService).Start()

	irrigationService := service.NewIrrigationService(store, mqttClient, notificationService)

	scheduleWorker := worker.NewIrrigationScheduleWorker(irrigationService)
	scheduleWorker.Start()

	server := &Server{
		config:                  config,
		store:                   store,
		tokenMaker:              tokenMaker,
		oauthService:            appoauth.NewService(config),
		mqttClient:              mqttClient,
		Hub:                     hub,
		WSHandler:               wsHandler,
		DeviceService:           service.NewDeviceService(store),
		ControllerDeviceService: service.NewControllerDeviceService(store, mqttClient),
		UserService:             service.NewUserService(store, tokenMaker, config),
		IrrigationService:       irrigationService,
		ContentService:          service.NewContentService(store),
		NotificationService:     notificationService,
		DeviceGroupService:      service.NewDeviceGroupService(store, irrigationService),
		SupportService:          service.NewSupportService(store, hub, notificationService),
		SoilHealthService:       service.NewSoilHealthService(store),
		FirmwareService:         firmwareService,
		loginAccountLimiter:     middleware.Disabled(),
		resetAccountLimiter:     middleware.Disabled(),
		weather:                 newWeatherProxy(config.OpenWeatherAPIKey),
	}
	if config.RateLimitEnabled {
		server.loginAccountLimiter = middleware.NewRateLimiter(time.Minute, 5)
		server.resetAccountLimiter = middleware.NewRateLimiter(20*time.Minute, 3)
	}

	server.router = routes.Setup(server.tokenMaker, server.store, server, routes.Options{
		TrustedProxies:   config.TrustedProxies,
		RateLimitEnabled: config.RateLimitEnabled,
	})

	return server, nil
}

// HTTPHandler exposes the router so tests can serve the real API in-process.
func (server *Server) HTTPHandler() http.Handler {
	return server.router
}

// Start serves with explicit timeouts: gin's Run() has none, which lets slow
// clients hold connections open indefinitely (slowloris). WriteTimeout is
// omitted because the websocket route keeps long-lived connections.
func (server *Server) Start(address string) error {
	httpServer := &http.Server{
		Addr:              address,
		Handler:           server.router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return httpServer.ListenAndServe()
}
