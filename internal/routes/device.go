package routes

import "github.com/gin-gonic/gin"

type DeviceHandler interface {
	CreateDevice(c *gin.Context)
	GetDevice(c *gin.Context)
	ListDevice(c *gin.Context)
	LinkDeviceToUserByQrToken(c *gin.Context)
	UnlinkDeviceFromUser(c *gin.Context)
	UpdateNameDevice(c *gin.Context)
}

func registerDeviceRoutes(authRoutes gin.IRoutes, h DeviceHandler) {
	authRoutes.POST("/devices", h.CreateDevice)
	authRoutes.GET("/devices/:uuid", h.GetDevice)
	authRoutes.GET("/devices", h.ListDevice)
	authRoutes.PUT("/devices/link/:qr_token", h.LinkDeviceToUserByQrToken)
	authRoutes.PUT("/devices/unlink/:uuid", h.UnlinkDeviceFromUser)
	authRoutes.PUT("/devices/name/:uuid", h.UpdateNameDevice)
}
