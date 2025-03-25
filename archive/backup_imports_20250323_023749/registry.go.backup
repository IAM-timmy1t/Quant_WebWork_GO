// registry.go - Service registry implementation

package discovery

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// RegistryImpl implements the Registry interface
type RegistryImpl struct {
	services           map[string]*ServiceInstance
	serviceGroups      map[string]*ServiceGroup
	mutex              sync.RWMutex
	watchers           map[string][]chan *ServiceInstance
	watchersMutex      sync.RWMutex
	healthChecker      HealthChecker
	registrationHandlers []RegistrationHandler
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewRegistry creates a new service registry
func NewRegistry(healthChecker HealthChecker) *RegistryImpl {
	ctx, cancel := context.WithCancel(context.Background())
	
	registry := &RegistryImpl{
		services:      make(map[string]*ServiceInstance),
		serviceGroups: make(map[string]*ServiceGroup),
		watchers:      make(map[string][]chan *ServiceInstance),
		healthChecker: healthChecker,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	return registry
}

// Register registers a service instance
func (r *RegistryImpl) Register(instance *ServiceInstance, options *RegistrationOptions) error {
	if instance == nil {
		return ErrInvalidService
	}
	
	if instance.ID == "" || instance.Name == "" || instance.Address == "" {
		return ErrInvalidService
	}
	
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Check if already exists
	if _, exists := r.services[instance.ID]; exists {
		return ErrDuplicateService
	}
	
	// Set initial status if not specified
	if instance.Status == "" {
		if options != nil && options.InitialStatus != "" {
			instance.Status = options.InitialStatus
		} else {
			instance.Status = StatusStarting
		}
	}
	
	// Set registration time
	now := time.Now()
	instance.RegistrationTime = now
	instance.LastUpdatedTime = now
	
	// Store the service instance
	r.services[instance.ID] = instance
	
	// Add to service group
	if _, exists := r.serviceGroups[instance.Name]; !exists {
		r.serviceGroups[instance.Name] = &ServiceGroup{
			Name:      instance.Name,
			Instances: make([]*ServiceInstance, 0),
		}
	}
	r.serviceGroups[instance.Name].Instances = append(r.serviceGroups[instance.Name].Instances, instance)
	
	// Start health checking if configured
	if r.healthChecker != nil && options != nil && options.HealthCheckInterval > 0 {
		if err := r.healthChecker.StartMonitoring(instance, options.HealthCheckInterval); err != nil {
			return fmt.Errorf("failed to start health monitoring: %w", err)
		}
	}
	
	// Notify watchers
	r.notifyWatchers(instance)
	
	// Call registration handlers
	for _, handler := range r.registrationHandlers {
		go handler(instance, true)
	}
	
	// Setup TTL and auto-renewal if configured
	if options != nil && options.TTL > 0 {
		if options.AutoRenew {
			go r.autoRenew(instance.ID, options.TTL)
		} else {
			go r.scheduleDeregistration(instance.ID, options.TTL)
		}
	}
	
	return nil
}

// autoRenew periodically renews a service registration
func (r *RegistryImpl) autoRenew(serviceID string, ttl time.Duration) {
	ticker := time.NewTicker(ttl / 2)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if err := r.Renew(serviceID); err != nil {
				// Service may have been deregistered
				return
			}
		}
	}
}

// scheduleDeregistration automatically deregisters a service after TTL expires
func (r *RegistryImpl) scheduleDeregistration(serviceID string, ttl time.Duration) {
	select {
	case <-r.ctx.Done():
		return
	case <-time.After(ttl):
		r.Deregister(serviceID)
	}
}

// Deregister removes a service instance
func (r *RegistryImpl) Deregister(serviceID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}
	
	// Stop health checking
	if r.healthChecker != nil {
		r.healthChecker.StopMonitoring(serviceID)
	}
	
	// Remove from service group
	if group, exists := r.serviceGroups[instance.Name]; exists {
		for i, svc := range group.Instances {
			if svc.ID == serviceID {
				group.Instances = append(group.Instances[:i], group.Instances[i+1:]...)
				break
			}
		}
		
		// If no instances left, remove the group
		if len(group.Instances) == 0 {
			delete(r.serviceGroups, instance.Name)
		}
	}
	
	// Remove the service
	delete(r.services, serviceID)
	
	// Call registration handlers
	for _, handler := range r.registrationHandlers {
		go handler(instance, false)
	}
	
	// Notify watchers of deregistration
	instance.Status = StatusDown
	r.notifyWatchers(instance)
	
	return nil
}

// Renew refreshes a service registration
func (r *RegistryImpl) Renew(serviceID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}
	
	instance.LastUpdatedTime = time.Now()
	return nil
}

// GetService finds all instances of a service by name
func (r *RegistryImpl) GetService(name string) (*ServiceGroup, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	group, exists := r.serviceGroups[name]
	if !exists {
		return nil, ErrServiceNotFound
	}
	
	// Return a copy to prevent concurrent modification
	groupCopy := &ServiceGroup{
		Name:      group.Name,
		Instances: make([]*ServiceInstance, len(group.Instances)),
	}
	
	copy(groupCopy.Instances, group.Instances)
	return groupCopy, nil
}

// GetServiceByID finds a specific service instance by ID
func (r *RegistryImpl) GetServiceByID(serviceID string) (*ServiceInstance, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return nil, ErrServiceNotFound
	}
	
	// Return a copy to prevent concurrent modification
	instanceCopy := *instance
	return &instanceCopy, nil
}

