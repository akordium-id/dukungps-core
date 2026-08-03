package handler

import (
	fiberwebsocket "github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"

	"github.com/akordium-id/dukungps/internal/infrastructure/websocket"
)

type WebSocketHandler struct {
	hub *websocket.Hub
}

func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

func (h *WebSocketHandler) Handle(c fiber.Ctx) error {
	return fiberwebsocket.New(func(conn *fiberwebsocket.Conn) {
		client := websocket.NewClient(h.hub)
		h.hub.Register(client)
		defer h.hub.Unregister(client)

		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				_ = msg
			}
		}()

		for {
			select {
			case data, ok := <-client.SendChannel():
				if !ok {
					return
				}
				if err := conn.WriteMessage(fiberwebsocket.TextMessage, data); err != nil {
					return
				}
			}
		}
	})(c)
}
