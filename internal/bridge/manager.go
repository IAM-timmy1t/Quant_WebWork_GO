// manager.go - Bridge management system

package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager handles lifecycle and operations of multiple bridges
type Manager struct {
	bridges      map[string]Bridge
	protocols    map[string]Protocol
	adapters     map[string]Adapter
	mutex        sync.RWMutex
	config       *ManagerConfig
	logger       Logger
	metrics      MetricsCollector
	eventCh      chan BridgeEvent
	stopCh       chan struct{}
	healthChecks map[string]*healthCheck
}

// ManagerConfig defines configuration for the bridge manager
type ManagerConfig struct {
	// Default configuration for new bridges
	DefaultTimeout         time.Duration
	DefaultRetryCount      int
	DefaultRetryDelay      time.Duration
	HealthCheckInterval    time.Duration
	EventBufferSize        int
	MetricsEnabled         bool
	DefaultProtocol        string
	DefaultAdapter         string
	EnableBridgeDiscovery  bool
	MaxConcurrentBridges   int
	ShutdownTimeout        time.Duration
	AuthenticationRequired bool
}

// healthCheck tracks health information for a bridge
type healthCheck struct {
	bridge       Bridge
	lastCheck    time.Time
	status       HealthStatus
	failureCount int
	latency      time.Duration
}

// HealthStatus represents the health status of a bridge
type HealthStatus string

const (
	// HealthUnknown indicates the bridge's health is unknown
	HealthUnknown HealthStatus = "unknown"
	
	// HealthHealthy indicates the bridge is healthy
	HealthHealthy HealthStatus = "healthy"
	
	// HealthDegraded indicates the bridge is working but with issues
	HealthDegraded HealthStatus = "degraded"
	
	// HealthUnhealthy indicates the bridge is not functioning
	HealthUnhealthy HealthStatus = "unhealthy"
)

// NewManager creates a new bridge manager
func NewManager(config *ManagerConfig) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}
	
	bufferSize := config.EventBufferSize
	if bufferSize <= 0 {
		bufferSize = 100
	}
	
	return &Manager{
		bridges:      make(map[string]Bridge),
		protocols:    make(map[string]Protocol),
		adapters:     make(map[string]Adapter),
		config:       config,
		eventCh:      make(chan BridgeEvent, bufferSize),
		stopCh:       make(chan struct{}),
		healthChecks: make(map[string]*healthCheck),
		logger:       &defaultLogger{},
	}
}

// DefaultManagerConfig returns the default manager configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		DefaultTimeout:        30 * time.Second,
		DefaultRetryCount:     3,
		DefaultRetryDelay:     5 * time.Second,
		HealthCheckInterval:   60 * time.Second,
		EventBufferSize:       100,
		MetricsEnabled:        true,
		MaxConcurrentBridges:  50,
		ShutdownTimeout:       30 * time.Second,
		EnableBridgeDiscovery: true,
	}
}

// SetLogger sets the logger for the manager
func (m *Manager) SetLogger(logger Logger) {
	m.logger = logger
}

// SetMetricsCollector sets the metrics collector
func (m *Manager) SetMetricsCollector(metrics MetricsCollector) {
	m.metrics = metrics
}

