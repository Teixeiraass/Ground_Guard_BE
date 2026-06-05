package routes

import (
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(tokenMaker token.Maker, handlers Handlers) *gin.Engine {
	router := gin.Default()

	registerUserRoutes(router, handlers)
	registerSwaggerRoutes(router)

	authRoutes := router.Group("/").Use(authMiddleware(tokenMaker))
	registerAuthUserRoutes(authRoutes, handlers)
	registerDeviceRoutes(authRoutes, handlers)

	return router
}

func registerSwaggerRoutes(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
