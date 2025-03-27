// factory.go - Bridge adapter factory implementations

package adapters

import (
	"context"
	"fmt"
	"sync"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/logging"
	"go.uber.org/zap"
)

// AdapterFactory creates a new adapter
type AdapterFactory func(config AdapterConfig) (Adapter, error)

// AdapterRegistry manages adapter factories
type AdapterRegistry struct {
	factories map[string]AdapterFactory
	mutex     sync.RWMutex
	logger    *zap.SugaredLogger
}

// NewAdapterRegistry creates a new adapter registry
func NewAdapterRegistry(logger *zap.SugaredLogger) *AdapterRegistry {
	if logger == nil {
		zapLogger, _ := zap.NewProduction()
		logger = zapLogger.Sugar()
	}

	return &AdapterRegistry{
		factories: make(map[string]AdapterFactory),
		logger:    logger,
	}
}

// RegisterFactory registers an adapter factory
func (r *AdapterRegistry) RegisterFactory(adapterType string, factory AdapterFactory) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.factories[adapterType]; exists {
		return fmt.Errorf("adapter factory for type '%s' already registered", adapterType)
	}

	r.factories[adapterType] = factory
	r.logger.Infow("Registered adapter factory", "adapter_type", adapterType)
	return nil
}

// GetFactory returns an adapter factory for the specified type
func (r *AdapterRegistry) GetFactory(adapterType string) (AdapterFactory, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	factory, exists := r.factories[adapterType]
	if !exists {
		return nil, fmt.Errorf("no adapter factory registered for type '%s'", adapterType)
	}

	return factory, nil
}

// CreateAdapter creates a new adapter instance
func (r *AdapterRegistry) CreateAdapter(config AdapterConfig) (Adapter, error) {
	factory, err := r.GetFactory(config.Type)
	if err != nil {
		return nil, err
	}

	adapter, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter of type '%s': %w", config.Type, err)
	}

	r.logger.Debugw("Created adapter instance", "adapter_type", config.Type, "adapter_name", config.Name)
	return adapter, nil
}

// ListRegisteredTypes returns a list of all registered adapter types
func (r *AdapterRegistry) ListRegisteredTypes() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}

	return types
}

// UnregisterFactory removes a factory from the registry
func (r *AdapterRegistry) UnregisterFactory(adapterType string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.factories[adapterType]; !exists {
		return fmt.Errorf("no adapter factory registered for type '%s'", adapterType)
	}

	delete(r.factories, adapterType)
	r.logger.Infow("Unregistered adapter factory", "adapter_type", adapterType)
	return nil
}

// Global adapter registry
var (
	globalRegistry     *AdapterRegistry
	globalRegistryOnce sync.Once
)

// GetGlobalAdapterRegistry returns the global adapter registry
func GetGlobalAdapterRegistry() *AdapterRegistry {
	globalRegistryOnce.Do(func() {
		logger, _ := logging.GetLogger("bridge.adapters")
		globalRegistry = NewAdapterRegistry(logger)
	})
	return globalRegistry
}

// RegisterAdapterFactory registers a factory with the global registry
func RegisterAdapterFactory(adapterType string, factory AdapterFactory) error {
	return GetGlobalAdapterRegistry().RegisterFactory(adapterType, factory)
}

// CreateAdapterInstance creates a new adapter using the global registry
func CreateAdapterInstance(config AdapterConfig) (Adapter, error) {
	return GetGlobalAdapterRegistry().CreateAdapter(config)
}

// ListAvailableAdapterTypes returns all registered adapter types from the global registry
func ListAvailableAdapterTypes() []string {
	return GetGlobalAdapterRegistry().ListRegisteredTypes()
}

// CreateAdapterWithContext creates a new adapter and initializes it with the provided context
func CreateAdapterWithContext(ctx context.Context, config AdapterConfig) (Adapter, error) {
	adapter, err := CreateAdapterInstance(config)
	if err != nil {
		return nil, err
	}

	if err := adapter.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize adapter: %w", err)
	}

	return adapter, nil
}

// ValidateAdapterConfig validates an adapter configuration
func ValidateAdapterConfig(config AdapterConfig) error {
	if config.Name == "" {
		return fmt.Errorf("adapter name cannot be empty")
	}

	if config.Type == "" {
		return fmt.Errorf("adapter type cannot be empty")
	}

	// Check if the adapter type is registered
	registry := GetGlobalAdapterRegistry()
	if _, err := registry.GetFactory(config.Type); err != nil {
		return err
	}

	return nil
}
