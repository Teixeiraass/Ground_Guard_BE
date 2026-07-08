package routes

import "github.com/gin-gonic/gin"

type FaqHandler interface {
	GetFaq(c *gin.Context)
	ListFaq(c *gin.Context)
}

func registerFaqRoutes(router gin.IRoutes, h FaqHandler) {
	router.GET("/faq/:uuid", h.GetFaq)
	router.GET("/faqs", h.ListFaq)
}
