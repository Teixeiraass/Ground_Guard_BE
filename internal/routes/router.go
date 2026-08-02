package routes

import (
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(tokenMaker token.Maker, handlers Handlers) *gin.Engine {
	router := gin.Default()

	router.Static("/uploads/profile", "./uploads/profile")
	router.Static("/uploads/qrcodes", "./uploads/qrcodes")

	registerSwaggerRoutes(router)
	registerV1Routes(router, tokenMaker, handlers)

	return router
}

func registerSwaggerRoutes(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
