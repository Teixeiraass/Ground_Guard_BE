package routes

import "github.com/gin-gonic/gin"

type NotificationHandler interface {
	ListNotifications(c *gin.Context)
	MarkNotificationRead(c *gin.Context)
	MarkAllNotificationsRead(c *gin.Context)
	ListNotificationPreferences(c *gin.Context)
	UpdateNotificationPreference(c *gin.Context)
	RegisterPushToken(c *gin.Context)
	DeletePushToken(c *gin.Context)
}

func registerNotificationRoutes(authRoutes gin.IRoutes, h NotificationHandler) {
	authRoutes.GET("/notifications", h.ListNotifications)
	authRoutes.PATCH("/notifications/:uuid/read", h.MarkNotificationRead)
	authRoutes.PATCH("/notifications/read-all", h.MarkAllNotificationsRead)
	authRoutes.GET("/notification-preferences", h.ListNotificationPreferences)
	authRoutes.PUT("/notification-preferences/:type", h.UpdateNotificationPreference)
	authRoutes.POST("/push-tokens", h.RegisterPushToken)
	authRoutes.DELETE("/push-tokens/:token", h.DeletePushToken)
}
