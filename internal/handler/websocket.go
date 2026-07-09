package handler

import "github.com/gin-gonic/gin"

func (server *Server) ServeWS(c *gin.Context) {
	server.WSHandler.Handle(c)
}