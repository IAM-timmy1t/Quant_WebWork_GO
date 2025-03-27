// bridge_manager.go - Bridge Manager implementation for creating and managing Bridge instances

package manager

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/protocols"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	// ErrBridgeNotFound is returned when a bridge is not found
	ErrBridgeNotFound = errors.New("bridge not found")

	// ErrBridgeAlreadyExists is returned when a bridge already exists
	ErrBridgeAlreadyExists = errors.New("bridge already exists")

	// ErrBridgeInitFailed is returned when bridge initialization fails
	ErrBridgeInitFailed = errors.New("bridge initialization failed")

	// ErrBridgeAlreadyRunning is returned when trying to start an already running bridge
	ErrBridgeAlreadyRunning = errors.New("bridge already running")

	// ErrBridgeNotRunning is returned when trying to stop a bridge that is not running
	ErrBridgeNotRunning = errors.New("bridge not running")
)

// BridgeManager manages multiple bridge instances
type BridgeManager struct {
	bridges     map[string]*bridge.Bridge
	bridgesMu   sync.RWMutex
	adapterReg  *adapters.SharedAdapterRegistry
	protocolReg *protocols.ProtocolRegistry
	logger      *zap.SugaredLogger
	config      *BridgeManagerConfig
}

// BridgeManagerConfig defines configuration for the bridge manager
type BridgeManagerConfig struct {
	DefaultBridgeOptions *bridge.BridgeOptions
	LogLevel             string
}

// NewBridgeManager creates a new bridge manager
func NewBridgeManager(config *BridgeManagerConfig, logger *zap.SugaredLogger) *BridgeManager {
	if config == nil {
		config = &BridgeManagerConfig{
			DefaultBridgeOptions: bridge.DefaultBridgeOptions(),
			LogLevel:             "info",
		}
	}

	if logger == nil {
		zapLogger, _ := zap.NewProduction()
		logger = zapLogger.Sugar()
	}

	return &BridgeManager{
		bridges:     make(map[string]*bridge.Bridge),
		adapterReg:  adapters.NewSharedAdapterRegistry(),
		protocolReg: protocols.NewProtocolRegistry(),
		logger:      logger,
		config:      config,
	}
}

// CreateBridge creates a new bridge instance with the given ID and options
func (m *BridgeManager) CreateBridge(ctx context.Context, id string, options *bridge.BridgeOptions) (*bridge.Bridge, error) {
	m.bridgesMu.Lock()
	defer m.bridgesMu.Unlock()

	// If ID is empty, generate a unique ID
	if id == "" {
		id = uuid.New().String()
	}

	// Check if bridge already exists
	if _, exists := m.bridges[id]; exists {
		return nil, ErrBridgeAlreadyExists
	}

	// Use default options if not provided
	if options == nil {
		options = m.config.DefaultBridgeOptions
	}

	// Create bridge logger
	bridgeLogger := &bridgeLoggerAdapter{
		logger: m.logger.With("bridge_id", id),
	}

	// Create new bridge
	b := bridge.NewBridge(options, bridgeLogger)

	// Initialize the bridge
	if err := b.Initialize(ctx); err != nil {
		m.logger.Errorw("Failed to initialize bridge", "bridge_id", id, "error", err)
		return nil, errors.Join(ErrBridgeInitFailed, err)
	}

	// Store the bridge
	m.bridges[id] = b
	m.logger.Infow("Created bridge", "bridge_id", id)

	return b, nil
}

// GetBridge returns a bridge by ID
func (m *BridgeManager) GetBridge(id string) (*bridge.Bridge, error) {
	m.bridgesMu.RLock()
	defer m.bridgesMu.RUnlock()

	bridge, exists := m.bridges[id]
	if !exists {
		return nil, ErrBridgeNotFound
	}

	return bridge, nil
}

