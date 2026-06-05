package routes

import (
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
)

type UserHandler interface {
	CreateUser(c *gin.Context)
	LoginUser(c *gin.Context)
	RenewAccessToken(c *gin.Context)
	GetUser(c *gin.Context)
	UpdateUserName(c *gin.Context)
	UpdateProfileImage(c *gin.Context)
}

func registerUserRoutes(router *gin.Engine, h UserHandler) {
	router.POST("/users", h.CreateUser)
	router.POST("/users/login", h.LoginUser)
	router.POST("/tokens/refresh", h.RenewAccessToken)
}

func registerAuthUserRoutes(authRoutes gin.IRoutes, h UserHandler) {
	authRoutes.GET("/users/me", h.GetUser)
	authRoutes.PUT("/users/name/:uuid", h.UpdateUserName)
	authRoutes.PUT("/users/profile-image", h.UpdateProfileImage)
}

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return middleware.Auth(tokenMaker)
}
