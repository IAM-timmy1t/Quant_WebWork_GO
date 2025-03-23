package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
}

// Handler manages WebSocket connections
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler
func NewHandler() *Handler {
	hub := NewHub()
	go hub.Run()
	return &Handler{hub: hub}
}

// ServeHTTP handles WebSocket connection requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	client := NewClient(h.hub, conn)
	client.hub.register <- client

	// Start processing messages
	client.Start()
}

// BroadcastMessage sends a message to all connected clients or specific channel
func (h *Handler) BroadcastMessage(messageType string, data interface{}, channel string) error {
	return h.hub.Broadcast(messageType, data, channel)
}
