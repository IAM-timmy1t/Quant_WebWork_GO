package dashboard

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Message types
	TypeSystemMetrics  MessageType = "system_metrics"
	TypeServiceMetrics MessageType = "service_metrics"
	TypeRouteMetrics   MessageType = "route_metrics"
	TypeLogs          MessageType = "logs"
	TypeError         MessageType = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType  `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client represents a connected WebSocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan Message
	filters  map[MessageType]bool
	mu       sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients
	broadcast chan Message

	// Metrics collector reference
	metrics *MetricsCollector

	// Synchronization
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub(metrics *MetricsCollector) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
		metrics:    metrics,
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	// Start periodic metrics broadcast
	go h.broadcastMetrics()

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

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if client.shouldReceiveMessage(message.Type) {
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

// broadcastMetrics periodically broadcasts system metrics
func (h *Hub) broadcastMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Broadcast system metrics
		metrics := h.metrics.GetMetrics()
		h.broadcast <- Message{
			Type:    TypeSystemMetrics,
			Payload: metrics,
		}
	}
}

// BroadcastServiceMetrics broadcasts service metrics to all clients
func (h *Hub) BroadcastServiceMetrics(serviceID string) {
	metrics := h.metrics.GetServiceMetrics(serviceID)
	if metrics != nil {
		h.broadcast <- Message{
			Type:    TypeServiceMetrics,
			Payload: metrics,
		}
	}
}

// BroadcastLogs broadcasts new log entries to all clients
func (h *Hub) BroadcastLogs(logs []LogEntry) {
	h.broadcast <- Message{
		Type:    TypeLogs,
		Payload: logs,
	}
}

// NewClient creates a new client connection
func (h *Hub) NewClient(conn *websocket.Conn) *Client {
	client := &Client{
		hub:     h,
		conn:    conn,
		send:    make(chan Message, 256),
		filters: make(map[MessageType]bool),
	}

	// Register the client
	h.register <- client

	// Start client message pumps
	go client.writePump()
	go client.readPump()

	return client
}

// Client methods

// shouldReceiveMessage checks if the client should receive a message of the given type
func (c *Client) shouldReceiveMessage(msgType MessageType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.filters) == 0 {
		return true
	}
	return c.filters[msgType]
}

// SetFilter sets message type filters for the client
func (c *Client) SetFilter(msgTypes []MessageType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.filters = make(map[MessageType]bool)
	for _, msgType := range msgTypes {
		c.filters[msgType] = true
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512) // Set maximum message size

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error if needed
			}
			break
		}

		// Handle client messages (e.g., filter updates)
		var msg struct {
			Action  string       `json:"action"`
			Filters []MessageType `json:"filters,omitempty"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Action {
		case "set_filters":
			c.SetFilter(msg.Filters)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(60 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Marshal and write the message
			data, err := json.Marshal(message)
			if err != nil {
				return
			}
			w.Write(data)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping message
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
