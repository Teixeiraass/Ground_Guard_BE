package routes

import "github.com/gin-gonic/gin"

type PasswordResetHandler interface {
	RequestPasswordReset(c *gin.Context)
	ConfirmPasswordReset(c *gin.Context)
	ResetPasswordPage(c *gin.Context)
}

func registerPasswordResetRoutes(router gin.IRoutes, h PasswordResetHandler) {
	router.POST("/password-reset/request", rl.resetRequest, h.RequestPasswordReset)
	router.POST("/password-reset/confirm", rl.resetConfirm, h.ConfirmPasswordReset)
	router.GET("/reset-password", h.ResetPasswordPage)
}
