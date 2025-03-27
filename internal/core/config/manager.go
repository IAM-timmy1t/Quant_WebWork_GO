// manager.go - Configuration management system

package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Security   SecurityConfig   `mapstructure:"security"`
	Bridge     BridgeConfig     `mapstructure:"bridge"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// ServerConfig represents server-specific configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Timeout         time.Duration `mapstructure:"timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// SecurityConfig represents security-specific configuration
type SecurityConfig struct {
	Level          string          `mapstructure:"level"`
	AuthRequired   bool            `mapstructure:"authRequired"`
	RateLimiting   RateLimitConfig `mapstructure:"rateLimiting"`
	IPMasking      IPMaskingConfig `mapstructure:"ipMasking"`
	EnableFirewall bool            `mapstructure:"enable_firewall"`
	AllowedOrigins []string        `mapstructure:"allowed_origins"`
	TrustedProxies []string        `mapstructure:"trusted_proxies"`
	MaxRequestSize int64           `mapstructure:"max_request_size"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	DefaultLimit int  `mapstructure:"defaultLimit"`
}

// IPMaskingConfig represents IP masking configuration
type IPMaskingConfig struct {
	Enabled             bool          `mapstructure:"enabled"`
	RotationInterval    time.Duration `mapstructure:"rotationInterval"`
	PreserveGeolocation bool          `mapstructure:"preserveGeolocation"`
	DNSPrivacyEnabled   bool          `mapstructure:"dnsPrivacyEnabled"`
}

// BridgeConfig represents bridge-specific configuration
type BridgeConfig struct {
	Protocols []string        `mapstructure:"protocols"`
	Discovery DiscoveryConfig `mapstructure:"discovery"`
}

// DiscoveryConfig represents service discovery configuration
type DiscoveryConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	RefreshInterval time.Duration `mapstructure:"refreshInterval"`
}

// MonitoringConfig represents monitoring-specific configuration
type MonitoringConfig struct {
	Metrics    MetricsConfig    `mapstructure:"metrics"`
	Dashboards DashboardsConfig `mapstructure:"dashboards"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	Interval      time.Duration `mapstructure:"interval"`
	CollectDetail bool          `mapstructure:"collect_detail"`
	FlushInterval time.Duration `mapstructure:"flush_interval"`
}

// DashboardsConfig represents dashboard configuration
type DashboardsConfig struct {
	AutoProvision bool `mapstructure:"autoProvision"`
}

// LoggingConfig contains logging-related configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Environment variables override file configuration
	v.AutomaticEnv()
	v.SetEnvPrefix("QUANT")

	// Set configuration file
	v.SetConfigFile(configPath)

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.timeout", 30*time.Second)
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)

	// Security defaults
	v.SetDefault("security.level", "medium")
	v.SetDefault("security.authRequired", false)
	v.SetDefault("security.enable_firewall", true)
	v.SetDefault("security.allowed_origins", []string{"*"})
	v.SetDefault("security.trusted_proxies", []string{"127.0.0.1"})
	v.SetDefault("security.max_request_size", 10*1024*1024) // 10MB
	v.SetDefault("security.rateLimiting.enabled", true)
	v.SetDefault("security.rateLimiting.defaultLimit", 100)
	v.SetDefault("security.ipMasking.enabled", false)
	v.SetDefault("security.ipMasking.rotationInterval", "1h")
	v.SetDefault("security.ipMasking.preserveGeolocation", true)
	v.SetDefault("security.ipMasking.dnsPrivacyEnabled", true)

	// Bridge defaults
	v.SetDefault("bridge.protocols", []string{"grpc", "rest", "websocket"})
	v.SetDefault("bridge.discovery.enabled", true)
	v.SetDefault("bridge.discovery.refreshInterval", "30s")

	// Monitoring defaults
	v.SetDefault("monitoring.metrics.enabled", true)
	v.SetDefault("monitoring.metrics.interval", "15s")
	v.SetDefault("monitoring.metrics.collect_detail", false)
	v.SetDefault("monitoring.metrics.flush_interval", 15*time.Second)
	v.SetDefault("monitoring.dashboards.autoProvision", true)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output_path", "stdout")
}

// Manager manages configuration from multiple providers
type Manager struct {
	providers       map[string]Provider
	mutex           sync.RWMutex
	changeHandlers  map[string][]func(ChangeEvent)
	defaultProvider string
}

// ChangeEvent represents a configuration change event
type ChangeEvent struct {
	Key      string
	OldValue interface{}
	NewValue interface{}
}

// Provider defines a configuration provider interface
type Provider interface {
	Load() error
	Get(key string) (interface{}, bool)
	GetString(key string) (string, bool)
	GetInt(key string) (int, bool)
	GetBool(key string) (bool, bool)
	GetFloat(key string) (float64, bool)
	GetDuration(key string) (time.Duration, bool)
	Set(key string, value interface{}) error
	Has(key string) bool
}

// Errors
var (
	ErrProviderNotFound = fmt.Errorf("provider not found")
	ErrKeyNotFound      = fmt.Errorf("configuration key not found")
	ErrInvalidValueType = fmt.Errorf("invalid value type")
	ErrReadOnlyProvider = fmt.Errorf("provider is read-only")
)

// ConfigurationSchema defines validation schema for configuration
type ConfigurationSchema struct {
	Required []string
	Schema   map[string]SchemaType
}

// SchemaType defines allowed types for schema validation
type SchemaType string

const (
	TypeString   SchemaType = "string"
	TypeInt      SchemaType = "int"
	TypeBool     SchemaType = "bool"
	TypeFloat    SchemaType = "float"
	TypeDuration SchemaType = "duration"
	TypeList     SchemaType = "list"
	TypeMap      SchemaType = "map"
)

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
	// This implementation will be added in a future update
	return fmt.Errorf("not implemented yet")
}

// notifyChangeHandlers notifies all registered handlers about a configuration change
func (m *Manager) notifyChangeHandlers(key string, oldValue, newValue interface{}) {
	m.mutex.RLock()
	handlers := m.changeHandlers[key]
	m.mutex.RUnlock()

	event := ChangeEvent{
		Key:      key,
		OldValue: oldValue,
		NewValue: newValue,
	}

	for _, handler := range handlers {
		go handler(event)
	}
}

// ValidateConfig performs basic validation on the configuration
func (m *Manager) ValidateConfig() error {
	// This implementation will be added in a future update
	return fmt.Errorf("not implemented yet")
}
