// manager.go - Configuration management system

package config

import (
	"fmt"
	"sync"
	"time"
)

// Manager manages configuration from multiple providers
type Manager struct {
	providers      map[string]Provider
	mutex          sync.RWMutex
	changeHandlers map[string][]func(ChangeEvent)
	defaultProvider string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		providers:      make(map[string]Provider),
		changeHandlers: make(map[string][]func(ChangeEvent)),
	}
}

// RegisterProvider registers a configuration provider
func (m *Manager) RegisterProvider(name string, provider Provider) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	m.providers[name] = provider
	
	// Set as default if it's the first provider
	if len(m.providers) == 1 {
		m.defaultProvider = name
	}
	
	return nil
}

// SetDefaultProvider sets the default configuration provider
func (m *Manager) SetDefaultProvider(name string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if _, exists := m.providers[name]; !exists {
		return ErrProviderNotFound
	}

	m.defaultProvider = name
	return nil
}

// GetProvider retrieves a provider by name
func (m *Manager) GetProvider(name string) (Provider, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		return nil, ErrProviderNotFound
	}

	return provider, nil
}

// GetDefaultProvider retrieves the default provider
func (m *Manager) GetDefaultProvider() (Provider, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.defaultProvider == "" {
		return nil, ErrProviderNotFound
	}

	return m.providers[m.defaultProvider], nil
}

// LoadAll loads configuration from all providers
func (m *Manager) LoadAll() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for name, provider := range m.providers {
		if err := provider.Load(); err != nil {
			return fmt.Errorf("failed to load from provider %s: %w", name, err)
		}
	}

	return nil
}

// Get retrieves a configuration value by key from the default provider
func (m *Manager) Get(key string) (interface{}, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return nil, err
	}

	value, found := provider.Get(key)
	if !found {
		return nil, ErrKeyNotFound
	}

	return value, nil
}

// GetString retrieves a string configuration value
func (m *Manager) GetString(key string) (string, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return "", err
	}

	value, found := provider.GetString(key)
	if !found {
		return "", ErrKeyNotFound
	}

	return value, nil
}

// GetInt retrieves an integer configuration value
func (m *Manager) GetInt(key string) (int, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return 0, err
	}

	value, found := provider.GetInt(key)
	if !found {
		return 0, ErrKeyNotFound
	}

	return value, nil
}

// GetBool retrieves a boolean configuration value
func (m *Manager) GetBool(key string) (bool, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return false, err
	}

	value, found := provider.GetBool(key)
	if !found {
		return false, ErrKeyNotFound
	}

	return value, nil
}

// GetFloat retrieves a float configuration value
func (m *Manager) GetFloat(key string) (float64, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return 0, err
	}

	value, found := provider.GetFloat(key)
	if !found {
		return 0, ErrKeyNotFound
	}

	return value, nil
}

// GetDuration retrieves a duration configuration value
func (m *Manager) GetDuration(key string) (time.Duration, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return 0, err
	}

	value, found := provider.GetDuration(key)
	if !found {
		return 0, ErrKeyNotFound
	}

	return value, nil
}

// GetFrom retrieves a configuration value from a specific provider
func (m *Manager) GetFrom(providerName, key string) (interface{}, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	value, found := provider.Get(key)
	if !found {
		return nil, ErrKeyNotFound
	}

	return value, nil
}

// Set sets a configuration value in the default provider
func (m *Manager) Set(key string, value interface{}) error {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return err
	}

	oldValue, _ := provider.Get(key)
	
	if err := provider.Set(key, value); err != nil {
		return err
	}
	
	m.notifyChangeHandlers(key, oldValue, value)
	return nil
}

// SetIn sets a configuration value in a specific provider
func (m *Manager) SetIn(providerName, key string, value interface{}) error {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return err
	}

	oldValue, _ := provider.Get(key)
	
	if err := provider.Set(key, value); err != nil {
		return err
	}
	
	m.notifyChangeHandlers(key, oldValue, value)
	return nil
}

// Has checks if a configuration key exists in the default provider
func (m *Manager) Has(key string) bool {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return false
	}

	return provider.Has(key)
}

// HasIn checks if a configuration key exists in a specific provider
func (m *Manager) HasIn(providerName, key string) bool {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return false
	}

	return provider.Has(key)
}

// WatchKey watches a configuration key for changes in the default provider
func (m *Manager) WatchKey(key string, handler func(ChangeEvent)) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.changeHandlers[key] = append(m.changeHandlers[key], handler)
	
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return err
	}
	
	_, err = provider.WatchKey(key)
	return err
}

// notifyChangeHandlers notifies all registered handlers about a configuration change
func (m *Manager) notifyChangeHandlers(key string, oldValue, newValue interface{}) {
	m.mutex.RLock()
	handlers := m.changeHandlers[key]
	m.mutex.RUnlock()

	event := ChangeEvent{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  newValue,
		Timestamp: time.Now(),
	}

	for _, handler := range handlers {
		go handler(event)
	}
}

// ValidateAgainstSchema validates configuration against a schema
func (m *Manager) ValidateAgainstSchema(schema ConfigurationSchema) []error {
	var errors []error
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return append(errors, err)
	}

	// Check all required options are present and valid
	for _, option := range schema.Options {
		if option.Required && !provider.Has(option.Key) {
			errors = append(errors, fmt.Errorf("required configuration key %s is missing", option.Key))
			continue
		}

		if provider.Has(option.Key) {
			value, _ := provider.Get(option.Key)
			if option.ValidationFunc != nil {
				if err := option.ValidationFunc(value); err != nil {
					errors = append(errors, fmt.Errorf("validation failed for key %s: %w", option.Key, err))
				}
			}
		}
	}

	// Run schema-level validators
	configMap := make(map[string]interface{})
	for _, option := range schema.Options {
		if provider.Has(option.Key) {
			value, _ := provider.Get(option.Key)
			configMap[option.Key] = value
		} else if option.DefaultValue != nil {
			configMap[option.Key] = option.DefaultValue
		}
	}

	for _, validator := range schema.Validators {
		if err := validator(configMap); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
