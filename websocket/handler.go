package websocket

import (
	"net/http"
	"strings"

	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewHandler: allowedOrigins lists browser origins that may connect. Native
// apps send no Origin header and are always accepted (they authenticate with
// the bearer token); this blocks cross-site websocket hijacking from browsers.
func NewHandler(hub *Hub, allowedOrigins []string) *Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o = strings.TrimSpace(o); o != "" {
			allowed[strings.ToLower(o)] = true
		}
	}
	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowed[strings.ToLower(origin)]
			},
		},
	}
}

func (h *Handler) Handle(c *gin.Context) {
	payload := c.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := NewClient(conn, payload.UserID, payload.ExpiredAt)

	h.hub.register <- client

	go client.WritePump()

	client.ReadPump(h.hub)
}
