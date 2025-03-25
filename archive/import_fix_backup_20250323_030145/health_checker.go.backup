// health_checker.go - Service health monitoring implementation

package discovery

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthCheckMode defines how health checks are performed
type HealthCheckMode int

const (
	// ModeHTTP performs health checks via HTTP
	ModeHTTP HealthCheckMode = iota
	
	// ModeCustom uses a custom health check function
	ModeCustom
)

// healthCheckMonitor tracks a single service's health
type healthCheckMonitor struct {
	instance    *ServiceInstance
	mode        HealthCheckMode
	interval    time.Duration
	handler     func() (ServiceStatus, error)
	lastCheck   time.Time
	lastStatus  ServiceStatus
	stopChan    chan struct{}
	registry    Registry
}

// HealthCheckerImpl implements the HealthChecker interface
type HealthCheckerImpl struct {
	monitors       map[string]*healthCheckMonitor
	mutex          sync.RWMutex
	client         *http.Client
	registry       Registry
	defaultTimeout time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration) *HealthCheckerImpl {
	ctx, cancel := context.WithCancel(context.Background())
	
	checker := &HealthCheckerImpl{
		monitors:       make(map[string]*healthCheckMonitor),
		client:         &http.Client{Timeout: timeout},
		defaultTimeout: timeout,
		ctx:            ctx,
		cancel:         cancel,
	}
	
	return checker
}

// SetRegistry sets the service registry to update status
func (h *HealthCheckerImpl) SetRegistry(registry Registry) {
	h.registry = registry
}

// CheckHealth performs a health check on a service
func (h *HealthCheckerImpl) CheckHealth(instance *ServiceInstance) (ServiceStatus, error) {
	if instance == nil {
		return StatusUnknown, ErrInvalidService
	}
	
	h.mutex.RLock()
	monitor, exists := h.monitors[instance.ID]
	h.mutex.RUnlock()
	
	if exists && monitor.mode == ModeCustom && monitor.handler != nil {
		return monitor.handler()
	}
	
	// Default to HTTP health check
	if instance.HealthCheckURL == nil {
		return StatusUnknown, fmt.Errorf("health check URL not defined")
	}
	
	resp, err := h.client.Get(instance.HealthCheckURL.String())
	if err != nil {
		return StatusDown, err
	}
	defer resp.Body.Close()
	
	// Map HTTP status to service status
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return StatusUp, nil
	case resp.StatusCode == http.StatusServiceUnavailable:
		return StatusDown, nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return StatusDegraded, nil
	case resp.StatusCode == http.StatusLocked:
		return StatusMaintenance, nil
	default:
		return StatusUnknown, fmt.Errorf("unexpected health check status: %d", resp.StatusCode)
	}
}

// StartMonitoring begins periodic health checks
func (h *HealthCheckerImpl) StartMonitoring(instance *ServiceInstance, interval time.Duration) error {
	if instance == nil || instance.ID == "" {
		return ErrInvalidService
	}
	
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if _, exists := h.monitors[instance.ID]; exists {
		return fmt.Errorf("already monitoring service %s", instance.ID)
	}
	
	stopChan := make(chan struct{})
	
	monitor := &healthCheckMonitor{
		instance:  instance,
		mode:      ModeHTTP,
		interval:  interval,
		stopChan:  stopChan,
		registry:  h.registry,
		lastCheck: time.Time{},
	}
	
	h.monitors[instance.ID] = monitor
	
	// Start monitoring goroutine
	go h.monitorService(monitor)
	
	return nil
}

// StopMonitoring stops periodic health checks
func (h *HealthCheckerImpl) StopMonitoring(serviceID string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	monitor, exists := h.monitors[serviceID]
	if !exists {
		return ErrServiceNotFound
	}
	
	// Signal the monitoring goroutine to stop
	close(monitor.stopChan)
	delete(h.monitors, serviceID)
	
	return nil
}

// SetHealthCheckHandler sets a custom health check function
func (h *HealthCheckerImpl) SetHealthCheckHandler(serviceID string, handler func() (ServiceStatus, error)) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	monitor, exists := h.monitors[serviceID]
	if !exists {
		return ErrServiceNotFound
	}
	
	monitor.mode = ModeCustom
	monitor.handler = handler
	
	return nil
}

// monitorService periodically checks service health
func (h *HealthCheckerImpl) monitorService(monitor *healthCheckMonitor) {
	ticker := time.NewTicker(monitor.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-monitor.stopChan:
			return
		case <-ticker.C:
			status, err := h.CheckHealth(monitor.instance)
			
			// Only update if status changed or there was an error
			if err != nil || monitor.lastStatus != status {
				if h.registry != nil {
					h.registry.UpdateStatus(monitor.instance.ID, status)
				}
				
				// Update monitoring state
				h.mutex.Lock()
				if mon, exists := h.monitors[monitor.instance.ID]; exists {
					mon.lastStatus = status
					mon.lastCheck = time.Now()
				}
				h.mutex.Unlock()
			}
		}
	}
}

// Stop stops all health checkers
func (h *HealthCheckerImpl) Stop() {
	h.cancel()
	
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	// Stop all monitors
	for id, monitor := range h.monitors {
		close(monitor.stopChan)
		delete(h.monitors, id)
	}
}
