package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Time    time.Time   `json:"time"`
	Channel string      `json:"channel"`
}

// Client represents a connected WebSocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	channels map[string]bool
	mu       sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Channel subscriptions
	channels map[string]map[*Client]bool

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
	}
}

// Run starts the hub's main event loop
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
				// Remove from all channels
				for channel := range client.channels {
					if clients, ok := h.channels[channel]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.channels, channel)
						}
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			var msg Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			h.mu.RLock()
			// If channel is specified, send only to subscribers
			if msg.Channel != "" {
				if clients, ok := h.channels[msg.Channel]; ok {
					for client := range clients {
						select {
						case client.send <- message:
						default:
							close(client.send)
							delete(h.clients, client)
							delete(clients, client)
						}
					}
				}
			} else {
				// Broadcast to all clients
				for client := range h.clients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Subscribe adds a client to a channel
func (h *Hub) Subscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channels[channel] == nil {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true

	client.mu.Lock()
	client.channels[channel] = true
	client.mu.Unlock()
}

// Unsubscribe removes a client from a channel
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.channels[channel]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channel)
		}
	}

	client.mu.Lock()
	delete(client.channels, channel)
	client.mu.Unlock()
}

// Broadcast sends a message to all clients or channel subscribers
func (h *Hub) Broadcast(messageType string, data interface{}, channel string) error {
	msg := Message{
		Type:    messageType,
		Data:    data,
		Time:    time.Now(),
		Channel: channel,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- payload
	return nil
}
