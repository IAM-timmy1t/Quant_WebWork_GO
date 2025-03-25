// service.go - Service discovery and registration component

package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrServiceNotFound    = errors.New("service not found")
	ErrServiceExists      = errors.New("service already exists")
	ErrInvalidService     = errors.New("invalid service definition")
	ErrRegistrationFailed = errors.New("service registration failed")
)

// ServiceStatus represents the current status of a service
type ServiceStatus string

// Service statuses
const (
	StatusUnknown    ServiceStatus = "unknown"
	StatusStarting   ServiceStatus = "starting"
	StatusRunning    ServiceStatus = "running"
	StatusDegraded   ServiceStatus = "degraded"
	StatusStopping   ServiceStatus = "stopping"
	StatusStopped    ServiceStatus = "stopped"
	StatusMaintenance ServiceStatus = "maintenance"
	StatusFailed     ServiceStatus = "failed"
)

// ServiceType represents the type of service
type ServiceType string

// Service types
const (
	TypeCore       ServiceType = "core"
	TypeAPI        ServiceType = "api"
	TypeSecurity   ServiceType = "security"
	TypeBridge     ServiceType = "bridge"
	TypeStorage    ServiceType = "storage"
	TypeProcessing ServiceType = "processing"
	TypeMonitoring ServiceType = "monitoring"
	TypeInternal   ServiceType = "internal"
	TypeExternal   ServiceType = "external"
)

// ServiceInfo contains information about a registered service
type ServiceInfo struct {
	ID              string                 // Unique identifier
	Name            string                 // Service name
	Type            ServiceType            // Service type
	Version         string                 // Service version
	Description     string                 // Service description
	Endpoints       []ServiceEndpoint      // Service endpoints
	Dependencies    []string               // Dependencies (other service IDs)
	Status          ServiceStatus          // Current status
	StatusMessage   string                 // Status message
	Metadata        map[string]interface{} // Additional metadata
	Tags            []string               // Service tags
	RegisteredAt    time.Time              // Registration time
	LastUpdated     time.Time              // Last update time
	HealthCheckData HealthCheckData        // Health check data
}

// ServiceEndpoint represents an endpoint exposed by a service
type ServiceEndpoint struct {
	Name        string                 // Endpoint name
	URL         string                 // Endpoint URL
	Protocol    string                 // Protocol (http, grpc, etc.)
	Description string                 // Endpoint description
	Methods     []string               // Supported methods
	Metadata    map[string]interface{} // Additional metadata
	Deprecated  bool                   // Whether the endpoint is deprecated
	AuthRequired bool                  // Whether authentication is required
}

// HealthCheckData contains health check information
type HealthCheckData struct {
	Status       ServiceStatus // Health status
	LastChecked  time.Time     // Last check time
	CheckCount   int           // Number of checks performed
	FailureCount int           // Number of failed checks
	LatencyMs    int64         // Average latency in milliseconds
	Message      string        // Health check message
	Details      interface{}   // Detailed health information
}

// DiscoveryEvent represents an event in the service discovery system
type DiscoveryEvent struct {
	Type      string      // Event type
	ServiceID string      // Related service ID
	Timestamp time.Time   // Event timestamp
	Payload   interface{} // Event payload
}

// EventType constants
const (
	EventRegistered   = "service.registered"
	EventUnregistered = "service.unregistered"
	EventUpdated      = "service.updated"
	EventStatusChange = "service.status.changed"
	EventHealthChange = "service.health.changed"
)

// EventHandlerFunc defines a function type for handling discovery events
type EventHandlerFunc func(event DiscoveryEvent)

// ServiceRegistration represents a service registration request
type ServiceRegistration struct {
	Name        string                 // Service name
	Type        ServiceType            // Service type
	Version     string                 // Service version
	Description string                 // Service description
	Endpoints   []ServiceEndpoint      // Service endpoints
	Dependencies []string              // Dependencies (other service IDs)
	Metadata    map[string]interface{} // Additional metadata
	Tags        []string               // Service tags
}

