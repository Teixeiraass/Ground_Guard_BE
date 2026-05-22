package handler

import (
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/gin-gonic/gin"
)

func (server *Server) setupRouter() *gin.Engine {
	router := gin.Default()

	router.POST("/users", server.CreateUser)
	router.POST("/users/login", server.LoginUser)
	router.POST("/tokens/refresh", server.RenewAccessToken)

	authRoutes := router.Group("/").Use(middleware.Auth(server.tokenMaker))

	authRoutes.GET("/users/me", server.GetUser)

	authRoutes.POST("/devices", server.CreateDevice)
	authRoutes.GET("/devices/:uuid", server.GetDevice)
	authRoutes.GET("/devices", server.ListDevice)

	return router
}
