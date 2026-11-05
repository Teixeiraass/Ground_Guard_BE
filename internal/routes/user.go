package routes

import (
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
)

type UserHandler interface {
	CreateUser(c *gin.Context)
	LoginUser(c *gin.Context)
	RenewAccessToken(c *gin.Context)
	GetUser(c *gin.Context)
	OAuthLogin(c *gin.Context)
	UpdateUserName(c *gin.Context)
	UpdateProfileImage(c *gin.Context)
	LogoutUser(c *gin.Context)
	ChangePassword(c *gin.Context)
	DeleteAccount(c *gin.Context)
	VerifyEmail(c *gin.Context)
	VerifyEmailPage(c *gin.Context)
	ResendVerificationEmail(c *gin.Context)
}

func registerUserRoutes(router gin.IRoutes, h UserHandler) {
	router.POST("/users", rl.register, h.CreateUser)
	router.POST("/users/login", rl.login, h.LoginUser)
	router.POST("/users/oauth/:provider", rl.oauth, h.OAuthLogin)
	router.POST("/tokens/refresh", rl.refresh, h.RenewAccessToken)
	router.POST("/users/verify-email", rl.verifyEmail, h.VerifyEmail)
	router.GET("/verify-email", h.VerifyEmailPage)
}

func registerAuthUserRoutes(authRoutes gin.IRoutes, h UserHandler) {
	authRoutes.GET("/users/me", h.GetUser)
	authRoutes.PUT("/users/name", h.UpdateUserName)
	authRoutes.PUT("/users/profile-image", h.UpdateProfileImage)
	authRoutes.PUT("/users/password", h.ChangePassword)
	authRoutes.DELETE("/users/me", h.DeleteAccount)
	authRoutes.POST("/users/verify-email/resend", rl.verifyResend, h.ResendVerificationEmail)
	authRoutes.POST("/users/logout", h.LogoutUser)
}

func authMiddleware(tokenMaker token.Maker, store db.Store) gin.HandlerFunc {
	return middleware.Auth(tokenMaker, store)
}
