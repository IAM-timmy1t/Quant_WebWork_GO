// file_provider.go - File-based configuration provider

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FileType defines the type of configuration file
type FileType string

const (
	// JSON configuration file
	JSON FileType = "json"
	// YAML configuration file
	YAML FileType = "yaml"
	// TOML configuration file
	TOML FileType = "toml"
	// INI configuration file
	INI FileType = "ini"
)

// FileProvider implements a file-based configuration provider
type FileProvider struct {
	filePath     string
	fileType     FileType
	configData   map[string]interface{}
	mutex        sync.RWMutex
	watchersMu   sync.Mutex
	watchers     map[string][]chan interface{}
	autoReload   bool
	reloadSignal chan struct{}
}

// NewFileProvider creates a new file-based configuration provider
func NewFileProvider(filePath string, autoReload bool) (*FileProvider, error) {
	fileType := getFileTypeFromExtension(filePath)
	if fileType == "" {
		return nil, fmt.Errorf("unsupported file type for %s", filePath)
	}

	provider := &FileProvider{
		filePath:     filePath,
		fileType:     fileType,
		configData:   make(map[string]interface{}),
		watchers:     make(map[string][]chan interface{}),
		autoReload:   autoReload,
		reloadSignal: make(chan struct{}),
	}

	if autoReload {
		go provider.watchFile()
	}

	return provider, nil
}

// getFileTypeFromExtension determines the file type from its extension
func getFileTypeFromExtension(filePath string) FileType {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return JSON
	case ".yaml", ".yml":
		return YAML
	case ".toml":
		return TOML
	case ".ini":
		return INI
	default:
		return ""
	}
}

// Load loads configuration from the file
func (p *FileProvider) Load() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	data, err := os.ReadFile(p.filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	switch p.fileType {
	case JSON:
		return p.parseJSON(data)
	case YAML:
		return fmt.Errorf("YAML support not implemented")
	case TOML:
		return fmt.Errorf("TOML support not implemented")
	case INI:
		return fmt.Errorf("INI support not implemented")
	default:
		return fmt.Errorf("unsupported file type: %s", p.fileType)
	}
}

// parseJSON parses JSON configuration data
func (p *FileProvider) parseJSON(data []byte) error {
	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	p.configData = flattenMap(configMap, "")
	return nil
}

// flattenMap flattens a nested map with dot notation
func flattenMap(input map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range input {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		
		switch val := v.(type) {
		case map[string]interface{}:
			// Recursive call for nested maps
			flattened := flattenMap(val, key)
			for fk, fv := range flattened {
				result[fk] = fv
			}
		default:
			result[key] = v
		}
	}
	
	return result
}

// Get retrieves a configuration value by key
func (p *FileProvider) Get(key string) (interface{}, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	value, exists := p.configData[key]
	return value, exists
}

// GetString retrieves a string configuration value
func (p *FileProvider) GetString(key string) (string, bool) {
	value, exists := p.Get(key)
	if !exists {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", v), true
	default:
		return "", false
	}
}

// GetInt retrieves an integer configuration value
func (p *FileProvider) GetInt(key string) (int, bool) {
	value, exists := p.Get(key)
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}

	return 0, false
}

// GetBool retrieves a boolean configuration value
func (p *FileProvider) GetBool(key string) (bool, bool) {
	value, exists := p.Get(key)
	if !exists {
		return false, false
	}

	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b, true
		}
	case int:
		return v != 0, true
	case float64:
		return v != 0, true
	}

	return false, false
}

// GetFloat retrieves a float configuration value
func (p *FileProvider) GetFloat(key string) (float64, bool) {
	value, exists := p.Get(key)
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}

	return 0, false
}

// GetDuration retrieves a duration configuration value
func (p *FileProvider) GetDuration(key string) (time.Duration, bool) {
	value, exists := p.GetString(key)
	if !exists {
		return 0, false
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		// Try integer seconds
		if seconds, ok := p.GetInt(key); ok {
			return time.Duration(seconds) * time.Second, true
		}
		return 0, false
	}

	return duration, true
}

// GetStringMap retrieves a map of string configuration values
func (p *FileProvider) GetStringMap(key string) (map[string]string, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Find all keys with the prefix
	prefix := key + "."
	stringMap := make(map[string]string)
	found := false

	for k, v := range p.configData {
		if strings.HasPrefix(k, prefix) {
			subKey := strings.TrimPrefix(k, prefix)
			if str, ok := p.convertToString(v); ok {
				stringMap[subKey] = str
				found = true
			}
		}
	}

	return stringMap, found
}

// GetStringSlice retrieves a slice of string configuration values
func (p *FileProvider) GetStringSlice(key string) ([]string, bool) {
	value, exists := p.Get(key)
	if !exists {
		return nil, false
	}

	switch v := value.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := p.convertToString(item); ok {
				result = append(result, str)
			}
		}
		return result, true
	case string:
		// Split comma-separated string
		return strings.Split(v, ","), true
	}

	return nil, false
}

// convertToString converts a value to string
func (p *FileProvider) convertToString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", v), true
	default:
		return "", false
	}
}

// Set sets a configuration value
func (p *FileProvider) Set(key string, value interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.configData[key] = value
	
	// Notify watchers
	p.notifyWatchers(key, value)
	
	return nil
}

// Has checks if a configuration key exists
func (p *FileProvider) Has(key string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	_, exists := p.configData[key]
	return exists
}

// WatchKey watches a configuration key for changes
func (p *FileProvider) WatchKey(key string) (<-chan interface{}, error) {
	p.watchersMu.Lock()
	defer p.watchersMu.Unlock()

	ch := make(chan interface{}, 1)
	
	if _, exists := p.watchers[key]; !exists {
		p.watchers[key] = make([]chan interface{}, 0)
	}
	
	p.watchers[key] = append(p.watchers[key], ch)
	
	return ch, nil
}

// notifyWatchers notifies all watchers of a key about changes
func (p *FileProvider) notifyWatchers(key string, value interface{}) {
	p.watchersMu.Lock()
	defer p.watchersMu.Unlock()

	if channels, exists := p.watchers[key]; exists {
		for _, ch := range channels {
			select {
			case ch <- value:
				// value sent
			default:
				// channel full, discard
			}
		}
	}
}

// watchFile monitors the configuration file for changes
func (p *FileProvider) watchFile() {
	// Simple polling-based file watcher
	// In a production environment, use a proper file watcher like fsnotify
	lastModTime := time.Time{}
	
	for {
		select {
		case <-p.reloadSignal:
			return
		case <-time.After(5 * time.Second):
			info, err := os.Stat(p.filePath)
			if err != nil {
				continue
			}
			
			if !lastModTime.IsZero() && info.ModTime().After(lastModTime) {
				if err := p.Load(); err == nil {
					lastModTime = info.ModTime()
				}
			} else if lastModTime.IsZero() {
				lastModTime = info.ModTime()
			}
		}
	}
}