// RegisterProtocol registers a protocol implementation
func (m *Manager) RegisterProtocol(protocol Protocol) error {
	if protocol == nil {
		return errors.New("protocol cannot be nil")
	}
	
	name := protocol.Name()
	if name == "" {
		return errors.New("protocol must have a name")
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if _, exists := m.protocols[name]; exists {
		return fmt.Errorf("protocol '%s' is already registered", name)
	}
	
	m.protocols[name] = protocol
	m.logger.Info("Protocol registered", map[string]interface{}{
		"protocol": name,
	})
	
	return nil
}

// RegisterAdapter registers an adapter implementation
func (m *Manager) RegisterAdapter(adapter Adapter) error {
	if adapter == nil {
		return errors.New("adapter cannot be nil")
	}
	
	name := adapter.Name()
	if name == "" {
		return errors.New("adapter must have a name")
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if _, exists := m.adapters[name]; exists {
		return fmt.Errorf("adapter '%s' is already registered", name)
	}
	
	m.adapters[name] = adapter
	m.logger.Info("Adapter registered", map[string]interface{}{
		"adapter": name,
	})
	
	return nil
}

// CreateBridge creates a new bridge instance
func (m *Manager) CreateBridge(config *BridgeConfig) (Bridge, error) {
	if config == nil {
		return nil, errors.New("bridge config cannot be nil")
	}
	
	if config.Name == "" {
		return nil, errors.New("bridge name cannot be empty")
	}
	
	// Validate or set protocol
	protocolName := config.Protocol
	if protocolName == "" {
		protocolName = m.config.DefaultProtocol
		if protocolName == "" {
			return nil, errors.New("no protocol specified and no default protocol configured")
		}
	}
	
	// Validate or set adapter
	adapterName := config.Adapter
	if adapterName == "" {
		adapterName = m.config.DefaultAdapter
		if adapterName == "" {
			return nil, errors.New("no adapter specified and no default adapter configured")
		}
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if protocol exists
	protocol, exists := m.protocols[protocolName]
	if !exists {
		return nil, fmt.Errorf("protocol '%s' is not registered", protocolName)
	}
	
	// Check if adapter exists
	adapter, exists := m.adapters[adapterName]
	if !exists {
		return nil, fmt.Errorf("adapter '%s' is not registered", adapterName)
	}
	
	// Generate ID if not provided
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	
	// Check for duplicate IDs
	if _, exists := m.bridges[config.ID]; exists {
		return nil, fmt.Errorf("bridge with ID '%s' already exists", config.ID)
	}
	
	// Set default timeout if not specified
	if config.Timeout <= 0 {
		config.Timeout = m.config.DefaultTimeout
	}
	
	// Create the bridge
	bridge, err := NewBridge(config, protocol, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create bridge: %w", err)
	}
	
	// Register the bridge
	m.bridges[config.ID] = bridge
	
	// Initialize health check
	m.healthChecks[config.ID] = &healthCheck{
		bridge:    bridge,
		lastCheck: time.Time{},
		status:    HealthUnknown,
	}
	
	m.logger.Info("Bridge created", map[string]interface{}{
		"id":       config.ID,
		"name":     config.Name,
		"protocol": protocolName,
		"adapter":  adapterName,
	})
	
	return bridge, nil
}

// GetBridge retrieves a bridge by ID
func (m *Manager) GetBridge(id string) (Bridge, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	bridge, exists := m.bridges[id]
	if !exists {
		return nil, fmt.Errorf("bridge with ID '%s' not found", id)
	}
	
	return bridge, nil
}

// ListBridges returns all registered bridges
func (m *Manager) ListBridges() []Bridge {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	bridges := make([]Bridge, 0, len(m.bridges))
	for _, bridge := range m.bridges {
		bridges = append(bridges, bridge)
	}
	
	return bridges
}

// DestroyBridge removes a bridge
func (m *Manager) DestroyBridge(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	bridge, exists := m.bridges[id]
	if !exists {
		return fmt.Errorf("bridge with ID '%s' not found", id)
	}
	
	// Close the bridge
	if err := bridge.Close(); err != nil {
		m.logger.Warn("Error closing bridge", map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		})
	}
	
	// Remove from manager
	delete(m.bridges, id)
	delete(m.healthChecks, id)
	
	m.logger.Info("Bridge destroyed", map[string]interface{}{
		"id": id,
	})
	
	return nil
}

// Start starts the bridge manager
func (m *Manager) Start(ctx context.Context) error {
	go m.runHealthChecks(ctx)
	go m.processEvents(ctx)
	
	m.logger.Info("Bridge manager started", nil)
	
	return nil
}

// Stop stops the bridge manager
func (m *Manager) Stop() error {
	close(m.stopCh)
	
	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), m.config.ShutdownTimeout)
	defer cancel()
	
	// Close all bridges
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for id, bridge := range m.bridges {
		if err := bridge.Close(); err != nil {
			m.logger.Warn("Error closing bridge", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
		}
	}
	
	m.logger.Info("Bridge manager stopped", nil)
	
	return nil
}

// runHealthChecks periodically checks the health of all bridges
func (m *Manager) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkBridgesHealth()
		}
	}
}

// checkBridgesHealth checks the health of all bridges
func (m *Manager) checkBridgesHealth() {
	m.mutex.RLock()
	bridgeIDs := make([]string, 0, len(m.bridges))
	for id := range m.bridges {
		bridgeIDs = append(bridgeIDs, id)
	}
	m.mutex.RUnlock()
	
	for _, id := range bridgeIDs {
		m.checkBridgeHealth(id)
	}
}

