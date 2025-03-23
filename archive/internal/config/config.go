package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// Config represents the configuration manager
type Config struct {
	settings map[string]Setting
	file     string
	mu       sync.RWMutex
}

// Setting represents a configuration setting with metadata
type Setting struct {
	Value       interface{} `json:"value" yaml:"value"`
	Description string     `json:"description" yaml:"description"`
	Type        string     `json:"type" yaml:"type"`
	Required    bool       `json:"required" yaml:"required"`
}

// ValidationIssue represents a configuration validation issue
type ValidationIssue struct {
	Key      string
	Severity string
	Message  string
}

// New creates a new configuration manager
func New(file string) (*Config, error) {
	c := &Config{
		settings: make(map[string]Setting),
		file:     file,
	}

	// Create config directory if it doesn't exist
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	// Load existing config if it exists
	if _, err := os.Stat(file); err == nil {
		if err := c.Load(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Get retrieves a configuration value
func (c *Config) Get(key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	setting, ok := c.settings[key]
	if !ok {
		return nil, fmt.Errorf("configuration key not found: %s", key)
	}

	return setting.Value, nil
}

// Set updates a configuration value
func (c *Config) Set(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get existing setting or create new one
	setting, ok := c.settings[key]
	if !ok {
		setting = Setting{
			Type: fmt.Sprintf("%T", value),
		}
	}

	setting.Value = value
	c.settings[key] = setting

	return nil
}

// List returns all configuration settings, optionally filtered by section
func (c *Config) List(section string) (map[string]Setting, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]Setting)
	for key, setting := range c.settings {
		if section == "" || strings.HasPrefix(key, section+".") {
			result[key] = setting
		}
	}

	return result, nil
}

// GetAll returns all configuration settings
func (c *Config) GetAll() map[string]Setting {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy to prevent external modification
	result := make(map[string]Setting, len(c.settings))
	for k, v := range c.settings {
		result[k] = v
	}

	return result
}

// Load loads configuration from file
func (c *Config) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.file)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var settings map[string]Setting
	if strings.HasSuffix(c.file, ".json") {
		err = json.Unmarshal(data, &settings)
	} else {
		err = yaml.Unmarshal(data, &settings)
	}

	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	c.settings = settings
	return nil
}

// Save saves configuration to file
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var data []byte
	var err error

	if strings.HasSuffix(c.file, ".json") {
		data, err = json.MarshalIndent(c.settings, "", "  ")
	} else {
		data, err = yaml.Marshal(c.settings)
	}

	if err != nil {
		return fmt.Errorf("failed to format config: %v", err)
	}

	if err := os.WriteFile(c.file, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Import imports configuration from a map
func (c *Config) Import(config map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Convert flat map to settings
	for key, value := range config {
		setting := Setting{
			Value: value,
			Type:  fmt.Sprintf("%T", value),
		}
		c.settings[key] = setting
	}

	return nil
}

// Validate validates the current configuration
func (c *Config) Validate() ([]ValidationIssue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var issues []ValidationIssue

	// Check required settings
	for key, setting := range c.settings {
		if setting.Required && setting.Value == nil {
			issues = append(issues, ValidationIssue{
				Key:      key,
				Severity: "error",
				Message:  "Required setting is not set",
			})
		}
	}

	// Add type validation
	for key, setting := range c.settings {
		if setting.Value != nil && setting.Type != "" {
			if actualType := fmt.Sprintf("%T", setting.Value); actualType != setting.Type {
				issues = append(issues, ValidationIssue{
					Key:      key,
					Severity: "warning",
					Message:  fmt.Sprintf("Type mismatch: expected %s, got %s", setting.Type, actualType),
				})
			}
		}
	}

	return issues, nil
}

// ParseInt parses a string into an int64
func ParseInt(value string) (int64, error) {
	// Try base 10 first
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i, nil
	}

	// Try parsing hex
	if strings.HasPrefix(strings.ToLower(value), "0x") {
		if i, err := strconv.ParseInt(value[2:], 16, 64); err == nil {
			return i, nil
		}
	}

	return 0, fmt.Errorf("invalid integer value: %s", value)
}
