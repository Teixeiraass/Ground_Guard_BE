package routes

import "github.com/gin-gonic/gin"

type IrrigationHandler interface {
	CreateIrrigationPreference(c *gin.Context)
	GetIrrigationPreference(c *gin.Context)
	GetIrrigationPreferenceByDevice(c *gin.Context)
}

func registerIrrigationPreferencesRoutes(authRoutes gin.IRoutes, h IrrigationHandler) {
	authRoutes.POST("/irrigation_preference", h.CreateIrrigationPreference)
	authRoutes.GET("/irrigation_preference/:uuid", h.GetIrrigationPreference)
	authRoutes.GET("/irrigation_preference/device/:uuid", h.GetIrrigationPreferenceByDevice)
}
