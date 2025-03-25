// manager.go - Bridge manager for coordinating adapters and protocols

package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapter"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/protocol"
)

// BridgeEvent represents an event in the bridge system
type BridgeEvent struct {
	Type      string                 // Event type
	Source    string                 // Event source
	Timestamp time.Time              // Event timestamp
	Data      interface{}            // Event data
	Metadata  map[string]interface{} // Additional metadata
}

// EventHandlerFunc defines a function type for handling bridge events
type EventHandlerFunc func(ctx context.Context, event *BridgeEvent) error

// BridgeConfig contains configuration for the bridge manager
type BridgeConfig struct {
	Name                 string        // Name of the bridge
	EnableEventLogging   bool          // Whether to log events
	DefaultTimeout       time.Duration // Default timeout for operations
	MaxConcurrentTasks   int           // Maximum number of concurrent tasks
	HeartbeatInterval    time.Duration // Interval for adapter heartbeats
	ConnectionRetryLimit int           // Number of retries for connection attempts
	AutoReconnect        bool          // Whether to automatically reconnect adapters
}

// defaultBridgeConfig returns the default bridge configuration
func defaultBridgeConfig() *BridgeConfig {
	return &BridgeConfig{
		Name:                 "default",
		EnableEventLogging:   true,
		DefaultTimeout:       30 * time.Second,
		MaxConcurrentTasks:   10,
		HeartbeatInterval:    15 * time.Second,
		ConnectionRetryLimit: 5,
		AutoReconnect:        true,
	}
}

// BridgeManager manages bridge adapters and protocols
type BridgeManager struct {
	config           *BridgeConfig
	adapters         map[string]adapter.Adapter
	protocols        map[string]protocol.Protocol
	eventHandlers    map[string][]EventHandlerFunc
	connectionStatus map[string]bool
	adaptersMutex    sync.RWMutex
	protocolsMutex   sync.RWMutex
	handlersMutex    sync.RWMutex
	statusMutex      sync.RWMutex
	logger           Logger
	shutdown         chan struct{}
	wg               sync.WaitGroup
}

// Logger interface for bridge logging
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// New creates a new bridge manager
func New(config *BridgeConfig, logger Logger) *BridgeManager {
	if config == nil {
		config = defaultBridgeConfig()
	}
	
	if logger == nil {
		// Use a no-op logger if none provided
		logger = &noopLogger{}
	}
	
	return &BridgeManager{
		config:           config,
		adapters:         make(map[string]adapter.Adapter),
		protocols:        make(map[string]protocol.Protocol),
		eventHandlers:    make(map[string][]EventHandlerFunc),
		connectionStatus: make(map[string]bool),
		logger:           logger,
		shutdown:         make(chan struct{}),
	}
}

// Start initializes and starts the bridge manager
func (m *BridgeManager) Start(ctx context.Context) error {
	m.logger.Info("Starting bridge manager", map[string]interface{}{
		"name": m.config.Name,
	})
	
	// Start heartbeat monitoring if enabled
	if m.config.HeartbeatInterval > 0 {
		m.wg.Add(1)
		go m.heartbeatMonitor(ctx)
	}
	
	return nil
}

// Stop gracefully stops the bridge manager
func (m *BridgeManager) Stop(ctx context.Context) error {
	m.logger.Info("Stopping bridge manager", map[string]interface{}{
		"name": m.config.Name,
	})
	
	// Signal shutdown
	close(m.shutdown)
	
	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	
	// Wait for shutdown to complete or context to be canceled
	select {
	case <-done:
		m.logger.Info("Bridge manager stopped", map[string]interface{}{
			"name": m.config.Name,
		})
	case <-ctx.Done():
		m.logger.Warn("Bridge manager shutdown context canceled", map[string]interface{}{
			"name":  m.config.Name,
			"error": ctx.Err(),
		})
		return ctx.Err()
	}
	
	return nil
}

// RegisterAdapter registers a bridge adapter
func (m *BridgeManager) RegisterAdapter(adapter adapter.Adapter) error {
	if adapter == nil {
		return errors.New("adapter cannot be nil")
	}
	
	name := adapter.Name()
	if name == "" {
		return errors.New("adapter must have a name")
	}
	
	m.adaptersMutex.Lock()
	defer m.adaptersMutex.Unlock()
	
	if _, exists := m.adapters[name]; exists {
		return fmt.Errorf("adapter with name '%s' already registered", name)
	}
	
	m.adapters[name] = adapter
	m.connectionStatus[name] = false
	
	m.logger.Info("Registered adapter", map[string]interface{}{
		"adapter": name,
		"type":    adapter.Type(),
	})
	
	return nil
}

