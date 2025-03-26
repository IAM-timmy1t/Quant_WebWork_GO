// adapter_shared.go - Shared definitions for bridge adapters

package adapters

import (
	"context"
	"sync"
	"time"
)

// SharedAdapterStatus represents the status of an adapter
type SharedAdapterStatus string

// Adapter status constants
const (
	SharedStatusUninitialized SharedAdapterStatus = "uninitialized"
	SharedStatusInitialized   SharedAdapterStatus = "initialized"
	SharedStatusConnecting    SharedAdapterStatus = "connecting"
	SharedStatusConnected     SharedAdapterStatus = "connected"
	SharedStatusDisconnecting SharedAdapterStatus = "disconnecting"
	SharedStatusDisconnected  SharedAdapterStatus = "disconnected"
	SharedStatusError         SharedAdapterStatus = "error"
)

// SharedAdapterConfig contains common configuration for all adapters
type SharedAdapterConfig struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Protocol   string                 `json:"protocol"`
	Host       string                 `json:"host"`
	Port       int                    `json:"port"`
	Path       string                 `json:"path"`
	Timeout    time.Duration          `json:"timeout"`
	RetryCount int                    `json:"retry_count"`
	RetryDelay time.Duration          `json:"retry_delay"`
	Options    map[string]interface{} `json:"options"`
}

// SharedAdapterMetadata contains metadata about an adapter
type SharedAdapterMetadata struct {
	Version       string            `json:"version"`
	Capabilities  []string          `json:"capabilities"`
	Author        string            `json:"author"`
	Documentation string            `json:"documentation"`
	Properties    map[string]string `json:"properties"`
}

// SharedAdapterStats contains statistics about an adapter
type SharedAdapterStats struct {
	MessagesSent        int64                  `json:"messages_sent"`
	MessagesReceived    int64                  `json:"messages_received"`
	BytesSent           int64                  `json:"bytes_sent"`
	BytesReceived       int64                  `json:"bytes_received"`
	Errors              int64                  `json:"errors"`
	ConnectCount        int64                  `json:"connect_count"`
	DisconnectCount     int64                  `json:"disconnect_count"`
	LastConnectTime     time.Time              `json:"last_connect_time"`
	LastDisconnectTime  time.Time              `json:"last_disconnect_time"`
	AverageResponseTime time.Duration          `json:"average_response_time"`
	MaxResponseTime     time.Duration          `json:"max_response_time"`
	MinResponseTime     time.Duration          `json:"min_response_time"`
	Uptime              time.Duration          `json:"uptime"`
	CustomStats         map[string]interface{} `json:"custom_stats"`
}

// SharedAdapter defines the interface for bridge adapters
type SharedAdapter interface {
	// Lifecycle methods
	Initialize(ctx context.Context) error
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Shutdown(ctx context.Context) error

	// Status methods
	Status() SharedAdapterStatus
	Stats() SharedAdapterStats

	// Configuration and information
	Name() string
	Type() string
	Metadata() SharedAdapterMetadata
	Config() SharedAdapterConfig

	// Communication methods
	Send(ctx context.Context, data []byte) ([]byte, error)
	Receive(ctx context.Context) ([]byte, error)

	// Error handling
	LastError() error
}

// SharedAdapterFactory creates a new adapter
type SharedAdapterFactory func(config SharedAdapterConfig) (SharedAdapter, error)

// SharedAdapterRegistry manages adapter factories
type SharedAdapterRegistry struct {
	factories map[string]SharedAdapterFactory
	mutex     sync.RWMutex
}

// NewSharedAdapterRegistry creates a new adapter registry
func NewSharedAdapterRegistry() *SharedAdapterRegistry {
	return &SharedAdapterRegistry{
		factories: make(map[string]SharedAdapterFactory),
	}
}

// RegisterFactory registers an adapter factory
func (r *SharedAdapterRegistry) RegisterFactory(adapterType string, factory SharedAdapterFactory) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.factories[adapterType] = factory
}

// GetFactory returns an adapter factory by type
func (r *SharedAdapterRegistry) GetFactory(adapterType string) (SharedAdapterFactory, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	factory, exists := r.factories[adapterType]
	return factory, exists
}

// CreateAdapter creates a new adapter using a registered factory
func (r *SharedAdapterRegistry) CreateAdapter(config SharedAdapterConfig) (SharedAdapter, error) {
	factory, exists := r.GetFactory(config.Type)
	if !exists {
		return nil, SharedErrUnknownAdapterType
	}
	return factory(config)
}

// ListAdapterTypes returns a list of all registered adapter types
func (r *SharedAdapterRegistry) ListAdapterTypes() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	types := make([]string, 0, len(r.factories))
	for adapterType := range r.factories {
		types = append(types, adapterType)
	}
	return types
}

// SharedBaseAdapter provides a basic implementation of the SharedAdapter interface
type SharedBaseAdapter struct {
	name        string
	adapterType string
	config      SharedAdapterConfig
	metadata    SharedAdapterMetadata
	status      SharedAdapterStatus
	stats       SharedAdapterStats
	lastError   error
	mutex       sync.RWMutex
	startTime   time.Time
}

// NewSharedBaseAdapter creates a new base adapter
func NewSharedBaseAdapter(name, adapterType string, config SharedAdapterConfig, metadata SharedAdapterMetadata) *SharedBaseAdapter {
	return &SharedBaseAdapter{
		name:        name,
		adapterType: adapterType,
		config:      config,
		metadata:    metadata,
		status:      SharedStatusUninitialized,
		stats: SharedAdapterStats{
			CustomStats: make(map[string]interface{}),
		},
	}
}

