package routes

import "github.com/gin-gonic/gin"

type DeviceHandler interface {
	CreateDevice(c *gin.Context)
	RegisterDevice(c *gin.Context)
	GetDevice(c *gin.Context)
	GetDeviceByUID(c *gin.Context)
	ListDevice(c *gin.Context)
	LinkDeviceToUserByQrToken(c *gin.Context)
	UnlinkDeviceFromUser(c *gin.Context)
	UpdateNameDevice(c *gin.Context)
	AssignDeviceToGroup(c *gin.Context)
	UpdateDeviceLocation(c *gin.Context)
	ClearDeviceLocation(c *gin.Context)
}

func registerDeviceRoutes(router gin.IRoutes, h DeviceHandler) {
	router.POST("/devices/register", rl.deviceRegister, h.RegisterDevice)
}

func registerAuthDeviceRoutes(authRoutes gin.IRoutes, h DeviceHandler) {
	authRoutes.POST("/devices", h.CreateDevice)
	authRoutes.GET("/devices/:uuid", h.GetDevice)
	authRoutes.GET("/devices", h.ListDevice)
	authRoutes.GET("/devices/uid/:uid", h.GetDeviceByUID)
	authRoutes.PUT("/devices/link/:qr_token", h.LinkDeviceToUserByQrToken)
	authRoutes.PUT("/devices/unlink/:uuid", h.UnlinkDeviceFromUser)
	authRoutes.PUT("/devices/name/:uuid", h.UpdateNameDevice)
	authRoutes.PUT("/devices/:uuid/group", h.AssignDeviceToGroup)
	authRoutes.PUT("/devices/:uuid/location", h.UpdateDeviceLocation)
	authRoutes.DELETE("/devices/:uuid/location", h.ClearDeviceLocation)
}
