package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultMaxBody  int64 = 1 << 20 // 1 MB for JSON APIs
	firmwareMaxBody int64 = 3 << 20 // firmware images are at most ~1.9 MB
	imageMaxBody    int64 = 6 << 20 // profile pictures
)

// SecurityHeaders sets defensive headers on every response.
func SecurityHeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h := ctx.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Cross-Origin-Resource-Policy", "same-site")
		if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		ctx.Next()
	}
}

// LimitBody caps request bodies so a client can't make the server read or
// buffer arbitrarily large payloads.
func LimitBody() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Body == nil {
			ctx.Next()
			return
		}
		limit := defaultMaxBody
		path := ctx.Request.URL.Path
		switch {
		case strings.HasSuffix(path, "/admin/firmware"):
			limit = firmwareMaxBody
		case strings.HasSuffix(path, "/users/profile-image"):
			limit = imageMaxBody
		}
		if ctx.Request.ContentLength > limit {
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, limit)
		ctx.Next()
	}
}
