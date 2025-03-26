// registry.go - Plugin registry for bridge components

package plugins

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages the registration and discovery of plugins
type Registry struct {
	plugins      map[string]Plugin
	factories    map[string]PluginFactory
	dependencies map[string][]string
	mutex        sync.RWMutex
	logger       Logger
}

// NewRegistry creates a new plugin registry
func NewRegistry(logger Logger) *Registry {
	return &Registry{
		plugins:      make(map[string]Plugin),
		factories:    make(map[string]PluginFactory),
		dependencies: make(map[string][]string),
		logger:       logger,
	}
}

// RegisterPlugin registers a plugin with the registry
func (r *Registry) RegisterPlugin(plugin Plugin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := plugin.ID()
	if _, exists := r.plugins[id]; exists {
		return fmt.Errorf("plugin '%s' already registered", id)
	}

	r.plugins[id] = plugin
	r.logger.Info(fmt.Sprintf("Registered plugin: %s", id), map[string]interface{}{
		"plugin_id":   id,
		"plugin_type": string(plugin.Type()),
	})

	return nil
}

// UnregisterPlugin unregisters a plugin from the registry
func (r *Registry) UnregisterPlugin(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.plugins[id]; !exists {
		return ErrPluginNotFound
	}

	// Check if other plugins depend on this one
	for pid, deps := range r.dependencies {
		for _, dep := range deps {
			if dep == id {
				return fmt.Errorf("cannot unregister plugin '%s': plugin '%s' depends on it", id, pid)
			}
		}
	}

	delete(r.plugins, id)
	r.logger.Info(fmt.Sprintf("Unregistered plugin: %s", id), map[string]interface{}{
		"plugin_id": id,
	})

	return nil
}

// GetPlugin retrieves a plugin by ID
func (r *Registry) GetPlugin(id string) (Plugin, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugin, exists := r.plugins[id]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return plugin, nil
}

// RegisterFactory registers a plugin factory
func (r *Registry) RegisterFactory(id string, factory PluginFactory) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.factories[id]; exists {
		return fmt.Errorf("factory '%s' already registered", id)
	}

	r.factories[id] = factory
	r.logger.Info(fmt.Sprintf("Registered plugin factory: %s", id), map[string]interface{}{
		"factory_id": id,
	})

	return nil
}

// CreatePlugin creates a plugin using a registered factory
func (r *Registry) CreatePlugin(factoryID string, pluginID string, config map[string]interface{}) (Plugin, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if plugin already exists
	if _, exists := r.plugins[pluginID]; exists {
		return nil, fmt.Errorf("plugin '%s' already exists", pluginID)
	}

	// Get factory
	factory, exists := r.factories[factoryID]
	if !exists {
		return nil, fmt.Errorf("factory '%s' not found", factoryID)
	}

	// Create plugin
	plugin, err := factory(pluginID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin '%s': %w", pluginID, err)
	}

	// Register plugin
	r.plugins[pluginID] = plugin

	r.logger.Info(fmt.Sprintf("Created plugin '%s' using factory '%s'", pluginID, factoryID), map[string]interface{}{
		"plugin_id":  pluginID,
		"factory_id": factoryID,
	})

	return plugin, nil
}