// UnregisterAdapter unregisters a bridge adapter
func (m *BridgeManager) UnregisterAdapter(name string) error {
	m.adaptersMutex.Lock()
	defer m.adaptersMutex.Unlock()
	
	adapter, exists := m.adapters[name]
	if !exists {
		return fmt.Errorf("adapter '%s' not found", name)
	}
	
	// Disconnect adapter if connected
	if m.connectionStatus[name] {
		ctx, cancel := context.WithTimeout(context.Background(), m.config.DefaultTimeout)
		defer cancel()
		
		if err := adapter.Disconnect(ctx); err != nil {
			m.logger.Warn("Error disconnecting adapter during unregistration", map[string]interface{}{
				"adapter": name,
				"error":   err.Error(),
			})
		}
	}
	
	delete(m.adapters, name)
	delete(m.connectionStatus, name)
	
	m.logger.Info("Unregistered adapter", map[string]interface{}{
		"adapter": name,
	})
	
	return nil
}

// GetAdapter returns a registered adapter by name
func (m *BridgeManager) GetAdapter(name string) (adapter.Adapter, error) {
	m.adaptersMutex.RLock()
	defer m.adaptersMutex.RUnlock()
	
	adapter, exists := m.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter '%s' not found", name)
	}
	
	return adapter, nil
}

// ListAdapters returns a list of all registered adapters
func (m *BridgeManager) ListAdapters() []adapter.Adapter {
	m.adaptersMutex.RLock()
	defer m.adaptersMutex.RUnlock()
	
	adapters := make([]adapter.Adapter, 0, len(m.adapters))
	for _, a := range m.adapters {
		adapters = append(adapters, a)
	}
	
	return adapters
}

// RegisterProtocol registers a communication protocol
func (m *BridgeManager) RegisterProtocol(protocol protocol.Protocol) error {
	if protocol == nil {
		return errors.New("protocol cannot be nil")
	}
	
	name := protocol.Name()
	if name == "" {
		return errors.New("protocol must have a name")
	}
	
	m.protocolsMutex.Lock()
	defer m.protocolsMutex.Unlock()
	
	if _, exists := m.protocols[name]; exists {
		return fmt.Errorf("protocol with name '%s' already registered", name)
	}
	
	m.protocols[name] = protocol
	
	m.logger.Info("Registered protocol", map[string]interface{}{
		"protocol": name,
		"type":     protocol.Type(),
		"version":  protocol.Version(),
	})
	
	return nil
}

// UnregisterProtocol unregisters a communication protocol
func (m *BridgeManager) UnregisterProtocol(name string) error {
	m.protocolsMutex.Lock()
	defer m.protocolsMutex.Unlock()
	
	if _, exists := m.protocols[name]; !exists {
		return fmt.Errorf("protocol '%s' not found", name)
	}
	
	delete(m.protocols, name)
	
	m.logger.Info("Unregistered protocol", map[string]interface{}{
		"protocol": name,
	})
	
	return nil
}

// GetProtocol returns a registered protocol by name
func (m *BridgeManager) GetProtocol(name string) (protocol.Protocol, error) {
	m.protocolsMutex.RLock()
	defer m.protocolsMutex.RUnlock()
	
	protocol, exists := m.protocols[name]
	if !exists {
		return nil, fmt.Errorf("protocol '%s' not found", name)
	}
	
	return protocol, nil
}

// ListProtocols returns a list of all registered protocols
func (m *BridgeManager) ListProtocols() []protocol.Protocol {
	m.protocolsMutex.RLock()
	defer m.protocolsMutex.RUnlock()
	
	protocols := make([]protocol.Protocol, 0, len(m.protocols))
	for _, p := range m.protocols {
		protocols = append(protocols, p)
	}
	
	return protocols
}

