package routes

import "github.com/gin-gonic/gin"

type AdminHandler interface {
	ListUsers(c *gin.Context)
	BanUser(c *gin.Context)
	UnbanUser(c *gin.Context)
}

func registerAdminRoutes(adminRoutes gin.IRoutes, h AdminHandler) {
	adminRoutes.GET("/admin/users", h.ListUsers)
	adminRoutes.PUT("/admin/users/:uuid/ban", h.BanUser)
	adminRoutes.PUT("/admin/users/:uuid/unban", h.UnbanUser)
}