// Name returns the adapter name
func (a *SharedBaseAdapter) Name() string {
	return a.name
}

// Type returns the adapter type
func (a *SharedBaseAdapter) Type() string {
	return a.adapterType
}

// Metadata returns the adapter metadata
func (a *SharedBaseAdapter) Metadata() SharedAdapterMetadata {
	return a.metadata
}

// Config returns the adapter configuration
func (a *SharedBaseAdapter) Config() SharedAdapterConfig {
	// Return a copy to prevent modification
	config := a.config

	// Redact sensitive information
	if options, ok := config.Options["password"]; ok {
		newOptions := make(map[string]interface{})
		for k, v := range config.Options {
			if k == "password" || k == "secret" || k == "api_key" || k == "token" {
				newOptions[k] = "******"
			} else {
				newOptions[k] = v
			}
		}
		config.Options = newOptions
	}

	return config
}

// Status returns the adapter status
func (a *SharedBaseAdapter) Status() SharedAdapterStatus {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.status
}

// Stats returns the adapter statistics
func (a *SharedBaseAdapter) Stats() SharedAdapterStats {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// Calculate uptime if connected
	stats := a.stats
	if a.status == SharedStatusConnected && !a.startTime.IsZero() {
		stats.Uptime = time.Since(a.startTime)
	}

	return stats
}

// LastError returns the last error encountered by the adapter
func (a *SharedBaseAdapter) LastError() error {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.lastError
}

// setStatus sets the adapter status
func (a *SharedBaseAdapter) setStatus(status SharedAdapterStatus) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.status = status

	if status == SharedStatusConnected {
		a.startTime = time.Now()
		a.stats.LastConnectTime = a.startTime
		a.stats.ConnectCount++
	} else if status == SharedStatusDisconnected {
		a.stats.LastDisconnectTime = time.Now()
		a.stats.DisconnectCount++
	}
}

// setError sets the last error
func (a *SharedBaseAdapter) setError(err error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.lastError = err
	if err != nil {
		a.stats.Errors++
	}
}

// recordSend records message send statistics
func (a *SharedBaseAdapter) recordSend(bytes int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.stats.MessagesSent++
	a.stats.BytesSent += int64(bytes)
}

// recordReceive records message receive statistics
func (a *SharedBaseAdapter) recordReceive(bytes int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.stats.MessagesReceived++
	a.stats.BytesReceived += int64(bytes)
}

// updateResponseTime updates response time statistics
func (a *SharedBaseAdapter) updateResponseTime(duration time.Duration) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	count := float64(a.stats.MessagesSent)
	if count == 1 {
		// First time
		a.stats.AverageResponseTime = duration
		a.stats.MinResponseTime = duration
		a.stats.MaxResponseTime = duration
	} else {
		// Update average (incremental formula to avoid overflow)
		a.stats.AverageResponseTime = time.Duration(
			(float64(a.stats.AverageResponseTime)*(count-1) + float64(duration)) / count,
		)

		// Update min/max
		if duration < a.stats.MinResponseTime {
			a.stats.MinResponseTime = duration
		}
		if duration > a.stats.MaxResponseTime {
			a.stats.MaxResponseTime = duration
		}
	}
}

// SharedLogger defines a logging interface for adapters
type SharedLogger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// Common adapter errors
var (
	SharedErrNotInitialized     = NewSharedAdapterError("adapter not initialized")
	SharedErrAlreadyInitialized = NewSharedAdapterError("adapter already initialized")
	SharedErrNotConnected       = NewSharedAdapterError("adapter not connected")
	SharedErrAlreadyConnected   = NewSharedAdapterError("adapter already connected")
	SharedErrConnectionFailed   = NewSharedAdapterError("connection failed")
	SharedErrInvalidConfig      = NewSharedAdapterError("invalid configuration")
	SharedErrTimeout            = NewSharedAdapterError("operation timed out")
	SharedErrClosed             = NewSharedAdapterError("adapter closed")
	SharedErrUnknownAdapterType = NewSharedAdapterError("unknown adapter type")
	SharedErrInvalidData        = NewSharedAdapterError("invalid data format")
)

// SharedAdapterError represents an adapter-specific error
type SharedAdapterError struct {
	msg string
}

// NewSharedAdapterError creates a new adapter error
func NewSharedAdapterError(msg string) *SharedAdapterError {
	return &SharedAdapterError{msg: msg}
}

// Error implements the error interface
func (e *SharedAdapterError) Error() string {
	return e.msg
}

// Global adapter registry for convenience
var sharedGlobalRegistry = NewSharedAdapterRegistry()

// RegisterSharedAdapterFactory registers an adapter factory with the global registry
func RegisterSharedAdapterFactory(adapterType string, factory SharedAdapterFactory) {
	sharedGlobalRegistry.RegisterFactory(adapterType, factory)
}

// GetSharedAdapterFactory returns an adapter factory from the global registry
func GetSharedAdapterFactory(adapterType string) (SharedAdapterFactory, bool) {
	return sharedGlobalRegistry.GetFactory(adapterType)
}

// CreateSharedAdapterInstance creates a new adapter using the global registry
func CreateSharedAdapterInstance(config SharedAdapterConfig) (SharedAdapter, error) {
	return sharedGlobalRegistry.CreateAdapter(config)
}

// ListAvailableSharedAdapterTypes returns a list of all registered adapter types from the global registry
func ListAvailableSharedAdapterTypes() []string {
	return sharedGlobalRegistry.ListAdapterTypes()
}
