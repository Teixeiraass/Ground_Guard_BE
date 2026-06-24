package routes

import "github.com/gin-gonic/gin"

type HelpContentHandler interface {
	GetHelpContent(c *gin.Context)
	ListHelpContent(c *gin.Context)
}

func registerHelpContentRoutes(router gin.IRoutes, h HelpContentHandler) {
	router.GET("/help_content/:uuid", h.GetHelpContent)
	router.GET("/help_contents", h.ListHelpContent)
}