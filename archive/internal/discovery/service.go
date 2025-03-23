package discovery

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ServiceMetadata contains additional service information
type ServiceMetadata struct {
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Tags        []string          `json:"tags"`
	Properties  map[string]string `json:"properties"`
}

// Service represents a discoverable service
type Service struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Address     string          `json:"address"`
	Port        int             `json:"port"`
	Protocol    string          `json:"protocol"`
	Status      string          `json:"status"`
	Metadata    ServiceMetadata `json:"metadata"`
	LastUpdated time.Time       `json:"lastUpdated"`
	TTL         time.Duration   `json:"ttl"`
}

// ServiceRegistry manages service registration and discovery
type ServiceRegistry struct {
	services map[string]*Service
	mu       sync.RWMutex

	// Channels for service events
	registerChan   chan *Service
	deregisterChan chan string
	updateChan     chan *Service
	
	// Subscribers for service events
	subscribers map[chan ServiceEvent]struct{}
	subMu       sync.RWMutex
}

// ServiceEvent represents a service state change
type ServiceEvent struct {
	Type    string   `json:"type"`
	Service *Service `json:"service"`
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	registry := &ServiceRegistry{
		services:       make(map[string]*Service),
		registerChan:   make(chan *Service),
		deregisterChan: make(chan string),
		updateChan:     make(chan *Service),
		subscribers:    make(map[chan ServiceEvent]struct{}),
	}

	go registry.processEvents()
	go registry.cleanupExpiredServices()

	return registry
}

// Register adds a new service to the registry
func (sr *ServiceRegistry) Register(service *Service) error {
	if service.ID == "" {
		service.ID = uuid.New().String()
	}

	if service.TTL == 0 {
		service.TTL = 30 * time.Second
	}

	service.LastUpdated = time.Now()
	service.Status = "healthy"

	sr.registerChan <- service
	return nil
}

// Deregister removes a service from the registry
func (sr *ServiceRegistry) Deregister(serviceID string) error {
	sr.deregisterChan <- serviceID
	return nil
}

// Update updates an existing service in the registry
func (sr *ServiceRegistry) Update(service *Service) error {
	service.LastUpdated = time.Now()
	sr.updateChan <- service
	return nil
}

// GetService retrieves a service by ID
func (sr *ServiceRegistry) GetService(serviceID string) (*Service, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	service, exists := sr.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return service, nil
}

// GetServicesByName retrieves all services with a given name
func (sr *ServiceRegistry) GetServicesByName(name string) []*Service {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	var matches []*Service
	for _, service := range sr.services {
		if service.Name == name {
			matches = append(matches, service)
		}
	}

	return matches
}

// Subscribe creates a new subscription for service events
func (sr *ServiceRegistry) Subscribe() chan ServiceEvent {
	ch := make(chan ServiceEvent, 100)
	
	sr.subMu.Lock()
	sr.subscribers[ch] = struct{}{}
	sr.subMu.Unlock()

	return ch
}

// Unsubscribe removes a subscription
func (sr *ServiceRegistry) Unsubscribe(ch chan ServiceEvent) {
	sr.subMu.Lock()
	delete(sr.subscribers, ch)
	sr.subMu.Unlock()
	close(ch)
}

// processEvents handles service events
func (sr *ServiceRegistry) processEvents() {
	for {
		select {
		case service := <-sr.registerChan:
			sr.mu.Lock()
			sr.services[service.ID] = service
			sr.mu.Unlock()
			sr.publishEvent("register", service)

		case serviceID := <-sr.deregisterChan:
			sr.mu.Lock()
			if service, exists := sr.services[serviceID]; exists {
				delete(sr.services, serviceID)
				sr.mu.Unlock()
				sr.publishEvent("deregister", service)
			} else {
				sr.mu.Unlock()
			}

		case service := <-sr.updateChan:
			sr.mu.Lock()
			if _, exists := sr.services[service.ID]; exists {
				sr.services[service.ID] = service
				sr.mu.Unlock()
				sr.publishEvent("update", service)
			} else {
				sr.mu.Unlock()
			}
		}
	}
}

// publishEvent sends an event to all subscribers
func (sr *ServiceRegistry) publishEvent(eventType string, service *Service) {
	event := ServiceEvent{
		Type:    eventType,
		Service: service,
	}

	sr.subMu.RLock()
	defer sr.subMu.RUnlock()

	for ch := range sr.subscribers {
		select {
		case ch <- event:
		default:
			// Channel is full, skip this subscriber
		}
	}
}

// cleanupExpiredServices removes services that haven't been updated within their TTL
func (sr *ServiceRegistry) cleanupExpiredServices() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		
		sr.mu.Lock()
		for id, service := range sr.services {
			if now.Sub(service.LastUpdated) > service.TTL {
				delete(sr.services, id)
				sr.publishEvent("expire", service)
			}
		}
		sr.mu.Unlock()
	}
}
