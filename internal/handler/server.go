package handler

import (
	"fmt"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	appoauth "github.com/Teixeiraass/ground_guard_be/internal/oauth"
	"github.com/Teixeiraass/ground_guard_be/internal/routes"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/internal/worker"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/mqtt/subscriber"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config            util.Config
	store             db.Store
	tokenMaker        token.Maker
	oauthService      *appoauth.Service
	mqttClient        client.Client
	router            *gin.Engine
	Hub               *websocket.Hub
	WSHandler         *websocket.Handler
	DeviceService     service.DeviceService
	UserService       service.UserService
	IrrigationService service.IrrigationService
	ContentService    service.ContentService
}

func NewServer(config util.Config, store db.Store, mqttClient client.Client) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	hub := websocket.NewHub()
	go hub.Run()

	if config.MQTTEnabled {
		sub := subscriber.NewTelemetrySubscriber(mqttClient, store, hub)

		if err := sub.Start(); err != nil {
			return nil, err
		}
	}

	wsHandler := websocket.NewHandler(hub)

	timeoutWorker := worker.NewIrrigationTimeoutWorker(store)
	timeoutWorker.Start()

	server := &Server{
		config:            config,
		store:             store,
		tokenMaker:        tokenMaker,
		oauthService:      appoauth.NewService(config),
		mqttClient:        mqttClient,
		Hub:               hub,
		WSHandler: wsHandler,
		DeviceService:     service.NewDeviceService(store),
		UserService:       service.NewUserService(store, tokenMaker, config),
		IrrigationService: service.NewIrrigationService(store, mqttClient),
		ContentService:    service.NewContentService(store),
	}

	server.router = routes.Setup(server.tokenMaker, server)

	return server, nil
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
