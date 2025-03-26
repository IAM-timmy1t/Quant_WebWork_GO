// websocket_adapter.go - WebSocket adapter implementation for bridge system

package adapters

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
	"github.com/gorilla/websocket"
)

// Common WebSocket errors
var (
	ErrWebSocketNotConnected     = errors.New("websocket not connected")
	ErrWebSocketAlreadyConnected = errors.New("websocket already connected")
	ErrMessageHandlerNotSet      = errors.New("message handler not set")
	ErrConnectionClosed          = errors.New("connection closed")
)

// WebSocketAdapterConfig contains configuration for the WebSocket adapter
type WebSocketAdapterConfig struct {
	URL               string
	Headers           map[string]string
	HandshakeTimeout  time.Duration
	PingInterval      time.Duration
	PongTimeout       time.Duration
	WriteTimeout      time.Duration
	ReadTimeout       time.Duration
	MessageBufferSize int
	ReconnectStrategy ReconnectStrategy
}

// ReconnectStrategy defines how reconnection attempts should be handled
type ReconnectStrategy struct {
	MaxAttempts       int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	DelayMultiplier   float64
	JitterFactor      float64
	ResetAfterSuccess time.Duration
}

// DefaultWebSocketAdapterConfig returns the default configuration
func DefaultWebSocketAdapterConfig() *WebSocketAdapterConfig {
	return &WebSocketAdapterConfig{
		Headers:           make(map[string]string),
		HandshakeTimeout:  10 * time.Second,
		PingInterval:      30 * time.Second,
		PongTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadTimeout:       10 * time.Second,
		MessageBufferSize: 100,
		ReconnectStrategy: ReconnectStrategy{
			MaxAttempts:       10,
			InitialDelay:      time.Second,
			MaxDelay:          60 * time.Second,
			DelayMultiplier:   1.5,
			JitterFactor:      0.2,
			ResetAfterSuccess: 60 * time.Second,
		},
	}
}

// WebSocketAdapter implements a WebSocket communication adapter
type WebSocketAdapter struct {
	name             string
	config           *WebSocketAdapterConfig
	conn             *websocket.Conn
	connMutex        sync.Mutex
	isConnected      bool
	reconnectAttempt int
	lastConnectTime  time.Time
	reconnectTimer   *time.Timer
	messageHandler   MessageHandler
	sendChan         chan []byte
	receiveChan      chan []byte
	errChan          chan error
	stopChan         chan struct{}
	metrics          *metrics.Collector
	logger           AdapterLogger
	pendingMessages  [][]byte
	pendingMutex     sync.Mutex
	initialized      bool
	ctx              context.Context
	cancel           context.CancelFunc
}

// MessageHandler handles messages received from the WebSocket
type MessageHandler func([]byte) error

// NewWebSocketAdapter creates a new WebSocket adapter
func NewWebSocketAdapter(name string, config *WebSocketAdapterConfig, metrics *metrics.Collector, logger AdapterLogger) (*WebSocketAdapter, error) {
	if config == nil {
		config = DefaultWebSocketAdapterConfig()
	}

	// Validate configuration
	if config.URL == "" {
		return nil, fmt.Errorf("WebSocket URL cannot be empty")
	}

	// Parse URL to validate
	_, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	adapter := &WebSocketAdapter{
		name:             name,
		config:           config,
		isConnected:      false,
		reconnectAttempt: 0,
		sendChan:         make(chan []byte, config.MessageBufferSize),
		receiveChan:      make(chan []byte, config.MessageBufferSize),
		errChan:          make(chan error, 10),
		stopChan:         make(chan struct{}),
		metrics:          metrics,
		logger:           logger,
		pendingMessages:  make([][]byte, 0),
	}

	return adapter, nil
}

// Initialize initializes the adapter
func (a *WebSocketAdapter) Initialize(ctx context.Context) error {
	a.logger.Info(fmt.Sprintf("Initializing WebSocket adapter '%s'", a.name), nil)

	// Create cancelable context
	a.ctx, a.cancel = context.WithCancel(ctx)

	a.initialized = true
	return nil
}

