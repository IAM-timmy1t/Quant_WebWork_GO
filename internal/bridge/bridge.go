// bridge.go - Bridge implementation for cross-component communication

package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/plugins"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/discovery"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
)

// Common errors
var (
	ErrAdapterNotFound          = errors.New("adapter not found")
	ErrProtocolNotFound         = errors.New("protocol not found")
	ErrInvalidTarget            = errors.New("invalid target specification")
	ErrInvalidMessage           = errors.New("invalid message format")
	ErrOperationTimeout         = errors.New("operation timed out")
	ErrBridgeNotInitialized     = errors.New("bridge not initialized")
	ErrAdapterInitFailed        = errors.New("adapter initialization failed")
	ErrProtocolInitFailed       = errors.New("protocol initialization failed")
	ErrBridgeServiceUnavailable = errors.New("service unavailable")
	ErrBridgeShuttingDown       = errors.New("bridge is shutting down")
)

// BridgeStatus represents the status of the bridge
type BridgeStatus string

const (
	StatusUninitialized BridgeStatus = "uninitialized"
	StatusInitializing  BridgeStatus = "initializing"
	StatusReady         BridgeStatus = "ready"
	StatusShuttingDown  BridgeStatus = "shutting_down"
	StatusError         BridgeStatus = "error"
)

