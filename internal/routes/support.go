package routes

import "github.com/gin-gonic/gin"

type SupportHandler interface {
	ListSupportMessages(c *gin.Context)
	SendSupportMessage(c *gin.Context)
	ListSupportConversations(c *gin.Context)
	ListSupportConversationMessages(c *gin.Context)
	ReplySupportConversation(c *gin.Context)
}

func registerSupportRoutes(authRoutes gin.IRoutes, h SupportHandler) {
	authRoutes.GET("/support/messages", h.ListSupportMessages)
	authRoutes.POST("/support/messages", rl.supportSend, h.SendSupportMessage)
}

func registerAdminSupportRoutes(adminRoutes gin.IRoutes, h SupportHandler) {
	adminRoutes.GET("/admin/support/conversations", h.ListSupportConversations)
	adminRoutes.GET("/admin/support/conversations/:uuid/messages", h.ListSupportConversationMessages)
	adminRoutes.POST("/admin/support/conversations/:uuid/messages", h.ReplySupportConversation)
}
