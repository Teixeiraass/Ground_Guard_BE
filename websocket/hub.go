package websocket

import (
	"encoding/json"
	"sync"
)

type Hub struct {
	clients map[*Client]bool

	register chan *Client

	unregister chan *Client

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {

		case client := <-h.register:

			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:

			h.mu.Lock()

			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

			h.mu.Unlock()
		}
	}
}

func (h *Hub) Broadcast(v any) {

	data, err := json.Marshal(v)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {

		select {

		case client.send <- data:

		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) BroadcastToUser(userID int64, payload any) {
	data, _ := json.Marshal(payload)

	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}