// ServiceQuery represents a query for finding services
type ServiceQuery struct {
	Name         string          // Service name
	Type         ServiceType     // Service type
	Tags         []string        // Service tags
	Dependencies []string        // Required dependencies
	Status       []ServiceStatus // Acceptable statuses
	Metadata     map[string]interface{} // Metadata to match
}

// ServiceRegistry provides service discovery and registration
type ServiceRegistry struct {
	services       map[string]*ServiceInfo   // Registered services by ID
	servicesByName map[string][]*ServiceInfo // Services by name
	servicesByType map[ServiceType][]*ServiceInfo // Services by type
	servicesByTag  map[string][]*ServiceInfo // Services by tag
	mutex          sync.RWMutex              // Mutex for thread safety
	eventHandlers  []EventHandlerFunc        // Event handlers
	healthChecker  HealthChecker             // Health checker
	logger         Logger                    // Logger
}

// HealthChecker performs health checks on services
type HealthChecker interface {
	CheckHealth(serviceInfo *ServiceInfo) (HealthCheckData, error)
	StartChecking(serviceInfo *ServiceInfo) error
	StopChecking(serviceID string) error
}

// Logger interface for discovery logging
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(healthChecker HealthChecker, logger Logger) *ServiceRegistry {
	if logger == nil {
		// Use a no-op logger if none provided
		logger = &noopLogger{}
	}

	return &ServiceRegistry{
		services:       make(map[string]*ServiceInfo),
		servicesByName: make(map[string][]*ServiceInfo),
		servicesByType: make(map[ServiceType][]*ServiceInfo),
		servicesByTag:  make(map[string][]*ServiceInfo),
		healthChecker:  healthChecker,
		logger:         logger,
	}
}

// RegisterService registers a new service
func (r *ServiceRegistry) RegisterService(reg ServiceRegistration) (*ServiceInfo, error) {
	// Validate registration
	if reg.Name == "" {
		return nil, errors.New("service name is required")
	}

	// Create service info
	serviceInfo := &ServiceInfo{
		ID:           uuid.New().String(),
		Name:         reg.Name,
		Type:         reg.Type,
		Version:      reg.Version,
		Description:  reg.Description,
		Endpoints:    reg.Endpoints,
		Dependencies: reg.Dependencies,
		Status:       StatusStarting,
		Metadata:     reg.Metadata,
		Tags:         reg.Tags,
		RegisteredAt: time.Now(),
		LastUpdated:  time.Now(),
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Register service
	r.services[serviceInfo.ID] = serviceInfo

	// Update indexes
	r.servicesByName[reg.Name] = append(r.servicesByName[reg.Name], serviceInfo)
	r.servicesByType[reg.Type] = append(r.servicesByType[reg.Type], serviceInfo)
	for _, tag := range reg.Tags {
		r.servicesByTag[tag] = append(r.servicesByTag[tag], serviceInfo)
	}

	// Start health checking if health checker available
	if r.healthChecker != nil {
		if err := r.healthChecker.StartChecking(serviceInfo); err != nil {
			r.logger.Warn("Failed to start health checking", map[string]interface{}{
				"service_id":   serviceInfo.ID,
				"service_name": serviceInfo.Name,
				"error":        err.Error(),
			})
		}
	}

	r.logger.Info("Service registered", map[string]interface{}{
		"service_id":   serviceInfo.ID,
		"service_name": serviceInfo.Name,
		"service_type": string(serviceInfo.Type),
	})

	// Notify event handlers
	r.notifyEventHandlers(DiscoveryEvent{
		Type:      EventRegistered,
		ServiceID: serviceInfo.ID,
		Timestamp: time.Now(),
		Payload:   serviceInfo,
	})

	return serviceInfo, nil
}

// UnregisterService removes a service registration
func (r *ServiceRegistry) UnregisterService(serviceID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	service, exists := r.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}

	// Stop health checking
	if r.healthChecker != nil {
		if err := r.healthChecker.StopChecking(serviceID); err != nil {
			r.logger.Warn("Failed to stop health checking", map[string]interface{}{
				"service_id":   serviceID,
				"service_name": service.Name,
				"error":        err.Error(),
			})
		}
	}

	// Remove from indexes
	delete(r.services, serviceID)

	// Update servicesByName
	servicesByName := r.servicesByName[service.Name]
	for i, s := range servicesByName {
		if s.ID == serviceID {
			r.servicesByName[service.Name] = append(servicesByName[:i], servicesByName[i+1:]...)
			break
		}
	}
	if len(r.servicesByName[service.Name]) == 0 {
		delete(r.servicesByName, service.Name)
	}

	// Update servicesByType
	servicesByType := r.servicesByType[service.Type]
	for i, s := range servicesByType {
		if s.ID == serviceID {
			r.servicesByType[service.Type] = append(servicesByType[:i], servicesByType[i+1:]...)
			break
		}
	}
	if len(r.servicesByType[service.Type]) == 0 {
		delete(r.servicesByType, service.Type)
	}

	// Update servicesByTag
	for _, tag := range service.Tags {
		servicesByTag := r.servicesByTag[tag]
		for i, s := range servicesByTag {
			if s.ID == serviceID {
				r.servicesByTag[tag] = append(servicesByTag[:i], servicesByTag[i+1:]...)
				break
			}
		}
		if len(r.servicesByTag[tag]) == 0 {
			delete(r.servicesByTag, tag)
		}
	}

	r.logger.Info("Service unregistered", map[string]interface{}{
		"service_id":   serviceID,
		"service_name": service.Name,
	})

	// Notify event handlers
	r.notifyEventHandlers(DiscoveryEvent{
		Type:      EventUnregistered,
		ServiceID: serviceID,
		Timestamp: time.Now(),
		Payload:   service,
	})

	return nil
}

