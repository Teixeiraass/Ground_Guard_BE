package routes

import "github.com/gin-gonic/gin"

type TutorialHandler interface {
	GetTutorial(c *gin.Context)
	ListTutorials(c *gin.Context)
}

func registerTutorialRoutes(router gin.IRoutes, h TutorialHandler) {
	router.GET("/tutorial/:uuid", h.GetTutorial)
	router.GET("/tutorials", h.ListTutorials)
}
