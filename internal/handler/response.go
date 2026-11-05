package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

// serverError logs the real cause and returns a generic message, so database
// and driver errors (table names, constraint names, SQL) never reach clients.
func serverError(ctx *gin.Context, err error) {
	log.Printf("internal error on %s %s: %v", ctx.Request.Method, ctx.FullPath(), err)
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