// UpdateService updates service information
func (r *ServiceRegistry) UpdateService(serviceID string, updater func(*ServiceInfo)) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	service, exists := r.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}

	// Create a copy for the event
	oldService := *service

	// Update service
	updater(service)
	service.LastUpdated = time.Now()

	r.logger.Debug("Service updated", map[string]interface{}{
		"service_id":   serviceID,
		"service_name": service.Name,
	})

	// Notify event handlers
	r.notifyEventHandlers(DiscoveryEvent{
		Type:      EventUpdated,
		ServiceID: serviceID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"old": oldService,
			"new": *service,
		},
	})

	return nil
}

// UpdateServiceStatus updates a service's status
func (r *ServiceRegistry) UpdateServiceStatus(serviceID string, status ServiceStatus, message string) error {
	return r.UpdateService(serviceID, func(service *ServiceInfo) {
		oldStatus := service.Status
		service.Status = status
		service.StatusMessage = message

		// Notify status change
		if oldStatus != status {
			r.notifyEventHandlers(DiscoveryEvent{
				Type:      EventStatusChange,
				ServiceID: serviceID,
				Timestamp: time.Now(),
				Payload: map[string]interface{}{
					"old_status": oldStatus,
					"new_status": status,
					"message":    message,
				},
			})
		}
	})
}

// GetService retrieves a service by ID
func (r *ServiceRegistry) GetService(serviceID string) (*ServiceInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	service, exists := r.services[serviceID]
	if !exists {
		return nil, ErrServiceNotFound
	}

	// Return a copy to prevent external modification
	serviceCopy := *service
	return &serviceCopy, nil
}