// Connect establishes a WebSocket connection
func (a *WebSocketAdapter) Connect(ctx context.Context) error {
	a.connMutex.Lock()
	defer a.connMutex.Unlock()

	if a.isConnected {
		return ErrWebSocketAlreadyConnected
	}

	// Create cancelable context for connection
	connCtx, cancel := context.WithTimeout(ctx, a.config.HandshakeTimeout)
	defer cancel()

	// Prepare headers
	header := http.Header{}
	for key, value := range a.config.Headers {
		header.Set(key, value)
	}

	// Establish connection
	a.logger.Debug(fmt.Sprintf("Connecting to WebSocket: %s", a.config.URL), nil)
	conn, _, err := websocket.DefaultDialer.DialContext(connCtx, a.config.URL, header)
	if err != nil {
		a.recordReconnectMetrics(false)
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	a.conn = conn
	a.isConnected = true
	a.lastConnectTime = time.Now()
	a.reconnectAttempt = 0

	// Setup ping handler
	a.conn.SetPongHandler(func(string) error {
		a.conn.SetReadDeadline(time.Now().Add(a.config.PongTimeout))
		return nil
	})

	// Start handler goroutines
	go a.readPump()
	go a.writePump()
	go a.pingPump()

	// Send any pending messages
	a.pendingMutex.Lock()
	if len(a.pendingMessages) > 0 {
		a.logger.Info(fmt.Sprintf("Sending %d pending messages", len(a.pendingMessages)), nil)
		for _, msg := range a.pendingMessages {
			select {
			case a.sendChan <- msg:
				// Message queued for sending
			default:
				a.logger.Warn("Send channel full, dropping pending message", nil)
			}
		}
		a.pendingMessages = make([][]byte, 0)
	}
	a.pendingMutex.Unlock()

	a.recordReconnectMetrics(true)
	a.logger.Info(fmt.Sprintf("Connected to WebSocket: %s", a.config.URL), nil)
	return nil
}

// Send sends data through the WebSocket
func (a *WebSocketAdapter) Send(ctx context.Context, data []byte) ([]byte, error) {
	if !a.initialized {
		return nil, fmt.Errorf("adapter not initialized")
	}

	if !a.isConnected {
		// Store message for later sending
		a.pendingMutex.Lock()
		a.pendingMessages = append(a.pendingMessages, data)
		a.pendingMutex.Unlock()

		// Try to reconnect
		go a.tryReconnect()

		return nil, ErrWebSocketNotConnected
	}

	// Send the message
	select {
	case a.sendChan <- data:
		// Message queued for sending
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("send channel full")
	}

	// WebSocket is asynchronous, so we don't wait for a response here
	// If a reply is needed, it should come via the message handler
	return nil, nil
}

// Receive synchronously waits for a message from the WebSocket
func (a *WebSocketAdapter) Receive(ctx context.Context) ([]byte, error) {
	if !a.initialized {
		return nil, fmt.Errorf("adapter not initialized")
	}

	if !a.isConnected {
		return nil, ErrWebSocketNotConnected
	}

	select {
	case msg := <-a.receiveChan:
		return msg, nil
	case err := <-a.errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SetMessageHandler sets the handler for received messages
func (a *WebSocketAdapter) SetMessageHandler(handler MessageHandler) {
	a.messageHandler = handler
}

// readPump handles reading messages from the WebSocket
func (a *WebSocketAdapter) readPump() {
	defer func() {
		a.closeConnection("read pump ending")
	}()

	for {
		if a.conn == nil {
			return
		}

		// Set read deadline
		a.conn.SetReadDeadline(time.Now().Add(a.config.ReadTimeout))

		// Read message
		_, message, err := a.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				a.logger.Error(fmt.Sprintf("Unexpected WebSocket close: %v", err), nil)
			}
			a.errChan <- fmt.Errorf("error reading from WebSocket: %w", err)
			return
		}

		// Handle message
		if a.messageHandler != nil {
			if err := a.messageHandler(message); err != nil {
				a.logger.Warn(fmt.Sprintf("Error handling WebSocket message: %v", err), nil)
			}
		}

		// Also send to receive channel
		select {
		case a.receiveChan <- message:
			// Message sent to channel
		default:
			// Channel full, discard oldest message
			select {
			case <-a.receiveChan:
				a.receiveChan <- message
			default:
				// If we can't discard, simply drop the message
				a.logger.Warn("Receive channel full, dropping message", nil)
			}
		}

		// Record metrics
		if a.metrics != nil {
			a.recordMessageMetrics("received", int64(len(message)))
		}
	}
}

// writePump handles writing messages to the WebSocket
func (a *WebSocketAdapter) writePump() {
	defer func() {
		a.closeConnection("write pump ending")
	}()

	for {
		select {
		case message := <-a.sendChan:
			if a.conn == nil {
				return
			}

			// Set write deadline
			a.conn.SetWriteDeadline(time.Now().Add(a.config.WriteTimeout))

			err := a.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				a.logger.Error(fmt.Sprintf("Error writing to WebSocket: %v", err), nil)
				a.errChan <- fmt.Errorf("error writing to WebSocket: %w", err)
				return
			}

			// Record metrics
			if a.metrics != nil {
				a.recordMessageMetrics("sent", int64(len(message)))
			}

		case <-a.stopChan:
			return
		}
	}
}

