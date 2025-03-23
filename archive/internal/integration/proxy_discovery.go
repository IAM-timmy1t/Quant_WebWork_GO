package integration

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/timot/Quant_WebWork_GO/internal/discovery"
	"github.com/timot/Quant_WebWork_GO/internal/proxy"
)

// ProxyDiscoveryIntegrator manages the integration between service discovery and proxy
type ProxyDiscoveryIntegrator struct {
	discovery     *discovery.ServiceRegistry
	proxyManager  *proxy.Manager
	eventChan     chan discovery.ServiceEvent
	routeMap      map[string][]string // Maps service names to their route paths
	serviceMap    map[string]*discovery.Service
	mu            sync.RWMutex
	metrics       *ServiceMetrics
}

// ServiceMetrics tracks service-level metrics
type ServiceMetrics struct {
	mu sync.RWMutex
	// Maps service ID to its metrics
	metrics map[string]*ServiceMetricData
}

// ServiceMetricData contains metrics for a single service
type ServiceMetricData struct {
	ServiceID      string
	ServiceName    string
	LastSeen       time.Time
	Status         string
	ResponseTimes  []time.Duration
	RequestCount   int64
	ErrorCount     int64
	LastError      time.Time
	LastErrorMsg   string
	AvailableTime  time.Duration
	DownTime       time.Duration
	StatusHistory  []StatusChange
}

// StatusChange represents a service status change
type StatusChange struct {
	Timestamp time.Time
	OldStatus string
	NewStatus string
}

// NewProxyDiscoveryIntegrator creates a new integrator instance
func NewProxyDiscoveryIntegrator(disc *discovery.ServiceRegistry, proxy *proxy.Manager) *ProxyDiscoveryIntegrator {
	return &ProxyDiscoveryIntegrator{
		discovery:    disc,
		proxyManager: proxy,
		eventChan:    disc.Subscribe(),
		routeMap:     make(map[string][]string),
		serviceMap:   make(map[string]*discovery.Service),
		metrics:      newServiceMetrics(),
	}
}

// Start begins monitoring service events and updating proxy routes
func (pdi *ProxyDiscoveryIntegrator) Start(ctx context.Context) error {
	go pdi.processEvents(ctx)
	go pdi.updateMetrics(ctx)
	return nil
}

// Stop gracefully shuts down the integrator
func (pdi *ProxyDiscoveryIntegrator) Stop() {
	pdi.discovery.Unsubscribe(pdi.eventChan)
}

// processEvents handles service discovery events
func (pdi *ProxyDiscoveryIntegrator) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-pdi.eventChan:
			pdi.handleServiceEvent(event)
		}
	}
}

// handleServiceEvent processes individual service events
func (pdi *ProxyDiscoveryIntegrator) handleServiceEvent(event discovery.ServiceEvent) {
	pdi.mu.Lock()
	defer pdi.mu.Unlock()

	service := event.Service
	serviceURL := fmt.Sprintf("%s://%s:%d", service.Protocol, service.Address, service.Port)

	switch event.Type {
	case "register", "update":
		// Update service map
		pdi.serviceMap[service.ID] = service

		// Update route map
		paths := pdi.routeMap[service.Name]
		if paths == nil {
			// Create default path based on service name
			paths = []string{fmt.Sprintf("/%s/*", service.Name)}
			pdi.routeMap[service.Name] = paths
		}

		// Update proxy routes
		for _, path := range paths {
			targets := []string{serviceURL}
			// Get other instances of the same service
			for _, s := range pdi.serviceMap {
				if s.Name == service.Name && s.ID != service.ID {
					targets = append(targets, fmt.Sprintf("%s://%s:%d", s.Protocol, s.Address, s.Port))
				}
			}

			// Update or create proxy route
			if err := pdi.updateProxyRoute(path, targets); err != nil {
				log.Printf("Error updating proxy route for service %s: %v", service.Name, err)
			}
		}

		// Update metrics
		pdi.updateServiceMetrics(service, event.Type)

	case "deregister", "expire":
		delete(pdi.serviceMap, service.ID)
		
		// Update proxy routes to remove this instance
		paths := pdi.routeMap[service.Name]
		for _, path := range paths {
			var targets []string
			for _, s := range pdi.serviceMap {
				if s.Name == service.Name {
					targets = append(targets, fmt.Sprintf("%s://%s:%d", s.Protocol, s.Address, s.Port))
				}
			}

			if len(targets) > 0 {
				if err := pdi.updateProxyRoute(path, targets); err != nil {
					log.Printf("Error updating proxy route after service removal: %v", err)
				}
			} else {
				// Remove route if no instances left
				pdi.proxyManager.RemoveRoute(path)
			}
		}

		// Update metrics
		pdi.updateServiceMetrics(service, event.Type)
	}
}