// FindServices finds services matching a query
func (r *ServiceRegistry) FindServices(query ServiceQuery) []*ServiceInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Start with all services or filter by type
	var candidates []*ServiceInfo
	if query.Type != "" {
		candidates = append([]*ServiceInfo{}, r.servicesByType[query.Type]...)
	} else {
		candidates = make([]*ServiceInfo, 0, len(r.services))
		for _, service := range r.services {
			candidates = append(candidates, service)
		}
	}

	// Apply filters
	var results []*ServiceInfo
	for _, service := range candidates {
		// Filter by name
		if query.Name != "" && service.Name != query.Name {
			continue
		}

		// Filter by status
		if len(query.Status) > 0 {
			statusMatch := false
			for _, status := range query.Status {
				if service.Status == status {
					statusMatch = true
					break
				}
			}
			if !statusMatch {
				continue
			}
		}

		// Filter by tags
		if len(query.Tags) > 0 {
			tagMatch := true
			for _, queryTag := range query.Tags {
				found := false
				for _, serviceTag := range service.Tags {
					if serviceTag == queryTag {
						found = true
						break
					}
				}
				if !found {
					tagMatch = false
					break
				}
			}
			if !tagMatch {
				continue
			}
		}

		// Filter by dependencies
		if len(query.Dependencies) > 0 {
			depMatch := true
			for _, queryDep := range query.Dependencies {
				found := false
				for _, serviceDep := range service.Dependencies {
					if serviceDep == queryDep {
						found = true
						break
					}
				}
				if !found {
					depMatch = false
					break
				}
			}
			if !depMatch {
				continue
			}
		}

		// Filter by metadata
		if len(query.Metadata) > 0 {
			metaMatch := true
			for key, queryValue := range query.Metadata {
				serviceValue, exists := service.Metadata[key]
				if !exists || serviceValue != queryValue {
					metaMatch = false
					break
				}
			}
			if !metaMatch {
				continue
			}
		}

		// Service matched all filters
		results = append(results, service)
	}

	// Return copies to prevent external modification
	resultsCopy := make([]*ServiceInfo, len(results))
	for i, service := range results {
		serviceCopy := *service
		resultsCopy[i] = &serviceCopy
	}

	return resultsCopy
}

// GetServicesByName finds services by name
func (r *ServiceRegistry) GetServicesByName(name string) []*ServiceInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	services := r.servicesByName[name]
	if len(services) == 0 {
		return nil
	}

	// Return copies to prevent external modification
	result := make([]*ServiceInfo, len(services))
	for i, service := range services {
		serviceCopy := *service
		result[i] = &serviceCopy
	}

	return result
}

// GetServicesByType finds services by type
func (r *ServiceRegistry) GetServicesByType(serviceType ServiceType) []*ServiceInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	services := r.servicesByType[serviceType]
	if len(services) == 0 {
		return nil
	}

	// Return copies to prevent external modification
	result := make([]*ServiceInfo, len(services))
	for i, service := range services {
		serviceCopy := *service
		result[i] = &serviceCopy
	}

	return result
}

// GetServicesByTag finds services by tag
func (r *ServiceRegistry) GetServicesByTag(tag string) []*ServiceInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	services := r.servicesByTag[tag]
	if len(services) == 0 {
		return nil
	}

	// Return copies to prevent external modification
	result := make([]*ServiceInfo, len(services))
	for i, service := range services {
		serviceCopy := *service
		result[i] = &serviceCopy
	}

	return result
}

// ListAllServices returns all registered services
func (r *ServiceRegistry) ListAllServices() []*ServiceInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*ServiceInfo, 0, len(r.services))
	for _, service := range r.services {
		serviceCopy := *service
		result = append(result, &serviceCopy)
	}

	return result
}

// AddEventHandler adds an event handler
func (r *ServiceRegistry) AddEventHandler(handler EventHandlerFunc) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.eventHandlers = append(r.eventHandlers, handler)
}

// notifyEventHandlers notifies all event handlers
func (r *ServiceRegistry) notifyEventHandlers(event DiscoveryEvent) {
	// Make a copy of the handlers to avoid deadlocks
	r.mutex.RLock()
	handlers := make([]EventHandlerFunc, len(r.eventHandlers))
	copy(handlers, r.eventHandlers)
	r.mutex.RUnlock()

	// Notify handlers
	for _, handler := range handlers {
		go handler(event)
	}
}