// pingPump handles sending ping messages to keep the connection alive
func (a *WebSocketAdapter) pingPump() {
	ticker := time.NewTicker(a.config.PingInterval)
	defer func() {
		ticker.Stop()
		a.closeConnection("ping pump ending")
	}()

	for {
		select {
		case <-ticker.C:
			if a.conn == nil {
				return
			}

			a.conn.SetWriteDeadline(time.Now().Add(a.config.WriteTimeout))
			if err := a.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				a.logger.Error(fmt.Sprintf("Error sending ping: %v", err), nil)
				return
			}

		case <-a.stopChan:
			return
		}
	}
}

// closeConnection closes the WebSocket connection
func (a *WebSocketAdapter) closeConnection(reason string) {
	a.connMutex.Lock()
	defer a.connMutex.Unlock()

	if !a.isConnected {
		return
	}

	a.logger.Info(fmt.Sprintf("Closing WebSocket connection: %s", reason), nil)

	// Close the connection
	if a.conn != nil {
		// Set deadline for close message
		a.conn.SetWriteDeadline(time.Now().Add(time.Second))

		// Try to send close message
		err := a.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason))
		if err != nil {
			a.logger.Warn(fmt.Sprintf("Error sending close message: %v", err), nil)
		}

		// Close the connection
		if err := a.conn.Close(); err != nil {
			a.logger.Warn(fmt.Sprintf("Error closing WebSocket: %v", err), nil)
		}
	}

	a.isConnected = false
	a.conn = nil

	// Schedule reconnection if not explicitly stopped
	select {
	case <-a.stopChan:
		// Adapter is being shut down, don't reconnect
	default:
		go a.tryReconnect()
	}
}