// GetServices returns all registered services
func (r *RegistryImpl) GetServices() ([]*ServiceGroup, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	groups := make([]*ServiceGroup, 0, len(r.serviceGroups))
	for _, group := range r.serviceGroups {
		// Create a copy
		groupCopy := &ServiceGroup{
			Name:      group.Name,
			Instances: make([]*ServiceInstance, len(group.Instances)),
		}
		copy(groupCopy.Instances, group.Instances)
		groups = append(groups, groupCopy)
	}
	
	return groups, nil
}

// Query searches for services based on criteria
func (r *RegistryImpl) Query(query *ServiceQuery) ([]*ServiceInstance, error) {
	if query == nil {
		return nil, ErrInvalidService
	}
	
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var results []*ServiceInstance
	
	// If name is specified, only look in that service group
	if query.Name != "" {
		group, exists := r.serviceGroups[query.Name]
		if !exists {
			return nil, ErrServiceNotFound
		}
		
		for _, instance := range group.Instances {
			if r.matchesQuery(instance, query) {
				instanceCopy := *instance
				results = append(results, &instanceCopy)
			}
		}
	} else {
		// Search across all services
		for _, instance := range r.services {
			if r.matchesQuery(instance, query) {
				instanceCopy := *instance
				results = append(results, &instanceCopy)
			}
		}
	}
	
	return results, nil
}

// matchesQuery checks if a service instance matches a query
func (r *RegistryImpl) matchesQuery(instance *ServiceInstance, query *ServiceQuery) bool {
	// Check status
	if query.Status != "" && instance.Status != query.Status {
		return false
	}
	
	// Check tags
	if len(query.Tags) > 0 {
		// Each tag in the query must be present in the instance
		tagMatches := make(map[string]bool)
		for _, tag := range instance.Tags {
			tagMatches[tag] = true
		}
		
		for _, tag := range query.Tags {
			if !tagMatches[tag] {
				return false
			}
		}
	}
	
	// Check metadata
	if len(query.MetadataFilters) > 0 {
		for key, value := range query.MetadataFilters {
			instanceValue, exists := instance.Metadata[key]
			if !exists || instanceValue != value {
				return false
			}
		}
	}
	
	// Check version constraints if implemented
	// This would require semantic version parsing
	
	return true
}

// Watch monitors service changes and sends updates
func (r *RegistryImpl) Watch(serviceNamePattern string) (<-chan *ServiceInstance, error) {
	r.watchersMutex.Lock()
	defer r.watchersMutex.Unlock()
	
	ch := make(chan *ServiceInstance, 10)
	
	// Compile the pattern
	var pattern *regexp.Regexp
	var err error
	if serviceNamePattern != "" {
		pattern, err = regexp.Compile(serviceNamePattern)
		if err != nil {
			return nil, fmt.Errorf("invalid service name pattern: %w", err)
		}
	}
	
	// Store the watcher
	patternKey := serviceNamePattern
	if patternKey == "" {
		patternKey = "*" // Watch all
	}
	
	r.watchers[patternKey] = append(r.watchers[patternKey], ch)
	
	// Send initial state
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	for _, instance := range r.services {
		if pattern == nil || pattern.MatchString(instance.Name) {
			select {
			case ch <- instance:
				// sent
			default:
				// channel full, skip
			}
		}
	}
	
	return ch, nil
}

// UpdateStatus updates a service's operational status
func (r *RegistryImpl) UpdateStatus(serviceID string, status ServiceStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}
	
	if instance.Status != status {
		instance.Status = status
		instance.LastUpdatedTime = time.Now()
		
		// Notify watchers
		r.notifyWatchers(instance)
	}
	
	return nil
}

// UpdateMetadata updates a service's metadata
func (r *RegistryImpl) UpdateMetadata(serviceID string, metadata ServiceMetadata) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	instance, exists := r.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}
	
	// Create a new metadata map if none exists
	if instance.Metadata == nil {
		instance.Metadata = make(ServiceMetadata)
	}
	
	// Update metadata
	for key, value := range metadata {
		instance.Metadata[key] = value
	}
	
	instance.LastUpdatedTime = time.Now()
	
	// Notify watchers
	r.notifyWatchers(instance)
	
	return nil
}

// AddRegistrationHandler adds a handler to be called on registration events
func (r *RegistryImpl) AddRegistrationHandler(handler RegistrationHandler) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.registrationHandlers = append(r.registrationHandlers, handler)
}

// notifyWatchers notifies all watchers of service changes
func (r *RegistryImpl) notifyWatchers(instance *ServiceInstance) {
	r.watchersMutex.RLock()
	defer r.watchersMutex.RUnlock()
	
	// Make a copy of the instance
	instanceCopy := *instance
	
	// Notify all exact name watchers
	if channels, exists := r.watchers[instance.Name]; exists {
		for _, ch := range channels {
			select {
			case ch <- &instanceCopy:
				// sent
			default:
				// channel full, skip
			}
		}
	}
	
	// Notify pattern watchers
	for pattern, channels := range r.watchers {
		// Skip exact name matches (already handled)
		if pattern == instance.Name {
			continue
		}
		
		// Wildcard
		if pattern == "*" {
			for _, ch := range channels {
				select {
				case ch <- &instanceCopy:
					// sent
				default:
					// channel full, skip
				}
			}
			continue
		}
		
		// Regex patterns
		if regexp, err := regexp.Compile(pattern); err == nil && regexp.MatchString(instance.Name) {
			for _, ch := range channels {
				select {
				case ch <- &instanceCopy:
					// sent
				default:
					// channel full, skip
				}
			}
		}
	}
}

// Stop stops the registry
func (r *RegistryImpl) Stop() {
	r.cancel()
	
	// Close all watcher channels
	r.watchersMutex.Lock()
	for _, channels := range r.watchers {
		for _, ch := range channels {
			close(ch)
		}
	}
	r.watchers = make(map[string][]chan *ServiceInstance)
	r.watchersMutex.Unlock()
}
