package routes

import "github.com/gin-gonic/gin"

type WebSocketHandler interface {
    ServeWS(c *gin.Context)
}

func registerAuthWsRoutes(authRoutes gin.IRoutes, h WebSocketHandler) {
    authRoutes.GET("/ws", h.ServeWS)
}