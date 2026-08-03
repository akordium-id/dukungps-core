package websocket

import (
	"encoding/json"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

type PositionUpdate struct {
	DeviceID int64              `json:"deviceId"`
	Position *position.Position `json:"position"`
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *PositionUpdate
	done       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *PositionUpdate),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.close()
			}
		case update := <-h.broadcast:
			h.dispatch(update)
		case <-h.done:
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.done)
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(update *PositionUpdate) {
	h.broadcast <- update
}

func (h *Hub) dispatch(update *PositionUpdate) {
	data, err := json.Marshal(update)
	if err != nil {
		return
	}
	for client := range h.clients {
		if !client.safeSend(data) {
			delete(h.clients, client)
			client.close()
		}
	}
}
