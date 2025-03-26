// service_extension.go - Discovery service extensions for bridge integration

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// BridgeServiceInfo extends ServiceInstance with bridge-specific information
type BridgeServiceInfo struct {
	ServiceInstance
	Protocols     []string               `json:"protocols"`
	Adapters      []string               `json:"adapters"`
	Capabilities  []string               `json:"capabilities"`
	Configuration map[string]interface{} `json:"configuration"`
	Stats         ServiceStats           `json:"stats"`
}

// ServiceStats contains runtime statistics for a service
type ServiceStats struct {
	Uptime            time.Duration `json:"uptime"`
	RequestCount      int64         `json:"request_count"`
	ErrorCount        int64         `json:"error_count"`
	LastRequestTime   time.Time     `json:"last_request_time"`
	AvgResponseTimeMs float64       `json:"avg_response_time_ms"`
	ActiveConnections int           `json:"active_connections"`
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
}

// BridgeDiscovery extends the Registry with bridge-specific functionality
type BridgeDiscovery struct {
	registry          *RegistryImpl
	bridgeServices    map[string]*BridgeServiceInfo
	bridgeServicesMu  sync.RWMutex
	serviceFilters    map[string]ServiceFilter
	serviceFiltersMu  sync.RWMutex
	notificationCh    map[string]chan *ServiceChangeEvent
	notificationChMu  sync.RWMutex
	healthCheckConfig HealthCheckConfig
}

// ServiceFilter contains criteria for filtering services
type ServiceFilter struct {
	Protocols    []string          `json:"protocols"`
	Capabilities []string          `json:"capabilities"`
	Tags         []string          `json:"tags"`
	Metadata     map[string]string `json:"metadata"`
	Status       ServiceStatus     `json:"status"`
}

// ServiceChangeEvent represents a change in a service
type ServiceChangeEvent struct {
	Type      string             `json:"type"`
	ServiceID string             `json:"service_id"`
	Service   *BridgeServiceInfo `json:"service"`
	Timestamp time.Time          `json:"timestamp"`
}

// HealthCheckConfig contains configuration for health checking
type HealthCheckConfig struct {
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
}

// NewBridgeDiscovery creates a new bridge discovery service
func NewBridgeDiscovery(registry *RegistryImpl, config HealthCheckConfig) *BridgeDiscovery {
	return &BridgeDiscovery{
		registry:          registry,
		bridgeServices:    make(map[string]*BridgeServiceInfo),
		serviceFilters:    make(map[string]ServiceFilter),
		notificationCh:    make(map[string]chan *ServiceChangeEvent),
		healthCheckConfig: config,
	}
}

// Registry returns the underlying registry
func (bd *BridgeDiscovery) Registry() *RegistryImpl {
	return bd.registry
}

// RegisterBridgeService registers a bridge service
func (bd *BridgeDiscovery) RegisterBridgeService(ctx context.Context, service *BridgeServiceInfo) error {
	// First register as a regular service
	regOptions := &RegistrationOptions{
		TTL: 60 * time.Second, // Default TTL
	}

	err := bd.registry.Register(&service.ServiceInstance, regOptions)
	if err != nil {
		return err
	}

	// Add to bridge services
	bd.bridgeServicesMu.Lock()
	bd.bridgeServices[service.ID] = service
	bd.bridgeServicesMu.Unlock()

	// Notify subscribers
	bd.notifySubscribers(ctx, &ServiceChangeEvent{
		Type:      "register",
		ServiceID: service.ID,
		Service:   service,
		Timestamp: time.Now(),
	})

	return nil
}

// UnregisterBridgeService unregisters a bridge service
func (bd *BridgeDiscovery) UnregisterBridgeService(ctx context.Context, id string) error {
	// First unregister from registry
	err := bd.registry.Deregister(id)
	if err != nil {
		return err
	}

	// Remove from bridge services
	bd.bridgeServicesMu.Lock()
	delete(bd.bridgeServices, id)
	bd.bridgeServicesMu.Unlock()

	// Notify subscribers
	bd.notifySubscribers(ctx, &ServiceChangeEvent{
		Type:      "unregister",
		ServiceID: id,
		Timestamp: time.Now(),
	})

	return nil
}

// GetBridgeService gets a bridge service by ID
func (bd *BridgeDiscovery) GetBridgeService(id string) (*BridgeServiceInfo, error) {
	bd.bridgeServicesMu.RLock()
	defer bd.bridgeServicesMu.RUnlock()

	service, exists := bd.bridgeServices[id]
	if !exists {
		return nil, fmt.Errorf("bridge service '%s' not found", id)
	}

	return service, nil
}

// UpdateBridgeServiceStats updates the stats for a bridge service
func (bd *BridgeDiscovery) UpdateBridgeServiceStats(id string, stats ServiceStats) error {
	bd.bridgeServicesMu.Lock()
	defer bd.bridgeServicesMu.Unlock()

	service, exists := bd.bridgeServices[id]
	if !exists {
		return fmt.Errorf("bridge service '%s' not found", id)
	}

	service.Stats = stats

	return nil
}

