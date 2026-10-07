package routes

import "github.com/gin-gonic/gin"

type ControllerDeviceHandler interface {
	CreateControllerDevice(c *gin.Context)
	RegisterControllerDevice(c *gin.Context)
	GetControllerDevice(c *gin.Context)
	GetControllerDeviceByUID(c *gin.Context)
	ClaimControllerDeviceByUID(c *gin.Context)
	ListControllerDevices(c *gin.Context)
	UnlinkControllerDeviceFromUser(c *gin.Context)
	UpdateNameControllerDevice(c *gin.Context)
}

func registerControllerDeviceRoutes(router gin.IRoutes, h ControllerDeviceHandler) {
	router.POST("/controller-devices/register", h.RegisterControllerDevice)
}

func registerAuthControllerDeviceRoutes(authRoutes gin.IRoutes, h ControllerDeviceHandler) {
	authRoutes.POST("/controller-devices", h.CreateControllerDevice)
	authRoutes.GET("/controller-devices", h.ListControllerDevices)
	authRoutes.GET("/controller-devices/:uuid", h.GetControllerDevice)
	authRoutes.GET("/controller-devices/uid/:uid", h.GetControllerDeviceByUID)
	authRoutes.PUT("/controller-devices/claim/:uid", h.ClaimControllerDeviceByUID)
	authRoutes.PUT("/controller-devices/unlink/:uuid", h.UnlinkControllerDeviceFromUser)
	authRoutes.PUT("/controller-devices/name/:uuid", h.UpdateNameControllerDevice)
}