// checkBridgeHealth checks the health of a single bridge
func (m *Manager) checkBridgeHealth(id string) {
	m.mutex.RLock()
	bridge, exists := m.bridges[id]
	check, checkExists := m.healthChecks[id]
	m.mutex.RUnlock()
	
	if !exists || !checkExists {
		return
	}
	
	startTime := time.Now()
	
	// Execute health check
	err := bridge.Ping(context.Background())
	
	latency := time.Since(startTime)
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Update health check status
	check.lastCheck = time.Now()
	check.latency = latency
	
	if err != nil {
		check.failureCount++
		if check.failureCount >= 3 {
			check.status = HealthUnhealthy
		} else if check.failureCount >= 1 {
			check.status = HealthDegraded
		}
		
		m.logger.Warn("Bridge health check failed", map[string]interface{}{
			"id":           id,
			"error":        err.Error(),
			"failureCount": check.failureCount,
			"status":       check.status,
		})
	} else {
		if check.failureCount > 0 {
			// Reset failure count but stay degraded for one more cycle
			check.failureCount = 0
			if check.status == HealthUnhealthy {
				check.status = HealthDegraded
			} else {
				check.status = HealthHealthy
			}
		} else {
			check.status = HealthHealthy
		}
		
		m.logger.Debug("Bridge health check passed", map[string]interface{}{
			"id":      id,
			"latency": latency.String(),
			"status":  check.status,
		})
	}
	
	// Record metrics if enabled
	if m.metrics != nil && m.config.MetricsEnabled {
		m.metrics.RecordLatency("bridge.health_check", float64(latency.Milliseconds()), map[string]string{
			"bridge_id": id,
			"status":    string(check.status),
		})
	}
}

// GetBridgeHealth returns the health status of a bridge
func (m *Manager) GetBridgeHealth(id string) (HealthStatus, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	check, exists := m.healthChecks[id]
	if !exists {
		return HealthUnknown, fmt.Errorf("bridge with ID '%s' not found", id)
	}
	
	return check.status, nil
}

// processEvents processes bridge events
func (m *Manager) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case event := <-m.eventCh:
			m.handleEvent(event)
		}
	}
}

// handleEvent processes a single bridge event
func (m *Manager) handleEvent(event BridgeEvent) {
	if m.metrics != nil && m.config.MetricsEnabled {
		m.metrics.IncrementCounter("bridge.event", map[string]string{
			"bridge_id": event.BridgeID,
			"type":      event.Type,
		})
	}
	
	m.logger.Debug("Bridge event received", map[string]interface{}{
		"bridge_id": event.BridgeID,
		"type":      event.Type,
		"timestamp": event.Timestamp,
	})
	
	// Handle specific event types
	switch event.Type {
	case EventConnected:
		m.handleConnectedEvent(event)
	case EventDisconnected:
		m.handleDisconnectedEvent(event)
	case EventError:
		m.handleErrorEvent(event)
	}
}

// handleConnectedEvent handles a bridge connected event
func (m *Manager) handleConnectedEvent(event BridgeEvent) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if check, exists := m.healthChecks[event.BridgeID]; exists {
		check.status = HealthHealthy
		check.failureCount = 0
	}
}

// handleDisconnectedEvent handles a bridge disconnected event
func (m *Manager) handleDisconnectedEvent(event BridgeEvent) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if check, exists := m.healthChecks[event.BridgeID]; exists {
		check.status = HealthUnhealthy
		check.failureCount++
	}
}

// handleErrorEvent handles a bridge error event
func (m *Manager) handleErrorEvent(event BridgeEvent) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if check, exists := m.healthChecks[event.BridgeID]; exists {
		check.failureCount++
		if check.failureCount >= 3 {
			check.status = HealthUnhealthy
		} else {
			check.status = HealthDegraded
		}
	}
	
	// Log the error
	m.logger.Error("Bridge error", map[string]interface{}{
		"bridge_id": event.BridgeID,
		"error":     event.Payload,
	})
}

// Adapter for interfacing with metrics collector
type MetricsCollector interface {
	RecordLatency(name string, valueMs float64, tags map[string]string)
	IncrementCounter(name string, tags map[string]string)
}

// Logger interface for bridge manager
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// defaultLogger is a basic implementation if none is provided
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *defaultLogger) Info(msg string, fields map[string]interface{})  {}
func (l *defaultLogger) Warn(msg string, fields map[string]interface{})  {}
func (l *defaultLogger) Error(msg string, fields map[string]interface{}) {}
