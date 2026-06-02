package handler

import (
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (server *Server) setupRouter() *gin.Engine {
	router := gin.Default()

	router.POST("/users", server.CreateUser)
	router.POST("/users/login", server.LoginUser)
	router.POST("/tokens/refresh", server.RenewAccessToken)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authRoutes := router.Group("/").Use(middleware.Auth(server.tokenMaker))

	authRoutes.GET("/users/me", server.GetUser)

	authRoutes.POST("/devices", server.CreateDevice)
	authRoutes.GET("/devices/:uuid", server.GetDevice)
	authRoutes.GET("/devices", server.ListDevice)
	authRoutes.PUT("/devices/link/:qr_token", server.LinkDeviceToUserByQrToken)
	authRoutes.PUT("/devices/unlink/:uuid", server.UnlinkDeviceFromUser)
	authRoutes.PUT("/devices/name/:uuid", server.UpdateNameDevice)

	return router
}
