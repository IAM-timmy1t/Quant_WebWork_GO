package dashboard

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

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

// WebSocketHub maintains the set of active WebSocket clients
type WebSocketHub struct {
	// Registered clients
	clients map[*WebSocketClient]bool

	// Register requests from the clients
	register chan *WebSocketClient

	// Unregister requests from clients
	unregister chan *WebSocketClient

	// Broadcast message to all clients
	broadcast chan interface{}

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// WebSocketClient represents a WebSocket connection client
type WebSocketClient struct {
	hub *WebSocketHub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan interface{}

	// Message filters
	filters map[string]bool

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan interface{}, 256),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
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
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastEvent sends an event to all connected clients
func (h *WebSocketHub) BroadcastEvent(eventType string, payload interface{}) {
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    eventType,
		Payload: payload,
	}
	h.broadcast <- event
}

// NewClient creates a new WebSocket client
func (h *WebSocketHub) NewClient(conn *websocket.Conn) *WebSocketClient {
	client := &WebSocketClient{
		hub:     h,
		conn:    conn,
		send:    make(chan interface{}, 256),
		filters: make(map[string]bool),
	}

	h.register <- client
	return client
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error if needed
			}
			break
		}

		// Handle client messages
		var msg struct {
			Action  string   `json:"action"`
			Filters []string `json:"filters,omitempty"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Action {
		case "set_filters":
			c.setFilters(msg.Filters)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// setFilters updates the client's message filters
func (c *WebSocketClient) setFilters(filters []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.filters = make(map[string]bool)
	for _, filter := range filters {
		c.filters[filter] = true
	}
}
