package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"
)

// ConfigManager handles configuration management for the integration layer
type ConfigManager struct {
	mu            sync.RWMutex
	config        *IntegrationConfig
	configFile    string
	lastModified  time.Time
	changeHandler func(config *IntegrationConfig)
}

// IntegrationConfig contains configuration for the integration layer
type IntegrationConfig struct {
	ServiceRoutes map[string][]string `json:"serviceRoutes"`
	ProxyConfig   ProxyConfig         `json:"proxyConfig"`
	Metrics       MetricsConfig       `json:"metrics"`
}

// ProxyConfig contains proxy-specific configuration
type ProxyConfig struct {
	EnableLoadBalancing bool              `json:"enableLoadBalancing"`
	HealthCheck        HealthCheckConfig `json:"healthCheck"`
	Timeouts           TimeoutConfig     `json:"timeouts"`
}

// HealthCheckConfig contains health check configuration
type HealthCheckConfig struct {
	Interval       time.Duration `json:"interval"`
	Timeout        time.Duration `json:"timeout"`
	FailureThreshold int         `json:"failureThreshold"`
}

// TimeoutConfig contains timeout configuration
type TimeoutConfig struct {
	Read       time.Duration `json:"read"`
	Write      time.Duration `json:"write"`
	Idle       time.Duration `json:"idle"`
	DialTimeout time.Duration `json:"dialTimeout"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	EnableDetailedMetrics bool          `json:"enableDetailedMetrics"`
	RetentionPeriod      time.Duration `json:"retentionPeriod"`
	SampleInterval       time.Duration `json:"sampleInterval"`
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configFile string, changeHandler func(config *IntegrationConfig)) (*ConfigManager, error) {
	cm := &ConfigManager{
		configFile:    configFile,
		changeHandler: changeHandler,
	}

	if err := cm.loadConfig(); err != nil {
		return nil, err
	}

	return cm, nil
}

// loadConfig loads configuration from file
func (cm *ConfigManager) loadConfig() error {
	data, err := ioutil.ReadFile(cm.configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	var config IntegrationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	cm.mu.Lock()
	cm.config = &config
	fileInfo, _ := ioutil.Stat(cm.configFile)
	cm.lastModified = fileInfo.ModTime()
	cm.mu.Unlock()

	if cm.changeHandler != nil {
		cm.changeHandler(&config)
	}

	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *IntegrationConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// UpdateConfig updates the configuration and saves it to file
func (cm *ConfigManager) UpdateConfig(config *IntegrationConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	if err := ioutil.WriteFile(cm.configFile, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	cm.mu.Lock()
	cm.config = config
	fileInfo, _ := ioutil.Stat(cm.configFile)
	cm.lastModified = fileInfo.ModTime()
	cm.mu.Unlock()

	if cm.changeHandler != nil {
		cm.changeHandler(config)
	}

	return nil
}

// AddServiceRoute adds a route configuration for a service
func (cm *ConfigManager) AddServiceRoute(serviceName, route string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.config.ServiceRoutes == nil {
		cm.config.ServiceRoutes = make(map[string][]string)
	}

	cm.config.ServiceRoutes[serviceName] = append(cm.config.ServiceRoutes[serviceName], route)
	return cm.UpdateConfig(cm.config)
}

// RemoveServiceRoute removes a route configuration for a service
func (cm *ConfigManager) RemoveServiceRoute(serviceName, route string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if routes, exists := cm.config.ServiceRoutes[serviceName]; exists {
		var newRoutes []string
		for _, r := range routes {
			if r != route {
				newRoutes = append(newRoutes, r)
			}
		}
		cm.config.ServiceRoutes[serviceName] = newRoutes
		return cm.UpdateConfig(cm.config)
	}

	return nil
}

// UpdateProxyConfig updates the proxy configuration
func (cm *ConfigManager) UpdateProxyConfig(config ProxyConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config.ProxyConfig = config
	return cm.UpdateConfig(cm.config)
}

// UpdateMetricsConfig updates the metrics configuration
func (cm *ConfigManager) UpdateMetricsConfig(config MetricsConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config.Metrics = config
	return cm.UpdateConfig(cm.config)
}