// updateProxyRoute updates or creates a proxy route
func (pdi *ProxyDiscoveryIntegrator) updateProxyRoute(path string, targets []string) error {
	// Parse target URLs
	var targetURLs []string
	for _, target := range targets {
		targetURL, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("invalid target URL %s: %v", target, err)
		}
		targetURLs = append(targetURLs, targetURL.String())
	}

	// Update route with load balancing enabled
	return pdi.proxyManager.UpdateRoute(path, targetURLs, true)
}

// AddServiceRoute adds a custom route path for a service
func (pdi *ProxyDiscoveryIntegrator) AddServiceRoute(serviceName, path string) {
	pdi.mu.Lock()
	defer pdi.mu.Unlock()

	paths := pdi.routeMap[serviceName]
	pdi.routeMap[serviceName] = append(paths, path)

	// Update proxy routes for existing service instances
	var targets []string
	for _, service := range pdi.serviceMap {
		if service.Name == serviceName {
			targets = append(targets, fmt.Sprintf("%s://%s:%d", 
				service.Protocol, service.Address, service.Port))
		}
	}

	if len(targets) > 0 {
		if err := pdi.updateProxyRoute(path, targets); err != nil {
			log.Printf("Error adding new route for service %s: %v", serviceName, err)
		}
	}
}

// newServiceMetrics creates a new ServiceMetrics instance
func newServiceMetrics() *ServiceMetrics {
	return &ServiceMetrics{
		metrics: make(map[string]*ServiceMetricData),
	}
}

// updateServiceMetrics updates metrics for a service
func (pdi *ProxyDiscoveryIntegrator) updateServiceMetrics(service *discovery.Service, eventType string) {
	pdi.metrics.mu.Lock()
	defer pdi.metrics.mu.Unlock()

	data, exists := pdi.metrics.metrics[service.ID]
	if !exists {
		data = &ServiceMetricData{
			ServiceID:   service.ID,
			ServiceName: service.Name,
			LastSeen:    time.Now(),
		}
		pdi.metrics.metrics[service.ID] = data
	}

	// Update metrics based on event type
	now := time.Now()
	data.LastSeen = now

	if data.Status != service.Status {
		data.StatusHistory = append(data.StatusHistory, StatusChange{
			Timestamp: now,
			OldStatus: data.Status,
			NewStatus: service.Status,
		})
		data.Status = service.Status
	}

	switch eventType {
	case "deregister", "expire":
		data.DownTime += time.Since(data.LastSeen)
	case "register", "update":
		if data.Status == "healthy" {
			data.AvailableTime += time.Since(data.LastSeen)
		}
	}
}

// updateMetrics periodically updates service metrics
func (pdi *ProxyDiscoveryIntegrator) updateMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pdi.metrics.mu.Lock()
			now := time.Now()
			for _, data := range pdi.metrics.metrics {
				if data.Status == "healthy" {
					data.AvailableTime += time.Since(data.LastSeen)
				} else {
					data.DownTime += time.Since(data.LastSeen)
				}
				data.LastSeen = now
			}
			pdi.metrics.mu.Unlock()
		}
	}
}

// GetServiceMetrics returns metrics for all services
func (pdi *ProxyDiscoveryIntegrator) GetServiceMetrics() map[string]*ServiceMetricData {
	pdi.metrics.mu.RLock()
	defer pdi.metrics.mu.RUnlock()

	metrics := make(map[string]*ServiceMetricData)
	for id, data := range pdi.metrics.metrics {
		metrics[id] = data
	}
	return metrics
}
