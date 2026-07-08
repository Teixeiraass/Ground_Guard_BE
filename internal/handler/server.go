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
	"github.com/gin-gonic/gin"
)

type Server struct {
	config            util.Config
	store             db.Store
	tokenMaker        token.Maker
	oauthService      *appoauth.Service
	mqttClient        client.Client
	router            *gin.Engine
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

	if config.MQTTEnabled {
		sub := subscriber.NewTelemetrySubscriber(mqttClient, store)
		if err := sub.Start(); err != nil {
			return nil, fmt.Errorf("cannot start mqtt telemetry subscriber: %w", err)
		}
	}

	timeoutWorker := worker.NewIrrigationTimeoutWorker(store)
	timeoutWorker.Start()

	server := &Server{
		config:            config,
		store:             store,
		tokenMaker:        tokenMaker,
		oauthService:      appoauth.NewService(config),
		mqttClient:        mqttClient,
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