// ConnectAdapter connects a bridge adapter
func (m *BridgeManager) ConnectAdapter(ctx context.Context, name string) error {
	m.adaptersMutex.RLock()
	adapter, exists := m.adapters[name]
	m.adaptersMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("adapter '%s' not found", name)
	}
	
	// Initialize the adapter if needed
	if adapter.Status() == adapter.StatusInitializing {
		if err := adapter.Initialize(ctx); err != nil {
			m.logger.Error("Failed to initialize adapter", map[string]interface{}{
				"adapter": name,
				"error":   err.Error(),
			})
			return fmt.Errorf("failed to initialize adapter: %w", err)
		}
	}
	
	// Connect the adapter
	if err := adapter.Connect(ctx); err != nil {
		m.logger.Error("Failed to connect adapter", map[string]interface{}{
			"adapter": name,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to connect adapter: %w", err)
	}
	
	// Update connection status
	m.statusMutex.Lock()
	m.connectionStatus[name] = true
	m.statusMutex.Unlock()
	
	m.logger.Info("Connected adapter", map[string]interface{}{
		"adapter": name,
	})
	
	// Raise an event for the connection
	m.raiseEvent(ctx, &BridgeEvent{
		Type:      "adapter.connected",
		Source:    name,
		Timestamp: time.Now(),
		Data:      adapter.Status(),
	})
	
	return nil
}

// DisconnectAdapter disconnects a bridge adapter
func (m *BridgeManager) DisconnectAdapter(ctx context.Context, name string) error {
	m.adaptersMutex.RLock()
	adapter, exists := m.adapters[name]
	m.adaptersMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("adapter '%s' not found", name)
	}
	
	// Disconnect the adapter
	if err := adapter.Disconnect(ctx); err != nil {
		m.logger.Error("Failed to disconnect adapter", map[string]interface{}{
			"adapter": name,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to disconnect adapter: %w", err)
	}
	
	// Update connection status
	m.statusMutex.Lock()
	m.connectionStatus[name] = false
	m.statusMutex.Unlock()
	
	m.logger.Info("Disconnected adapter", map[string]interface{}{
		"adapter": name,
	})
	
	// Raise an event for the disconnection
	m.raiseEvent(ctx, &BridgeEvent{
		Type:      "adapter.disconnected",
		Source:    name,
		Timestamp: time.Now(),
		Data:      adapter.Status(),
	})
	
	return nil
}

// SendMessage sends a message through a bridge adapter
func (m *BridgeManager) SendMessage(ctx context.Context, adapterName string, message interface{}) error {
	m.adaptersMutex.RLock()
	adapter, exists := m.adapters[adapterName]
	connected := m.connectionStatus[adapterName]
	m.adaptersMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("adapter '%s' not found", adapterName)
	}
	
	if !connected {
		return fmt.Errorf("adapter '%s' is not connected", adapterName)
	}
	
	// Send the message
	if err := adapter.Send(ctx, message); err != nil {
		m.logger.Error("Failed to send message", map[string]interface{}{
			"adapter": adapterName,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	return nil
}

// RegisterEventHandler registers a handler for bridge events
func (m *BridgeManager) RegisterEventHandler(eventType string, handler EventHandlerFunc) {
	m.handlersMutex.Lock()
	defer m.handlersMutex.Unlock()
	
	m.eventHandlers[eventType] = append(m.eventHandlers[eventType], handler)
	
	m.logger.Debug("Registered event handler", map[string]interface{}{
		"event_type": eventType,
	})
}

// UnregisterEventHandler removes all handlers for an event type
func (m *BridgeManager) UnregisterEventHandler(eventType string) {
	m.handlersMutex.Lock()
	defer m.handlersMutex.Unlock()
	
	delete(m.eventHandlers, eventType)
	
	m.logger.Debug("Unregistered event handlers", map[string]interface{}{
		"event_type": eventType,
	})
}

// raiseEvent raises a bridge event and notifies handlers
func (m *BridgeManager) raiseEvent(ctx context.Context, event *BridgeEvent) {
	if event == nil {
		return
	}
	
	// Log the event if enabled
	if m.config.EnableEventLogging {
		m.logger.Debug("Bridge event", map[string]interface{}{
			"event_type": event.Type,
			"source":     event.Source,
			"timestamp":  event.Timestamp,
		})
	}
	
	// Get handlers for this event type
	m.handlersMutex.RLock()
	handlers := m.eventHandlers[event.Type]
	m.handlersMutex.RUnlock()
	
	// No handlers for this event type
	if len(handlers) == 0 {
		return
	}
	
	// Notify handlers
	for _, handler := range handlers {
		// Execute handler in a separate goroutine
		m.wg.Add(1)
		go func(h EventHandlerFunc) {
			defer m.wg.Done()
			
			// Create a context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, m.config.DefaultTimeout)
			defer cancel()
			
			// Call the handler
			if err := h(timeoutCtx, event); err != nil {
				m.logger.Error("Event handler error", map[string]interface{}{
					"event_type": event.Type,
					"error":      err.Error(),
				})
			}
		}(handler)
	}
}

// heartbeatMonitor periodically checks adapter health
func (m *BridgeManager) heartbeatMonitor(ctx context.Context) {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.HeartbeatInterval)
	defer ticker.Stop()
	
	m.logger.Debug("Starting adapter heartbeat monitor", map[string]interface{}{
		"interval": m.config.HeartbeatInterval.String(),
	})
	
	for {
		select {
		case <-ticker.C:
			m.checkAdaptersHealth(ctx)
		case <-m.shutdown:
			m.logger.Debug("Stopping adapter heartbeat monitor", nil)
			return
		case <-ctx.Done():
			m.logger.Debug("Context canceled, stopping adapter heartbeat monitor", nil)
			return
		}
	}
}

// checkAdaptersHealth checks the health of all connected adapters
func (m *BridgeManager) checkAdaptersHealth(ctx context.Context) {
	m.adaptersMutex.RLock()
	adapters := make(map[string]adapter.Adapter)
	for name, a := range m.adapters {
		adapters[name] = a
	}
	m.statusMutex.RLock()
	statuses := make(map[string]bool)
	for name, status := range m.connectionStatus {
		statuses[name] = status
	}
	m.statusMutex.RUnlock()
	m.adaptersMutex.RUnlock()
	
	for name, a := range adapters {
		// Skip adapters that aren't connected
		if !statuses[name] {
			continue
		}
		
		// Check adapter status
		status := a.Status()
		if status == adapter.StatusError || status == adapter.StatusDisconnected {
			m.logger.Warn("Adapter health check failed", map[string]interface{}{
				"adapter": name,
				"status":  status,
				"error":   a.LastError(),
			})
			
			// Auto-reconnect if enabled
			if m.config.AutoReconnect {
				m.logger.Info("Attempting to reconnect adapter", map[string]interface{}{
					"adapter": name,
				})
				
				// Create a new context for reconnection
				reconnectCtx, cancel := context.WithTimeout(context.Background(), m.config.DefaultTimeout)
				
				// Attempt to reconnect
				go func(adapterName string) {
					defer cancel()
					
					if err := m.ConnectAdapter(reconnectCtx, adapterName); err != nil {
						m.logger.Error("Failed to reconnect adapter", map[string]interface{}{
							"adapter": adapterName,
							"error":   err.Error(),
						})
						
						// Raise an event for the reconnection failure
						m.raiseEvent(context.Background(), &BridgeEvent{
							Type:      "adapter.reconnect_failed",
							Source:    adapterName,
							Timestamp: time.Now(),
							Data:      err.Error(),
						})
					} else {
						m.logger.Info("Successfully reconnected adapter", map[string]interface{}{
							"adapter": adapterName,
						})
						
						// Raise an event for the successful reconnection
						m.raiseEvent(context.Background(), &BridgeEvent{
							Type:      "adapter.reconnected",
							Source:    adapterName,
							Timestamp: time.Now(),
						})
					}
				}(name)
			}
		}
	}
}

// CreateBridge creates a new bridge between two adapters
func (m *BridgeManager) CreateBridge(ctx context.Context, sourceName, targetName string) error {
	m.adaptersMutex.RLock()
	sourceAdapter, sourceExists := m.adapters[sourceName]
	targetAdapter, targetExists := m.adapters[targetName]
	m.adaptersMutex.RUnlock()
	
	if !sourceExists {
		return fmt.Errorf("source adapter '%s' not found", sourceName)
	}
	
	if !targetExists {
		return fmt.Errorf("target adapter '%s' not found", targetName)
	}
	
	// Create a message handler for the source adapter
	sourceAdapter.SetMessageHandler(func(ctx context.Context, message interface{}) error {
		// Forward the message to the target adapter
		return targetAdapter.Send(ctx, message)
	})
	
	m.logger.Info("Created bridge between adapters", map[string]interface{}{
		"source": sourceName,
		"target": targetName,
	})
	
	return nil
}

// noopLogger is a no-op implementation of the Logger interface
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *noopLogger) Info(msg string, fields map[string]interface{})  {}
func (l *noopLogger) Warn(msg string, fields map[string]interface{})  {}
func (l *noopLogger) Error(msg string, fields map[string]interface{}) {}




