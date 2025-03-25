// types.go - Service discovery type definitions

package discovery

import (
	"errors"
	"net/url"
	"time"
)

// ServiceStatus represents the current status of a service
type ServiceStatus string

const (
	// StatusUp indicates the service is operational
	StatusUp ServiceStatus = "UP"
	
	// StatusDown indicates the service is not operational
	StatusDown ServiceStatus = "DOWN"
	
	// StatusDegraded indicates the service is operational with reduced functionality
	StatusDegraded ServiceStatus = "DEGRADED"
	
	// StatusMaintenance indicates the service is in maintenance mode
	StatusMaintenance ServiceStatus = "MAINTENANCE"
	
	// StatusStarting indicates the service is starting up
	StatusStarting ServiceStatus = "STARTING"
	
	// StatusStopping indicates the service is shutting down
	StatusStopping ServiceStatus = "STOPPING"
	
	// StatusUnknown indicates the service status cannot be determined
	StatusUnknown ServiceStatus = "UNKNOWN"
)

// ServiceMetadata stores additional information about a service
type ServiceMetadata map[string]string

// ServiceInstance represents a single instance of a service
type ServiceInstance struct {
	// ID is a unique identifier for this service instance
	ID string
	
	// Name is the service name
	Name string
	
	// Version is the service version
	Version string
	
	// Address is the service network address (host:port)
	Address string
	
	// Status represents the current operational status
	Status ServiceStatus
	
	// Metadata contains additional service information
	Metadata ServiceMetadata
	
	// HealthCheckURL is the URL for health checks
	HealthCheckURL *url.URL
	
	// RegistrationTime is when the service was registered
	RegistrationTime time.Time
	
	// LastUpdatedTime is when the service information was last updated
	LastUpdatedTime time.Time
	
	// Tags are used for categorization and filtering
	Tags []string
	
	// Weight is used for load balancing (higher values get more traffic)
	Weight int
}

// ServiceGroup represents a group of related service instances
type ServiceGroup struct {
	// Name is the service group name
	Name string
	
	// Instances are the service instances in this group
	Instances []*ServiceInstance
}

// ServiceQuery defines search criteria for service discovery
type ServiceQuery struct {
	// Name is the service name to search for
	Name string
	
	// Tags are the tags that services must have
	Tags []string
	
	// Status filters services by status
	Status ServiceStatus
	
	// MinVersion specifies the minimum allowed version
	MinVersion string
	
	// MaxVersion specifies the maximum allowed version
	MaxVersion string
	
	// MetadataFilters are key-value pairs that services must match
	MetadataFilters ServiceMetadata
}

// RegistrationOptions defines options for service registration
type RegistrationOptions struct {
	// TTL is the time-to-live for a registration
	TTL time.Duration
	
	// AutoRenew automatically renews registration before expiration
	AutoRenew bool
	
	// HealthCheckInterval is how often to perform health checks
	HealthCheckInterval time.Duration
	
	// InitialStatus is the starting status for the service
	InitialStatus ServiceStatus
	
	// DeregistrationDelay is how long to wait before deregistering
	DeregistrationDelay time.Duration
}

// Registry defines the interface for service registration and discovery
type Registry interface {
	// Register registers a service instance
	Register(*ServiceInstance, *RegistrationOptions) error
	
	// Deregister removes a service instance
	Deregister(serviceID string) error
	
	// Renew refreshes a service registration
	Renew(serviceID string) error
	
	// GetService finds all instances of a service by name
	GetService(name string) (*ServiceGroup, error)
	
	// GetServiceByID finds a specific service instance by ID
	GetServiceByID(serviceID string) (*ServiceInstance, error)
	
	// GetServices returns all registered services
	GetServices() ([]*ServiceGroup, error)
	
	// Query searches for services based on criteria
	Query(*ServiceQuery) ([]*ServiceInstance, error)
	
	// Watch monitors service changes and sends updates
	Watch(serviceNamePattern string) (<-chan *ServiceInstance, error)
	
	// UpdateStatus updates a service's operational status
	UpdateStatus(serviceID string, status ServiceStatus) error
	
	// UpdateMetadata updates a service's metadata
	UpdateMetadata(serviceID string, metadata ServiceMetadata) error
}

// Common errors
var (
	ErrServiceNotFound    = errors.New("service not found")
	ErrDuplicateService   = errors.New("service already registered")
	ErrInvalidService     = errors.New("invalid service definition")
	ErrRegistryNotReady   = errors.New("service registry not ready")
	ErrRegistrationFailed = errors.New("service registration failed")
)

// HealthChecker defines the interface for service health checking
type HealthChecker interface {
	// CheckHealth performs a health check on a service
	CheckHealth(*ServiceInstance) (ServiceStatus, error)
	
	// StartMonitoring begins periodic health checks
	StartMonitoring(*ServiceInstance, time.Duration) error
	
	// StopMonitoring stops periodic health checks
	StopMonitoring(serviceID string) error
	
	// SetHealthCheckHandler sets a custom health check function
	SetHealthCheckHandler(serviceID string, handler func() (ServiceStatus, error)) error
}

// RegistrationHandler is called when service registration changes
type RegistrationHandler func(*ServiceInstance, bool) // (instance, isRegistration)

// Constants for default values
const (
	DefaultTTL               = 60 * time.Second
	DefaultHealthCheckPath   = "/health"
	DefaultRefreshInterval   = 15 * time.Second
	DefaultDeregistrationDelay = 30 * time.Second
)
