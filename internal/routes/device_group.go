package routes

import "github.com/gin-gonic/gin"

type DeviceGroupHandler interface {
	CreateDeviceGroup(c *gin.Context)
	ListDeviceGroups(c *gin.Context)
	GetDeviceGroup(c *gin.Context)
	UpdateDeviceGroupName(c *gin.Context)
	DeleteDeviceGroup(c *gin.Context)
	TriggerGroupIrrigationCommand(c *gin.Context)
	CreateGroupSchedule(c *gin.Context)
	GetGroupStatistics(c *gin.Context)
}

func registerDeviceGroupRoutes(authRoutes gin.IRoutes, h DeviceGroupHandler) {
	authRoutes.POST("/device-groups", h.CreateDeviceGroup)
	authRoutes.GET("/device-groups", h.ListDeviceGroups)
	authRoutes.GET("/device-groups/:uuid", h.GetDeviceGroup)
	authRoutes.PUT("/device-groups/:uuid", h.UpdateDeviceGroupName)
	authRoutes.DELETE("/device-groups/:uuid", h.DeleteDeviceGroup)
	authRoutes.POST("/device-groups/:uuid/irrigation-commands", h.TriggerGroupIrrigationCommand)
	authRoutes.POST("/device-groups/:uuid/schedules", h.CreateGroupSchedule)
	authRoutes.GET("/device-groups/:uuid/statistics", h.GetGroupStatistics)
}
