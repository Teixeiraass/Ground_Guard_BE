package routes

import "github.com/gin-gonic/gin"

type IrrigationScheduleHandler interface {
	CreateIrrigationSchedule(c *gin.Context)
	ListIrrigationSchedules(c *gin.Context)
	GetIrrigationSchedule(c *gin.Context)
	UpdateIrrigationSchedule(c *gin.Context)
	DeleteIrrigationSchedule(c *gin.Context)
}

func registerIrrigationScheduleRoutes(authRoutes gin.IRoutes, h IrrigationScheduleHandler) {
	authRoutes.POST("/irrigation/schedules", h.CreateIrrigationSchedule)
	authRoutes.GET("/irrigation/schedules", h.ListIrrigationSchedules)
	authRoutes.GET("/irrigation/schedules/:uuid", h.GetIrrigationSchedule)
	authRoutes.PUT("/irrigation/schedules/:uuid", h.UpdateIrrigationSchedule)
	authRoutes.DELETE("/irrigation/schedules/:uuid", h.DeleteIrrigationSchedule)
}