// tryReconnect attempts to reconnect to the WebSocket
func (a *WebSocketAdapter) tryReconnect() {
	a.connMutex.Lock()

	// Check if already connected or reconnecting
	if a.isConnected || a.reconnectTimer != nil {
		a.connMutex.Unlock()
		return
	}

	// Increment reconnect attempt counter
	a.reconnectAttempt++

	// Check max attempts
	if a.config.ReconnectStrategy.MaxAttempts > 0 &&
		a.reconnectAttempt > a.config.ReconnectStrategy.MaxAttempts {
		a.logger.Error(fmt.Sprintf("Max reconnect attempts (%d) reached",
			a.config.ReconnectStrategy.MaxAttempts), nil)
		a.connMutex.Unlock()
		return
	}

	// Calculate backoff delay
	delay := a.calculateBackoff()

	a.logger.Info(fmt.Sprintf("Scheduling reconnect attempt %d in %v",
		a.reconnectAttempt, delay), nil)

	// Schedule reconnection
	a.reconnectTimer = time.AfterFunc(delay, func() {
		a.connMutex.Lock()
		a.reconnectTimer = nil
		a.connMutex.Unlock()

		// Attempt reconnection
		ctx, cancel := context.WithTimeout(a.ctx, a.config.HandshakeTimeout)
		defer cancel()

		if err := a.Connect(ctx); err != nil {
			a.logger.Error(fmt.Sprintf("Reconnect attempt %d failed: %v",
				a.reconnectAttempt, err), nil)
		}
	})

	a.connMutex.Unlock()
}

// calculateBackoff calculates the backoff delay for reconnection
func (a *WebSocketAdapter) calculateBackoff() time.Duration {
	strategy := a.config.ReconnectStrategy

	// Base delay with exponential backoff
	delayMs := float64(strategy.InitialDelay.Milliseconds()) *
		pow(strategy.DelayMultiplier, float64(a.reconnectAttempt-1))

	// Cap at max delay
	if maxMs := float64(strategy.MaxDelay.Milliseconds()); delayMs > maxMs {
		delayMs = maxMs
	}

	// Add jitter
	if strategy.JitterFactor > 0 {
		jitter := (rand()*2 - 1) * strategy.JitterFactor * delayMs
		delayMs += jitter
	}

	return time.Duration(delayMs) * time.Millisecond
}

// rand returns a random number between 0 and 1
func rand() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// pow calculates x^y
func pow(x, y float64) float64 {
	if y == 0 {
		return 1
	}
	if y == 1 {
		return x
	}

	result := x
	for i := 1; i < int(y); i++ {
		result *= x
	}
	return result
}

// recordMessageMetrics records metrics for messages
func (a *WebSocketAdapter) recordMessageMetrics(direction string, size int64) {
	if a.metrics == nil {
		return
	}

	tags := map[string]string{
		"adapter":   a.name,
		"direction": direction,
	}

	a.metrics.Collect("websocket", "message_size", float64(size), tags)
}

// recordReconnectMetrics records metrics for reconnection attempts
func (a *WebSocketAdapter) recordReconnectMetrics(success bool) {
	if a.metrics == nil {
		return
	}

	tags := map[string]string{
		"adapter": a.name,
		"attempt": fmt.Sprintf("%d", a.reconnectAttempt),
	}

	eventType := "reconnect_failure"
	if success {
		eventType = "reconnect_success"
	}

	a.metrics.Collect("websocket", eventType, 1.0, tags)
}

// Close closes the adapter
func (a *WebSocketAdapter) Close() error {
	if !a.initialized {
		return nil
	}

	// Signal all goroutines to stop
	close(a.stopChan)

	// Cancel context
	if a.cancel != nil {
		a.cancel()
	}

	// Close WebSocket connection
	a.connMutex.Lock()
	if a.conn != nil {
		a.conn.Close()
		a.conn = nil
	}
	a.isConnected = false
	a.connMutex.Unlock()

	a.initialized = false
	return nil
}

// Name returns the adapter name
func (a *WebSocketAdapter) Name() string {
	return a.name
}

// Type returns the adapter type
func (a *WebSocketAdapter) Type() string {
	return "websocket"
}

// Config returns the adapter configuration
func (a *WebSocketAdapter) Config() map[string]interface{} {
	return map[string]interface{}{
		"name":        a.name,
		"type":        "websocket",
		"url":         a.config.URL,
		"connected":   a.isConnected,
		"buffer_size": a.config.MessageBufferSize,
	}
}

// This function will be replaced by the central adapter registry
// func init() {
// 	// Adapter factory registration code will go here
// }
