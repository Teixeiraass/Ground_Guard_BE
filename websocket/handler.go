package websocket

import (
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

var upgrader = websocket.Upgrader{

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) Handle(c *gin.Context) {
	payload := c.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := NewClient(conn, payload.UserID)

	h.hub.register <- client

	go client.WritePump()

	client.ReadPump(h.hub)
}