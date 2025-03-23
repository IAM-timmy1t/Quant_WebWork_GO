// types.go - Configuration system type definitions

package config

import (
	"errors"
	"sync"
	"time"
)

// Provider defines the interface for configuration providers
type Provider interface {
	// Load loads configuration from the provider
	Load() error

	// Get retrieves a configuration value by key
	Get(key string) (interface{}, bool)

	// GetString retrieves a string configuration value
	GetString(key string) (string, bool)

	// GetInt retrieves an integer configuration value
	GetInt(key string) (int, bool)

	// GetBool retrieves a boolean configuration value
	GetBool(key string) (bool, bool)

	// GetFloat retrieves a float configuration value
	GetFloat(key string) (float64, bool)

	// GetDuration retrieves a duration configuration value
	GetDuration(key string) (time.Duration, bool)

	// GetStringMap retrieves a map of string configuration values
	GetStringMap(key string) (map[string]string, bool)

	// GetStringSlice retrieves a slice of string configuration values
	GetStringSlice(key string) ([]string, bool)

	// Set sets a configuration value
	Set(key string, value interface{}) error

	// Has checks if a configuration key exists
	Has(key string) bool

	// WatchKey watches a configuration key for changes
	WatchKey(key string) (<-chan interface{}, error)
}

// ChangeEvent represents a configuration change event
type ChangeEvent struct {
	// Key is the configuration key that changed
	Key string

	// OldValue is the previous value
	OldValue interface{}

	// NewValue is the new value
	NewValue interface{}

	// Timestamp is when the change occurred
	Timestamp time.Time
}

// ProviderType defines the type of configuration provider
type ProviderType string

const (
	// File provider loads configuration from a file
	File ProviderType = "file"

	// Environment provider loads configuration from environment variables
	Environment ProviderType = "environment"

	// Remote provider loads configuration from a remote source
	Remote ProviderType = "remote"

	// Memory provider stores configuration in memory
	Memory ProviderType = "memory"
)

// Common errors
var (
	ErrKeyNotFound      = errors.New("configuration key not found")
	ErrTypeConversion   = errors.New("configuration value type conversion failed")
	ErrProviderNotFound = errors.New("configuration provider not found")
	ErrLoadFailed       = errors.New("failed to load configuration")
	ErrWatchFailed      = errors.New("failed to watch configuration key")
)

// ConfigOption represents a configuration option
type ConfigOption struct {
	// Key is the configuration key
	Key string

	// DefaultValue is the default value if not specified
	DefaultValue interface{}

	// Required indicates if the configuration is required
	Required bool

	// Description describes the configuration option
	Description string

	// Type defines the expected data type
	Type string

	// ValidationFunc validates the configuration value
	ValidationFunc func(interface{}) error
}

// ConfigurationSchema defines a schema for configuration validation
type ConfigurationSchema struct {
	// Name of the schema
	Name string

	// Version of the schema
	Version string

	// Options defines the configuration options
	Options []ConfigOption

	// Validators are functions that validate the entire configuration
	Validators []func(map[string]interface{}) error
}
