// service.go - Service discovery implementation

package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"
	"go.uber.org/zap"
)

// Service represents a service in the discovery system
type Service struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Protocol    string            `json:"protocol"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	HealthCheck string            `json:"health_check"`
	Metadata    map[string]string `json:"metadata"`
	Status      string            `json:"status"`
	LastSeen    time.Time         `json:"last_seen"`
}

// BridgeDiscovery manages service discovery
type BridgeDiscovery struct {
	config       config.DiscoveryConfig
	services     map[string]*Service
	mutex        sync.RWMutex
	logger       *zap.SugaredLogger
	refreshTimer *time.Ticker
	shutdown     chan struct{}
}

// NewService creates a new discovery service
func NewService(cfg config.DiscoveryConfig, logger *zap.SugaredLogger) (*BridgeDiscovery, error) {
	service := &BridgeDiscovery{
		config:   cfg,
		services: make(map[string]*Service),
		logger:   logger,
		shutdown: make(chan struct{}),
	}

	return service, nil
}

// Start starts the discovery service
func (d *BridgeDiscovery) Start(ctx context.Context) error {
	if !d.config.Enabled {
		d.logger.Info("Service discovery is disabled")
		return nil
	}

	d.logger.Info("Starting service discovery with refresh interval",
		"interval", d.config.RefreshInterval)

	// Initialize refresh timer
	d.refreshTimer = time.NewTicker(d.config.RefreshInterval)

	// Start refresh loop
	go d.refreshLoop()

	return nil
}

// Stop stops the discovery service
func (d *BridgeDiscovery) Stop() error {
	if d.refreshTimer != nil {
		d.refreshTimer.Stop()
	}

	close(d.shutdown)
	return nil
}

// RegisterService registers a service with the discovery system
func (d *BridgeDiscovery) RegisterService(service *Service) error {
	if service.ID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	service.LastSeen = time.Now()
	d.services[service.ID] = service

	d.logger.Infow("Registered service",
		"id", service.ID,
		"name", service.Name,
		"protocol", service.Protocol,
		"host", service.Host,
		"port", service.Port)

	return nil
}

// UnregisterService removes a service from the discovery system
func (d *BridgeDiscovery) UnregisterService(serviceID string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if _, exists := d.services[serviceID]; !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	delete(d.services, serviceID)
	d.logger.Infow("Unregistered service", "id", serviceID)

	return nil
}

// GetService retrieves a service by ID
func (d *BridgeDiscovery) GetService(serviceID string) (*Service, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	service, exists := d.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return service, nil
}

// ListServices lists all registered services
func (d *BridgeDiscovery) ListServices() []*Service {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	services := make([]*Service, 0, len(d.services))
	for _, service := range d.services {
		services = append(services, service)
	}

	return services
}

// FindServicesByProtocol finds services by protocol
func (d *BridgeDiscovery) FindServicesByProtocol(protocol string) []*Service {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var results []*Service
	for _, service := range d.services {
		if service.Protocol == protocol {
			results = append(results, service)
		}
	}

	return results
}

// refreshLoop periodically refreshes the service registry
func (d *BridgeDiscovery) refreshLoop() {
	for {
		select {
		case <-d.refreshTimer.C:
			d.refreshServices()
		case <-d.shutdown:
			return
		}
	}
}

// refreshServices refreshes the service registry
func (d *BridgeDiscovery) refreshServices() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	now := time.Now()
	timeout := now.Add(-d.config.RefreshInterval * 3) // Consider a service timed out after 3 refresh intervals

	// Check for stale services
	for id, service := range d.services {
		if service.LastSeen.Before(timeout) {
			d.logger.Warnw("Service timed out, marking as unavailable",
				"id", id,
				"name", service.Name,
				"lastSeen", service.LastSeen)
			service.Status = "unavailable"
		}
	}

	// In a real implementation, we would also perform health checks here
	// and potentially remove services that have been unavailable for too long
}