// BridgeTarget specifies the destination for a message
type BridgeTarget struct {
	Adapter   string                 `json:"adapter"`
	Protocol  string                 `json:"protocol"`
	Service   string                 `json:"service,omitempty"`
	Operation string                 `json:"operation,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// BridgeOptions contains configuration for the bridge
type BridgeOptions struct {
	DefaultTimeout    time.Duration
	RetryCount        int
	RetryDelay        time.Duration
	MaxConcurrency    int
	EnableDiscovery   bool
	EnableMetrics     bool
	EnableCompression bool
	BufferSize        int
	LogLevel          string
}

// DefaultBridgeOptions returns default options
func DefaultBridgeOptions() *BridgeOptions {
	return &BridgeOptions{
		DefaultTimeout:    30 * time.Second,
		RetryCount:        3,
		RetryDelay:        time.Second,
		MaxConcurrency:    100,
		EnableDiscovery:   true,
		EnableMetrics:     true,
		EnableCompression: true,
		BufferSize:        1024,
		LogLevel:          "info",
	}
}

// Bridge provides cross-component communication
type Bridge struct {
	adapters         map[string]adapters.SharedAdapter
	protocols        map[string]*plugins.ProtocolPlugin
	discoveryClient  *discovery.BridgeDiscovery
	metricsCollector *metrics.Collector
	options          *BridgeOptions
	status           BridgeStatus
	adaptersMutex    sync.RWMutex
	protocolsMutex   sync.RWMutex
	statusMutex      sync.RWMutex
	logger           BridgeLogger
	lastError        error
	ctx              context.Context
	cancel           context.CancelFunc
}

// BridgeLogger interface
type BridgeLogger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// NewBridge creates a new bridge
func NewBridge(options *BridgeOptions, logger BridgeLogger) *Bridge {
	if options == nil {
		options = DefaultBridgeOptions()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Bridge{
		adapters:  make(map[string]adapters.SharedAdapter),
		protocols: make(map[string]*plugins.ProtocolPlugin),
		options:   options,
		status:    StatusUninitialized,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Initialize initializes the bridge
func (b *Bridge) Initialize(ctx context.Context) error {
	b.statusMutex.Lock()
	if b.status != StatusUninitialized {
		b.statusMutex.Unlock()
		return fmt.Errorf("bridge already initialized")
	}
	b.status = StatusInitializing
	b.statusMutex.Unlock()

	b.logger.Info("Initializing bridge", nil)

	// Initialize metrics if enabled
	if b.options.EnableMetrics {
		// Create a simple metrics collector
		// Using default config since we just need basic metric tracking
		b.metricsCollector = metrics.NewCollector(nil)
	}

	// Initialize discovery if enabled (skip for now due to compatibility issues)
	// We'll need to adapt the discovery system to match the interface expected here
	// or create a mock implementation for testing
	if b.options.EnableDiscovery {
		b.logger.Info("Discovery service integration disabled pending implementation", nil)
		// This would be implemented later to properly initialize the discovery system
	}

	// Initialize adapters
	b.adaptersMutex.Lock()
	for name, adapter := range b.adapters {
		if err := adapter.Initialize(ctx); err != nil {
			b.logger.Error(fmt.Sprintf("Failed to initialize adapter '%s': %v", name, err), nil)
			b.lastError = fmt.Errorf("%w: %s: %v", ErrAdapterInitFailed, name, err)
		}
	}
	b.adaptersMutex.Unlock()

	// Initialize protocols
	b.protocolsMutex.Lock()
	for name, protocol := range b.protocols {
		if err := protocol.Initialize(ctx, nil); err != nil {
			b.logger.Error(fmt.Sprintf("Failed to initialize protocol '%s': %v", name, err), nil)
			b.lastError = fmt.Errorf("%w: %s: %v", ErrProtocolInitFailed, name, err)
		}
	}
	b.protocolsMutex.Unlock()

	// Update status
	b.statusMutex.Lock()
	b.status = StatusReady
	b.statusMutex.Unlock()

	b.logger.Info("Bridge initialized successfully", nil)
	return nil
}

// RegisterAdapter registers an adapter with the bridge
func (b *Bridge) RegisterAdapter(name string, adapter adapters.SharedAdapter) error {
	b.adaptersMutex.Lock()
	defer b.adaptersMutex.Unlock()

	if _, exists := b.adapters[name]; exists {
		return fmt.Errorf("adapter '%s' already registered", name)
	}

	b.adapters[name] = adapter
	b.logger.Info(fmt.Sprintf("Registered adapter '%s'", name), nil)
	return nil
}

// RegisterProtocol registers a protocol with the bridge
func (b *Bridge) RegisterProtocol(name string, protocol *plugins.ProtocolPlugin) error {
	b.protocolsMutex.Lock()
	defer b.protocolsMutex.Unlock()

	if _, exists := b.protocols[name]; exists {
		return fmt.Errorf("protocol '%s' already registered", name)
	}

	b.protocols[name] = protocol
	b.logger.Info(fmt.Sprintf("Registered protocol '%s'", name), nil)
	return nil
}

// Status returns the current bridge status
func (b *Bridge) Status() BridgeStatus {
	b.statusMutex.Lock()
	defer b.statusMutex.Unlock()
	return b.status
}

// Call sends a message through the bridge
func (b *Bridge) Call(ctx context.Context, target BridgeTarget, operation string, data interface{}) (interface{}, error) {
	// Check bridge status
	if b.Status() != StatusReady {
		return nil, ErrBridgeNotInitialized
	}

	// Validate target
	if target.Adapter == "" || target.Protocol == "" {
		return nil, ErrInvalidTarget
	}

	// Create timeout context if not already specified
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, b.options.DefaultTimeout)
		defer cancel()
	}

	// Set operation if specified
	if operation != "" {
		target.Operation = operation
	}

	// Get protocol
	b.protocolsMutex.RLock()
	protocol, exists := b.protocols[target.Protocol]
	b.protocolsMutex.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProtocolNotFound, target.Protocol)
	}

	// Get adapter
	b.adaptersMutex.RLock()
	adapter, exists := b.adapters[target.Adapter]
	b.adaptersMutex.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrAdapterNotFound, target.Adapter)
	}

	// Create protocol message
	message := &plugins.ProtocolMessage{
		ID:        generateID(),
		Type:      target.Operation,
		Payload:   data,
		Timestamp: time.Now(),
	}

	// Encode message
	encodedData, err := protocol.Encode(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	// Send message through adapter
	start := time.Now()
	response, err := adapter.Send(ctx, encodedData)
	duration := time.Since(start)

	// Record metrics
	if b.metricsCollector != nil {
		tags := map[string]string{
			"adapter":   target.Adapter,
			"protocol":  target.Protocol,
			"operation": target.Operation,
		}

		b.metricsCollector.Collect("bridge", "request_duration", duration.Seconds(), tags)

		// Record error metric if needed
		if err != nil {
			b.metricsCollector.IncCounter("errors", tags)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Decode response
	responseMsg, err := protocol.Decode(ctx, response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return responseMsg.Payload, nil
}

// Shutdown shuts down the bridge
func (b *Bridge) Shutdown(ctx context.Context) error {
	b.statusMutex.Lock()
	if b.status == StatusShuttingDown {
		b.statusMutex.Unlock()
		return nil
	}
	b.status = StatusShuttingDown
	b.statusMutex.Unlock()

	b.logger.Info("Shutting down bridge", nil)

	// Cancel context
	b.cancel()

	// Shutdown adapters
	b.adaptersMutex.Lock()
	for name, adapter := range b.adapters {
		if err := adapter.Shutdown(ctx); err != nil {
			b.logger.Error(fmt.Sprintf("Error shutting down adapter '%s': %v", name, err), nil)
		}
	}
	b.adaptersMutex.Unlock()

	// Clear registries
	b.adaptersMutex.Lock()
	b.adapters = make(map[string]adapters.SharedAdapter)
	b.adaptersMutex.Unlock()

	b.protocolsMutex.Lock()
	b.protocols = make(map[string]*plugins.ProtocolPlugin)
	b.protocolsMutex.Unlock()

	b.logger.Info("Bridge shutdown complete", nil)
	return nil
}

// GetAdapter returns a registered adapter
func (b *Bridge) GetAdapter(name string) (adapters.SharedAdapter, error) {
	b.adaptersMutex.RLock()
	defer b.adaptersMutex.RUnlock()

	adapter, exists := b.adapters[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrAdapterNotFound, name)
	}

	return adapter, nil
}

// GetProtocol returns a registered protocol
func (b *Bridge) GetProtocol(name string) (*plugins.ProtocolPlugin, error) {
	b.protocolsMutex.RLock()
	defer b.protocolsMutex.RUnlock()

	protocol, exists := b.protocols[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProtocolNotFound, name)
	}

	return protocol, nil
}

// ListAdapters returns a list of registered adapters
func (b *Bridge) ListAdapters() []string {
	b.adaptersMutex.RLock()
	defer b.adaptersMutex.RUnlock()

	adapters := make([]string, 0, len(b.adapters))
	for name := range b.adapters {
		adapters = append(adapters, name)
	}

	return adapters
}

// ListProtocols returns a list of registered protocols
func (b *Bridge) ListProtocols() []string {
	b.protocolsMutex.RLock()
	defer b.protocolsMutex.RUnlock()

	protocols := make([]string, 0, len(b.protocols))
	for name := range b.protocols {
		protocols = append(protocols, name)
	}

	return protocols
}

// LastError returns the last error encountered by the bridge
func (b *Bridge) LastError() error {
	return b.lastError
}

// Generate a unique ID for messages
func generateID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}