// RemoveBridge removes a bridge by ID
func (m *BridgeManager) RemoveBridge(ctx context.Context, id string) error {
	m.bridgesMu.Lock()
	defer m.bridgesMu.Unlock()

	bridge, exists := m.bridges[id]
	if !exists {
		return ErrBridgeNotFound
	}

	// Shutdown the bridge
	if err := bridge.Shutdown(ctx); err != nil {
		m.logger.Warnw("Error shutting down bridge during removal", "bridge_id", id, "error", err)
	}

	// Remove the bridge
	delete(m.bridges, id)
	m.logger.Infow("Removed bridge", "bridge_id", id)

	return nil
}

// ListBridges returns a list of all bridge IDs
func (m *BridgeManager) ListBridges() []string {
	m.bridgesMu.RLock()
	defer m.bridgesMu.RUnlock()

	bridges := make([]string, 0, len(m.bridges))
	for id := range m.bridges {
		bridges = append(bridges, id)
	}

	return bridges
}

// Start starts a bridge by ID
func (m *BridgeManager) Start(ctx context.Context, id string) error {
	bridge, err := m.GetBridge(id)
	if err != nil {
		return err
	}

	// Check if bridge is already running
	status := bridge.Status()
	if status == bridge.StatusReady {
		return ErrBridgeAlreadyRunning
	}

	// Initialize the bridge (this will start all adapters and protocols)
	err = bridge.Initialize(ctx)
	if err != nil {
		m.logger.Errorw("Failed to start bridge", "bridge_id", id, "error", err)
		return err
	}

	m.logger.Infow("Started bridge", "bridge_id", id)
	return nil
}

// Stop stops a bridge by ID
func (m *BridgeManager) Stop(ctx context.Context, id string) error {
	bridge, err := m.GetBridge(id)
	if err != nil {
		return err
	}

	// Check if bridge is already stopped
	status := bridge.Status()
	if status != bridge.StatusReady {
		return ErrBridgeNotRunning
	}

	// Shutdown the bridge
	err = bridge.Shutdown(ctx)
	if err != nil {
		m.logger.Errorw("Failed to stop bridge", "bridge_id", id, "error", err)
		return err
	}

	m.logger.Infow("Stopped bridge", "bridge_id", id)
	return nil
}

// RegisterAdapter registers an adapter factory with the bridge manager
func (m *BridgeManager) RegisterAdapter(adapterType string, factory adapters.SharedAdapterFactory) {
	m.adapterReg.RegisterFactory(adapterType, factory)
	m.logger.Infow("Registered adapter factory", "type", adapterType)
}

// RegisterProtocol registers a protocol factory with the bridge manager
func (m *BridgeManager) RegisterProtocol(protocolType string, factory protocols.ProtocolFactory) {
	m.protocolReg.RegisterFactory(protocolType, factory)
	m.logger.Infow("Registered protocol factory", "type", protocolType)
}

// GetAdapterRegistry returns the adapter registry
func (m *BridgeManager) GetAdapterRegistry() *adapters.SharedAdapterRegistry {
	return m.adapterReg
}

// GetProtocolRegistry returns the protocol registry
func (m *BridgeManager) GetProtocolRegistry() *protocols.ProtocolRegistry {
	return m.protocolReg
}

// bridgeLoggerAdapter adapts zap.SugaredLogger to bridge.BridgeLogger
type bridgeLoggerAdapter struct {
	logger *zap.SugaredLogger
}

func (a *bridgeLoggerAdapter) Debug(msg string, fields map[string]interface{}) {
	a.logger.Debugw(msg, toSlice(fields)...)
}

func (a *bridgeLoggerAdapter) Info(msg string, fields map[string]interface{}) {
	a.logger.Infow(msg, toSlice(fields)...)
}

func (a *bridgeLoggerAdapter) Warn(msg string, fields map[string]interface{}) {
	a.logger.Warnw(msg, toSlice(fields)...)
}

func (a *bridgeLoggerAdapter) Error(msg string, fields map[string]interface{}) {
	a.logger.Errorw(msg, toSlice(fields)...)
}

// toSlice converts a map to a slice of alternating key, value pairs for zap
func toSlice(m map[string]interface{}) []interface{} {
	if m == nil {
		return nil
	}
	
	slice := make([]interface{}, 0, len(m)*2)
	for k, v := range m {
		slice = append(slice, k, v)
	}
	return slice
}
