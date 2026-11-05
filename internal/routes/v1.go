package routes

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
)

func registerV1Routes(router *gin.Engine, tokenMaker token.Maker, store db.Store, handlers Handlers) {
	v1 := router.Group(V1Prefix)

	registerHelperRoutes(v1, handlers)
	registerUserRoutes(v1, handlers)
	registerDeviceRoutes(v1, handlers)
	registerControllerDeviceRoutes(v1, handlers)
	registerFaqRoutes(v1, handlers)
	registerHelpContentRoutes(v1, handlers)
	registerTutorialRoutes(v1, handlers)
	registerLegalDocumentRoutes(v1, handlers)
	registerPasswordResetRoutes(v1, handlers)
	registerFirmwareRoutes(v1, handlers)

	authRoutes := v1.Group("/")
	authRoutes.Use(authMiddleware(tokenMaker, store))
	registerAuthUserRoutes(authRoutes, handlers)
	registerAuthDeviceRoutes(authRoutes, handlers)
	registerAuthControllerDeviceRoutes(authRoutes, handlers)
	registerIrrigationPreferencesRoutes(authRoutes, handlers)
	registerIrrigationActionRoutes(authRoutes, handlers)
	registerIrrigationScheduleRoutes(authRoutes, handlers)
	registerStatisticsRoutes(authRoutes, handlers)
	registerNotificationRoutes(authRoutes, handlers)
	registerDeviceGroupRoutes(authRoutes, handlers)
	registerAuthLegalDocumentRoutes(authRoutes, handlers)
	registerAuthWsRoutes(authRoutes, handlers)
	registerSupportRoutes(authRoutes, handlers)
	registerAuthFirmwareRoutes(authRoutes, handlers)
	registerWeatherRoutes(authRoutes, handlers)

	adminRoutes := authRoutes.Group("/").Use(middleware.RequireAdmin())
	registerAdminRoutes(adminRoutes, handlers)
	registerAdminSupportRoutes(adminRoutes, handlers)
	registerAdminFirmwareRoutes(adminRoutes, handlers)
	registerAdminControllerDeviceRoutes(adminRoutes, handlers)
}
