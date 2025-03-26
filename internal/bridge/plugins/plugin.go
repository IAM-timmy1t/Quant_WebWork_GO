// plugin.go - Plugin system for bridge components

package plugins

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Common errors
var (
	ErrPluginNotFound       = errors.New("plugin not found")
	ErrPluginAlreadyExists  = errors.New("plugin already exists")
	ErrPluginNotInitialized = errors.New("plugin not initialized")
	ErrInvalidPluginType    = errors.New("invalid plugin type")
)

// PluginType represents the type of a plugin
type PluginType string

// Plugin types
const (
	PluginTypeAdapter  PluginType = "adapter"
	PluginTypeProtocol PluginType = "protocol"
	PluginTypeUtility  PluginType = "utility"
	PluginTypeMonitor  PluginType = "monitor"
	PluginTypeSecurity PluginType = "security"
)

// PluginStatus represents the status of a plugin
type PluginStatus string

// Plugin statuses
const (
	PluginStatusUninitialized PluginStatus = "uninitialized"
	PluginStatusInitialized   PluginStatus = "initialized"
	PluginStatusStarted       PluginStatus = "started"
	PluginStatusStopped       PluginStatus = "stopped"
	PluginStatusError         PluginStatus = "error"
)

// PluginMetadata contains metadata about a plugin
type PluginMetadata struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	Homepage     string            `json:"homepage"`
	Repository   string            `json:"repository"`
	Tags         []string          `json:"tags"`
	Dependencies []string          `json:"dependencies"`
	Properties   map[string]string `json:"properties"`
}

// Plugin defines the interface for all plugins
type Plugin interface {
	// Basic plugin information
	ID() string
	Type() PluginType
	Metadata() PluginMetadata
	Status() PluginStatus

	// Lifecycle methods
	Initialize(ctx context.Context, config map[string]interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Cleanup(ctx context.Context) error

	// Capability methods
	Capabilities() []string
	SupportsCapability(capability string) bool

	// Configuration
	Configure(config map[string]interface{}) error
	GetConfig() map[string]interface{}

	// Error handling
	LastError() error
}

// BasePlugin provides a basic implementation of the Plugin interface
type BasePlugin struct {
	id           string
	pluginType   PluginType
	metadata     PluginMetadata
	status       PluginStatus
	capabilities []string
	config       map[string]interface{}
	lastError    error
	startTime    time.Time
	stopTime     time.Time
	mutex        sync.RWMutex
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(id string, pluginType PluginType, metadata PluginMetadata) *BasePlugin {
	return &BasePlugin{
		id:           id,
		pluginType:   pluginType,
		metadata:     metadata,
		status:       PluginStatusUninitialized,
		capabilities: make([]string, 0),
		config:       make(map[string]interface{}),
	}
}

// ID returns the plugin ID
func (p *BasePlugin) ID() string {
	return p.id
}

// Type returns the plugin type
func (p *BasePlugin) Type() PluginType {
	return p.pluginType
}

// Metadata returns the plugin metadata
func (p *BasePlugin) Metadata() PluginMetadata {
	return p.metadata
}

// Status returns the plugin status
func (p *BasePlugin) Status() PluginStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// setStatus sets the plugin status
func (p *BasePlugin) setStatus(status PluginStatus) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.status = status
}

// Initialize initializes the plugin (base implementation)
func (p *BasePlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Save configuration
	p.config = config

	// Update status
	p.status = PluginStatusInitialized
	return nil
}

// Start starts the plugin (base implementation)
func (p *BasePlugin) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status != PluginStatusInitialized && p.status != PluginStatusStopped {
		return fmt.Errorf("cannot start plugin with status %s", p.status)
	}

	p.startTime = time.Now()
	p.status = PluginStatusStarted
	return nil
}

// Stop stops the plugin (base implementation)
func (p *BasePlugin) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status != PluginStatusStarted {
		return fmt.Errorf("cannot stop plugin with status %s", p.status)
	}

	p.stopTime = time.Now()
	p.status = PluginStatusStopped
	return nil
}

// Cleanup performs cleanup (base implementation)
func (p *BasePlugin) Cleanup(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status != PluginStatusStopped {
		return fmt.Errorf("cannot cleanup plugin with status %s", p.status)
	}

	// Reset configuration
	p.config = make(map[string]interface{})

	return nil
}

// Capabilities returns the plugin capabilities
func (p *BasePlugin) Capabilities() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.capabilities
}

// SupportsCapability checks if the plugin supports a capability
func (p *BasePlugin) SupportsCapability(capability string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, cap := range p.capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// AddCapability adds a capability to the plugin
func (p *BasePlugin) AddCapability(capability string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if capability already exists
	for _, cap := range p.capabilities {
		if cap == capability {
			return
		}
	}

	p.capabilities = append(p.capabilities, capability)
}

// Configure configures the plugin
func (p *BasePlugin) Configure(config map[string]interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Merge with existing configuration
	for key, value := range config {
		p.config[key] = value
	}

	return nil
}

// GetConfig returns the plugin configuration
func (p *BasePlugin) GetConfig() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Return a copy to prevent modification
	configCopy := make(map[string]interface{})
	for key, value := range p.config {
		configCopy[key] = value
	}

	return configCopy
}

// LastError returns the last error
func (p *BasePlugin) LastError() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.lastError
}

// setError sets the last error
func (p *BasePlugin) setError(err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.lastError = err
	if err != nil {
		p.status = PluginStatusError
	}
}

// Logger defines a logging interface for plugins
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// PluginFactory creates plugins
type PluginFactory func(id string, config map[string]interface{}) (Plugin, error)

// PluginOption represents an option for creating a plugin
type PluginOption func(plugin *BasePlugin)

// WithCapabilities adds capabilities to a plugin
func WithCapabilities(capabilities ...string) PluginOption {
	return func(plugin *BasePlugin) {
		for _, capability := range capabilities {
			plugin.AddCapability(capability)
		}
	}
}

// WithMetadata sets metadata for a plugin
func WithMetadata(metadata PluginMetadata) PluginOption {
	return func(plugin *BasePlugin) {
		plugin.metadata = metadata
	}
}

// WithConfig sets configuration for a plugin
func WithConfig(config map[string]interface{}) PluginOption {
	return func(plugin *BasePlugin) {
		for key, value := range config {
			plugin.config[key] = value
		}
	}
}

// NewPlugin creates a new plugin with options
func NewPlugin(id string, pluginType PluginType, options ...PluginOption) *BasePlugin {
	plugin := &BasePlugin{
		id:           id,
		pluginType:   pluginType,
		status:       PluginStatusUninitialized,
		capabilities: make([]string, 0),
		config:       make(map[string]interface{}),
		metadata:     PluginMetadata{},
	}

	// Apply options
	for _, option := range options {
		option(plugin)
	}

	return plugin
}
