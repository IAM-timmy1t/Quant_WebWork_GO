package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/timot/Quant_WebWork_GO/internal/dashboard"
	"compress/flate"
)

// DashboardHub manages WebSocket connections for the dashboard
type DashboardHub struct {
	mu sync.RWMutex

	// Dependencies
	service *dashboard.Service

	// Client management
	clients    map[*DashboardClient]bool
	broadcast  chan interface{}
	register   chan *DashboardClient
	unregister chan *DashboardClient

	// Control
	stopChan chan struct{}
}

// DashboardClient represents a connected dashboard client
type DashboardClient struct {
	hub  *DashboardHub
	conn *websocket.Conn

	// Client configuration
	config dashboard.ClientConfig

	// Message batching
	batchMu    sync.Mutex
	batchTimer *time.Timer
	batch      []interface{}

	// Message handling
	send     chan []byte
	compress *CompressionHandler
}

// CompressionHandler handles WebSocket message compression
type CompressionHandler struct {
	writer *websocket.Conn
	level  int
}

// NewCompressionHandler creates a new compression handler
func NewCompressionHandler(conn *websocket.Conn, level int) *CompressionHandler {
	return &CompressionHandler{
		writer: conn,
		level:  level,
	}
}

// Compress compresses data using the specified compression level
func (h *CompressionHandler) Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := flate.NewWriter(&b, h.level)
	if err != nil {
		return nil, fmt.Errorf("failed to create compression writer: %v", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write compressed data: %v", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close compression writer: %v", err)
	}

	return b.Bytes(), nil
}

// sendBatch sends a batch of messages to the client
func (c *DashboardClient) sendBatch() {
	c.batchMu.Lock()
	defer c.batchMu.Unlock()

	if len(c.batch) == 0 {
		return
	}

	// Create message batch
	msg := &dashboard.Message{
		Type:    dashboard.MessageTypeMetrics,
		Payload: c.batch,
		Time:    time.Now(),
	}

	// Encode message
	data, err := json.Marshal(msg)
	if err != nil {
		// Log error and skip batch
		fmt.Printf("Error encoding batch: %v\n", err)
		c.batch = nil
		return
	}

	// Compress if enabled
	if c.config.Compressed {
		compressed, err := c.compress.Compress(data)
		if err != nil {
			fmt.Printf("Error compressing batch: %v\n", err)
		} else {
			data = compressed
		}
	}

	// Send to client
	select {
	case c.send <- data:
		// Clear batch after successful send
		c.batch = nil
	default:
		// Channel full, close connection
		c.hub.unregister <- c
	}
}

// queueMessage adds a message to the batch queue
func (c *DashboardClient) queueMessage(msg interface{}) {
	c.batchMu.Lock()
	defer c.batchMu.Unlock()

	c.batch = append(c.batch, msg)

	// Send immediately if batch size reached
	if len(c.batch) >= c.config.BatchSize {
		c.sendBatch()
		return
	}

	// Reset batch timer
	if c.batchTimer != nil {
		c.batchTimer.Stop()
	}
	c.batchTimer = time.AfterFunc(c.config.BatchDelay, c.sendBatch)
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *DashboardClient) writePump() {
	ticker := time.NewTicker(time.Second * 54)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// NewDashboardHub creates a new dashboard hub
func NewDashboardHub(service *dashboard.Service) *DashboardHub {
	return &DashboardHub{
		service:    service,
		clients:    make(map[*DashboardClient]bool),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *DashboardClient),
		unregister: make(chan *DashboardClient),
		stopChan:   make(chan struct{}),
	}
}

// Start begins processing messages and events
func (h *DashboardHub) Start(ctx context.Context) {
	go h.run(ctx)
}

// Stop stops the hub and closes all client connections
func (h *DashboardHub) Stop() {
	close(h.stopChan)
}

// run processes messages and events
func (h *DashboardHub) run(ctx context.Context) {
	// Subscribe to dashboard service updates
	updateChan := h.service.Subscribe(256)
	defer h.service.Unsubscribe(updateChan)

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			go h.sendInitialState(ctx, client)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case msg := <-updateChan:
			h.broadcastMessage(msg)
		}
	}
}

// sendInitialState sends initial dashboard state to a new client
func (h *DashboardHub) sendInitialState(ctx context.Context, client *DashboardClient) {
	// Get current metrics
	metrics, err := h.service.GetLatestMetrics(ctx)
	if err != nil {
		fmt.Printf("Error getting initial metrics: %v\n", err)
		return
	}

	// Get security status
	security, err := h.service.GetSecurityStatus(ctx)
	if err != nil {
		fmt.Printf("Error getting initial security status: %v\n", err)
		return
	}

	// Send initial state
	initialState := map[string]interface{}{
		"metrics":  metrics,
		"security": security,
	}

	client.queueMessage(&dashboard.Message{
		Type:    dashboard.MessageTypeConfiguration,
		Payload: initialState,
		Time:    time.Now(),
	})
}

// broadcastMessage sends a message to all clients
func (h *DashboardHub) broadcastMessage(msg dashboard.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Apply client filters
		switch msg.Type {
		case dashboard.MessageTypeMetrics:
			if metrics, ok := msg.Payload.(map[string]interface{}); ok {
				if !client.config.Filters.FilterMetrics(metrics) {
					continue
				}
			}
		case dashboard.MessageTypeSecurityEvent:
			if event, ok := msg.Payload.(map[string]interface{}); ok {
				if !client.config.Filters.FilterSecurityEvent(event) {
					continue
				}
			}
		}

		client.queueMessage(msg)
	}
}

// ServeWS handles WebSocket connections from clients
func (h *DashboardHub) ServeWS(ctx context.Context, conn *websocket.Conn) {
	client := &DashboardClient{
		hub:  h,
		conn: conn,
		config: dashboard.ClientConfig{
			BatchSize:  10,
			BatchDelay: time.Second,
		},
		send:     make(chan []byte, 256),
		compress: NewCompressionHandler(conn, flate.DefaultCompression),
	}

	// Register client
	h.register <- client

	// Start client message pumps
	go client.writePump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *DashboardClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.conn.SetReadDeadline(time.Now().Add(time.Minute))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(time.Minute))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket read error: %v\n", err)
			}
			break
		}

		// Handle client configuration updates
		var config dashboard.ClientConfig
		if err := json.Unmarshal(message, &config); err == nil {
			c.config = config
		}
	}
}
