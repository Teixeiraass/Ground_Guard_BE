package routes

import "github.com/gin-gonic/gin"

type FirmwareHandler interface {
	GetFirmwareManifest(c *gin.Context)
	DownloadFirmware(c *gin.Context)
	GetLatestFirmware(c *gin.Context)
	GetControllerFirmware(c *gin.Context)
	UpdateControllerFirmware(c *gin.Context)
	SetControllerAutoUpdate(c *gin.Context)
	PublishFirmware(c *gin.Context)
	ListFirmwareReleases(c *gin.Context)
}

// Controllers have no user token, so they read the manifest and download
// anonymously; the image is verified on the device by its SHA-256.
func registerFirmwareRoutes(router gin.IRoutes, h FirmwareHandler) {
	router.GET("/firmware/manifest", rl.firmwareDownload, h.GetFirmwareManifest)
	router.GET("/firmware/:uuid/download", rl.firmwareDownload, h.DownloadFirmware)
}

func registerAuthFirmwareRoutes(authRoutes gin.IRoutes, h FirmwareHandler) {
	authRoutes.GET("/firmware/latest", h.GetLatestFirmware)
	authRoutes.GET("/controller-devices/:uuid/firmware", h.GetControllerFirmware)
	authRoutes.POST("/controller-devices/:uuid/firmware/update", h.UpdateControllerFirmware)
	authRoutes.PUT("/controller-devices/:uuid/firmware/auto-update", h.SetControllerAutoUpdate)
}

func registerAdminFirmwareRoutes(adminRoutes gin.IRoutes, h FirmwareHandler) {
	adminRoutes.POST("/admin/firmware", h.PublishFirmware)
	adminRoutes.GET("/admin/firmware", h.ListFirmwareReleases)
}