// UpdateHealthCheck updates a service's health check data
func (r *ServiceRegistry) UpdateHealthCheck(serviceID string, health HealthCheckData) error {
	return r.UpdateService(serviceID, func(service *ServiceInfo) {
		oldStatus := service.HealthCheckData.Status
		service.HealthCheckData = health

		// Notify health change
		if oldStatus != health.Status {
			r.notifyEventHandlers(DiscoveryEvent{
				Type:      EventHealthChange,
				ServiceID: serviceID,
				Timestamp: time.Now(),
				Payload: map[string]interface{}{
					"old_status": oldStatus,
					"new_status": health.Status,
					"health":     health,
				},
			})
		}
	})
}

// GetHealthStatus gets a service's health status
func (r *ServiceRegistry) GetHealthStatus(serviceID string) (HealthCheckData, error) {
	service, err := r.GetService(serviceID)
	if err != nil {
		return HealthCheckData{}, err
	}

	return service.HealthCheckData, nil
}

// DependencyCheck checks if all dependencies are available for a service
func (r *ServiceRegistry) DependencyCheck(serviceID string) (bool, []string, error) {
	service, err := r.GetService(serviceID)
	if err != nil {
		return false, nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	missing := []string{}
	for _, depID := range service.Dependencies {
		dep, exists := r.services[depID]
		if !exists || dep.Status != StatusRunning {
			missing = append(missing, depID)
		}
	}

	return len(missing) == 0, missing, nil
}

// noopLogger is a no-op implementation of the Logger interface
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *noopLogger) Info(msg string, fields map[string]interface{})  {}
func (l *noopLogger) Warn(msg string, fields map[string]interface{})  {}
func (l *noopLogger) Error(msg string, fields map[string]interface{}) {}

// BasicHealthChecker provides a simple implementation of HealthChecker
type BasicHealthChecker struct {
	checkFn func(*ServiceInfo) (HealthCheckData, error)
	checkers map[string]*time.Ticker
	stopChan map[string]chan struct{}
	interval time.Duration
	mutex    sync.Mutex
	logger   Logger
}

// NewBasicHealthChecker creates a new basic health checker
func NewBasicHealthChecker(checkFn func(*ServiceInfo) (HealthCheckData, error), interval time.Duration, logger Logger) *BasicHealthChecker {
	if logger == nil {
		logger = &noopLogger{}
	}
	
	return &BasicHealthChecker{
		checkFn:   checkFn,
		checkers:  make(map[string]*time.Ticker),
		stopChan:  make(map[string]chan struct{}),
		interval:  interval,
		logger:    logger,
	}
}

// CheckHealth performs a health check on a service
func (c *BasicHealthChecker) CheckHealth(service *ServiceInfo) (HealthCheckData, error) {
	if c.checkFn == nil {
		return HealthCheckData{
			Status:      service.Status,
			LastChecked: time.Now(),
		}, nil
	}
	
	return c.checkFn(service)
}

// StartChecking starts periodic health checking for a service
func (c *BasicHealthChecker) StartChecking(service *ServiceInfo) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if _, exists := c.checkers[service.ID]; exists {
		return fmt.Errorf("health checker already running for service %s", service.ID)
	}
	
	ticker := time.NewTicker(c.interval)
	stopChan := make(chan struct{})
	c.checkers[service.ID] = ticker
	c.stopChan[service.ID] = stopChan
	
	go func() {
		for {
			select {
			case <-ticker.C:
				health, err := c.CheckHealth(service)
				if err != nil {
					c.logger.Warn("Health check failed", map[string]interface{}{
						"service_id":   service.ID,
						"service_name": service.Name,
						"error":        err.Error(),
					})
				} else {
					// This would normally update the registry, but for the prototype
					// we'll just log the health status
					c.logger.Debug("Health check completed", map[string]interface{}{
						"service_id":   service.ID,
						"service_name": service.Name,
						"status":       health.Status,
					})
				}
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()
	
	return nil
}

// StopChecking stops health checking for a service
func (c *BasicHealthChecker) StopChecking(serviceID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	stopChan, exists := c.stopChan[serviceID]
	if !exists {
		return fmt.Errorf("no health checker running for service %s", serviceID)
	}
	
	close(stopChan)
	delete(c.checkers, serviceID)
	delete(c.stopChan, serviceID)
	
	return nil
}
