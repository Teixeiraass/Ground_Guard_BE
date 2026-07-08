package routes

import (
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
)

func registerV1Routes(router *gin.Engine, tokenMaker token.Maker, handlers Handlers) {
	v1 := router.Group(V1Prefix)

	registerUserRoutes(v1, handlers)
	registerDeviceRoutes(v1, handlers)
	registerFaqRoutes(v1, handlers)
	registerHelpContentRoutes(v1, handlers)
	registerTutorialRoutes(v1, handlers)
	registerLegalDocumentRoutes(v1, handlers)

	authRoutes := v1.Group("/").Use(authMiddleware(tokenMaker))
	registerAuthUserRoutes(authRoutes, handlers)
	registerAuthDeviceRoutes(authRoutes, handlers)
	registerIrrigationPreferencesRoutes(authRoutes, handlers)
	registerIrrigationActionRoutes(authRoutes, handlers)
}
