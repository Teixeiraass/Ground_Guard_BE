package routes

import "github.com/gin-gonic/gin"

type IrrigationActionHandler interface {
	CreateIrrigationCommand(c *gin.Context)
	UpdateIrrigationCommand(c *gin.Context)
	ListIrrigationHistory(c *gin.Context)
	GetIrrigationCommands(c *gin.Context)
	GetIrrigationHistory(c *gin.Context)
	GetIrrigationStatus(c *gin.Context)
}

func registerIrrigationActionRoutes(authRoutes gin.IRoutes, h IrrigationActionHandler) {
	authRoutes.POST("/irrigation/commands", h.CreateIrrigationCommand)
	authRoutes.PUT("/irrigation/commands/:uuid", h.UpdateIrrigationCommand)
	authRoutes.GET("/irrigation/history", h.ListIrrigationHistory)
	authRoutes.GET("/irrigation/history/:uuid", h.GetIrrigationHistory)
	authRoutes.GET("/irrigation/status", h.GetIrrigationStatus)
	authRoutes.GET("/irrigation/command/:uuid", h.GetIrrigationCommands)
}