// FindServices finds services matching a filter
func (bd *BridgeDiscovery) FindServices(filter ServiceFilter) ([]*BridgeServiceInfo, error) {
	bd.bridgeServicesMu.RLock()
	defer bd.bridgeServicesMu.RUnlock()

	matches := make([]*BridgeServiceInfo, 0)

	for _, service := range bd.bridgeServices {
		if bd.matchesFilter(service, filter) {
			matches = append(matches, service)
		}
	}

	return matches, nil
}

// RegisterServiceWatcher registers a service watcher with a filter
func (bd *BridgeDiscovery) RegisterServiceWatcher(id string, filter ServiceFilter, ch chan *ServiceChangeEvent) error {
	bd.serviceFiltersMu.Lock()
	bd.serviceFilters[id] = filter
	bd.serviceFiltersMu.Unlock()

	bd.notificationChMu.Lock()
	bd.notificationCh[id] = ch
	bd.notificationChMu.Unlock()

	return nil
}

// UnregisterServiceWatcher unregisters a service watcher
func (bd *BridgeDiscovery) UnregisterServiceWatcher(id string) error {
	bd.serviceFiltersMu.Lock()
	delete(bd.serviceFilters, id)
	bd.serviceFiltersMu.Unlock()

	bd.notificationChMu.Lock()
	delete(bd.notificationCh, id)
	bd.notificationChMu.Unlock()

	return nil
}

// notifySubscribers notifies subscribers of a service change
func (bd *BridgeDiscovery) notifySubscribers(ctx context.Context, event *ServiceChangeEvent) {
	bd.notificationChMu.RLock()
	defer bd.notificationChMu.RUnlock()

	// If this is a register/update event, check filters
	if event.Type == "register" || event.Type == "update" {
		for id, ch := range bd.notificationCh {
			bd.serviceFiltersMu.RLock()
			filter, exists := bd.serviceFilters[id]
			bd.serviceFiltersMu.RUnlock()

			if !exists || bd.matchesFilter(event.Service, filter) {
				select {
				case ch <- event:
					// Notification sent
				default:
					// Channel full, skip
				}
			}
		}
	} else {
		// For unregister events, notify all subscribers
		for _, ch := range bd.notificationCh {
			select {
			case ch <- event:
				// Notification sent
			default:
				// Channel full, skip
			}
		}
	}
}

// matchesFilter checks if a service matches a filter
func (bd *BridgeDiscovery) matchesFilter(service *BridgeServiceInfo, filter ServiceFilter) bool {
	// Check protocols
	if len(filter.Protocols) > 0 {
		matches := false
		for _, proto := range filter.Protocols {
			for _, sProto := range service.Protocols {
				if proto == sProto {
					matches = true
					break
				}
			}
			if matches {
				break
			}
		}
		if !matches {
			return false
		}
	}

	// Check capabilities
	if len(filter.Capabilities) > 0 {
		matches := false
		for _, cap := range filter.Capabilities {
			for _, sCap := range service.Capabilities {
				if cap == sCap {
					matches = true
					break
				}
			}
			if matches {
				break
			}
		}
		if !matches {
			return false
		}
	}

	// Check tags
	if len(filter.Tags) > 0 {
		matches := false
		for _, tag := range filter.Tags {
			for _, sTag := range service.Tags {
				if tag == sTag {
					matches = true
					break
				}
			}
			if matches {
				break
			}
		}
		if !matches {
			return false
		}
	}

	// Check metadata
	if len(filter.Metadata) > 0 {
		for key, value := range filter.Metadata {
			sValue, exists := service.Metadata[key]
			if !exists || sValue != value {
				return false
			}
		}
	}

	// Check status
	if filter.Status != "" && service.Status != filter.Status {
		return false
	}

	return true
}

// SerializeBridgeService serializes a bridge service to JSON
func SerializeBridgeService(service *BridgeServiceInfo) ([]byte, error) {
	return json.Marshal(service)
}

// DeserializeBridgeService deserializes a bridge service from JSON
func DeserializeBridgeService(data []byte) (*BridgeServiceInfo, error) {
	var service BridgeServiceInfo
	err := json.Unmarshal(data, &service)
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// NewBridgeServiceInfo creates a new bridge service info
func NewBridgeServiceInfo(baseService *ServiceInstance) *BridgeServiceInfo {
	return &BridgeServiceInfo{
		ServiceInstance: *baseService,
		Protocols:       []string{},
		Adapters:        []string{},
		Capabilities:    []string{},
		Configuration:   make(map[string]interface{}),
		Stats: ServiceStats{
			Uptime: 0,
		},
	}
}

// RegistrationOptions defines options for service registration
type RegistrationOptions struct {
	TTL time.Duration
}