// ListPlugins lists all registered plugins
func (r *Registry) ListPlugins() []Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// ListPluginsByType lists plugins of a specific type
func (r *Registry) ListPluginsByType(pluginType PluginType) []Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugins := make([]Plugin, 0)
	for _, plugin := range r.plugins {
		if plugin.Type() == pluginType {
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

// ListPluginsByCapability lists plugins that support a specific capability
func (r *Registry) ListPluginsByCapability(capability string) []Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugins := make([]Plugin, 0)
	for _, plugin := range r.plugins {
		if plugin.SupportsCapability(capability) {
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

// InitializePlugin initializes a plugin
func (r *Registry) InitializePlugin(id string, ctx context.Context, config map[string]interface{}) error {
	r.mutex.Lock()
	plugin, exists := r.plugins[id]
	r.mutex.Unlock()

	if !exists {
		return ErrPluginNotFound
	}

	// Initialize the plugin
	if err := plugin.Initialize(ctx, config); err != nil {
		r.logger.Error(fmt.Sprintf("Failed to initialize plugin '%s': %v", id, err), map[string]interface{}{
			"plugin_id": id,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to initialize plugin '%s': %w", id, err)
	}

	r.logger.Info(fmt.Sprintf("Initialized plugin: %s", id), map[string]interface{}{
		"plugin_id": id,
	})

	return nil
}

// StartPlugin starts a plugin
func (r *Registry) StartPlugin(id string, ctx context.Context) error {
	r.mutex.Lock()
	plugin, exists := r.plugins[id]
	r.mutex.Unlock()

	if !exists {
		return ErrPluginNotFound
	}

	// Check plugin dependencies
	if err := r.checkDependencies(id); err != nil {
		return err
	}

	// Start the plugin
	if err := plugin.Start(ctx); err != nil {
		r.logger.Error(fmt.Sprintf("Failed to start plugin '%s': %v", id, err), map[string]interface{}{
			"plugin_id": id,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to start plugin '%s': %w", id, err)
	}

	r.logger.Info(fmt.Sprintf("Started plugin: %s", id), map[string]interface{}{
		"plugin_id": id,
	})

	return nil
}

// StopPlugin stops a plugin
func (r *Registry) StopPlugin(id string, ctx context.Context) error {
	r.mutex.Lock()
	plugin, exists := r.plugins[id]
	r.mutex.Unlock()

	if !exists {
		return ErrPluginNotFound
	}

	// Check reverse dependencies
	if err := r.checkReverseDependencies(id); err != nil {
		return err
	}

	// Stop the plugin
	if err := plugin.Stop(ctx); err != nil {
		r.logger.Error(fmt.Sprintf("Failed to stop plugin '%s': %v", id, err), map[string]interface{}{
			"plugin_id": id,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to stop plugin '%s': %w", id, err)
	}

	r.logger.Info(fmt.Sprintf("Stopped plugin: %s", id), map[string]interface{}{
		"plugin_id": id,
	})

	return nil
}

// CleanupPlugin performs cleanup for a plugin
func (r *Registry) CleanupPlugin(id string, ctx context.Context) error {
	r.mutex.Lock()
	plugin, exists := r.plugins[id]
	r.mutex.Unlock()

	if !exists {
		return ErrPluginNotFound
	}

	// Cleanup the plugin
	if err := plugin.Cleanup(ctx); err != nil {
		r.logger.Error(fmt.Sprintf("Failed to cleanup plugin '%s': %v", id, err), map[string]interface{}{
			"plugin_id": id,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to cleanup plugin '%s': %w", id, err)
	}

	r.logger.Info(fmt.Sprintf("Cleaned up plugin: %s", id), map[string]interface{}{
		"plugin_id": id,
	})

	return nil
}

// AddDependency adds a dependency between plugins
func (r *Registry) AddDependency(pluginID, dependencyID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if both plugins exist
	if _, exists := r.plugins[pluginID]; !exists {
		return fmt.Errorf("plugin '%s' not found", pluginID)
	}

	if _, exists := r.plugins[dependencyID]; !exists {
		return fmt.Errorf("dependency plugin '%s' not found", dependencyID)
	}

	// Initialize dependency list if needed
	if _, exists := r.dependencies[pluginID]; !exists {
		r.dependencies[pluginID] = make([]string, 0)
	}

	// Check if dependency already exists
	for _, dep := range r.dependencies[pluginID] {
		if dep == dependencyID {
			return nil // Dependency already exists
		}
	}

	// Add dependency
	r.dependencies[pluginID] = append(r.dependencies[pluginID], dependencyID)

	r.logger.Info(fmt.Sprintf("Added dependency: %s -> %s", pluginID, dependencyID), map[string]interface{}{
		"plugin_id":     pluginID,
		"dependency_id": dependencyID,
	})

	return nil
}

// RemoveDependency removes a dependency between plugins
func (r *Registry) RemoveDependency(pluginID, dependencyID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if dependency list exists
	deps, exists := r.dependencies[pluginID]
	if !exists {
		return nil // No dependencies to remove
	}

	// Find and remove dependency
	for i, dep := range deps {
		if dep == dependencyID {
			// Remove dependency by swapping with last element and truncating
			r.dependencies[pluginID][i] = r.dependencies[pluginID][len(deps)-1]
			r.dependencies[pluginID] = r.dependencies[pluginID][:len(deps)-1]

			r.logger.Info(fmt.Sprintf("Removed dependency: %s -> %s", pluginID, dependencyID), map[string]interface{}{
				"plugin_id":     pluginID,
				"dependency_id": dependencyID,
			})

			return nil
		}
	}

	// Dependency not found
	return nil
}

// GetDependencies gets dependencies for a plugin
func (r *Registry) GetDependencies(pluginID string) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, exists := r.plugins[pluginID]; !exists {
		return nil, fmt.Errorf("plugin '%s' not found", pluginID)
	}

	deps, exists := r.dependencies[pluginID]
	if !exists {
		return make([]string, 0), nil // No dependencies
	}

	// Return a copy to prevent modification
	depsCopy := make([]string, len(deps))
	copy(depsCopy, deps)

	return depsCopy, nil
}

// GetReverseDependencies gets plugins that depend on a plugin
func (r *Registry) GetReverseDependencies(pluginID string) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, exists := r.plugins[pluginID]; !exists {
		return nil, fmt.Errorf("plugin '%s' not found", pluginID)
	}

	revDeps := make([]string, 0)
	for pid, deps := range r.dependencies {
		for _, dep := range deps {
			if dep == pluginID {
				revDeps = append(revDeps, pid)
				break
			}
		}
	}

	return revDeps, nil
}

// checkDependencies checks if all dependencies of a plugin are available and started
func (r *Registry) checkDependencies(pluginID string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deps, exists := r.dependencies[pluginID]
	if !exists || len(deps) == 0 {
		return nil // No dependencies
	}

	for _, depID := range deps {
		dep, exists := r.plugins[depID]
		if !exists {
			return fmt.Errorf("dependency '%s' for plugin '%s' not found", depID, pluginID)
		}

		if dep.Status() != PluginStatusStarted {
			return fmt.Errorf("dependency '%s' for plugin '%s' is not started", depID, pluginID)
		}
	}

	return nil
}

// checkReverseDependencies checks if any plugins depend on the given plugin
func (r *Registry) checkReverseDependencies(pluginID string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for pid, deps := range r.dependencies {
		plugin, exists := r.plugins[pid]
		if !exists {
			continue // Plugin not found, skip
		}

		if plugin.Status() != PluginStatusStarted {
			continue // Plugin not started, skip
		}

		for _, dep := range deps {
			if dep == pluginID {
				return fmt.Errorf("cannot stop plugin '%s': plugin '%s' depends on it", pluginID, pid)
			}
		}
	}

	return nil
}
