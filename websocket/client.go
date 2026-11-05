package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	// The server only pushes events; clients send pongs/close frames, so
	// anything larger is abuse.
	maxMessageSize = 512
)

type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	userID    int64
	expiresAt time.Time
}

func NewClient(conn *websocket.Conn, userID int64, expiresAt time.Time) *Client {
	return &Client{
		conn:      conn,
		send:      make(chan []byte, 256),
		userID:    userID,
		expiresAt: expiresAt,
	}
}

func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

// WritePump delivers events, pings to detect dead peers, and closes the
// connection when the access token used to open it expires, so a banned or
// logged-out user stops receiving data.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	expiry := time.NewTimer(time.Until(c.expiresAt))
	defer func() {
		ticker.Stop()
		expiry.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println(err)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-expiry.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "token expired"))
			return
		}
	}
}
