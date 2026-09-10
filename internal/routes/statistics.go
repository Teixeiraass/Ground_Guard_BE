package routes

import "github.com/gin-gonic/gin"

type StatisticsHandler interface {
	GetStatistics(c *gin.Context)
}

func registerStatisticsRoutes(authRoutes gin.IRoutes, h StatisticsHandler) {
	authRoutes.GET("/statistics", h.GetStatistics)
}
