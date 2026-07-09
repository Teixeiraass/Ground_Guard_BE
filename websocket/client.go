package websocket

import (
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	userID int64
}

func NewClient(conn *websocket.Conn, userID int64) *Client {
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}
}

func (c *Client) ReadPump(hub *Hub) {

	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	for {

		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) WritePump() {

	defer c.conn.Close()

	for {

		msg, ok := <-c.send

		if !ok {
			return
		}

		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println(err)
			return
		}
	}
}