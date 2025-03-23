package dashboard

import (
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemCollector collects system-wide metrics
type SystemCollector struct {
	mu sync.RWMutex

	// System metrics
	cpuUsage    float64
	memoryUsage float64
	numGoroutine int

	// Collection configuration
	interval time.Duration
	stopChan chan struct{}
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector(interval time.Duration) *SystemCollector {
	return &SystemCollector{
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start begins collecting system metrics
func (sc *SystemCollector) Start() {
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sc.collect()
		case <-sc.stopChan:
			return
		}
	}
}

// Stop stops the metrics collection
func (sc *SystemCollector) Stop() {
	close(sc.stopChan)
}

// GetMetrics returns the current system metrics
func (sc *SystemCollector) GetMetrics() SystemMetrics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return SystemMetrics{
		CPUUsage:     sc.cpuUsage,
		MemoryUsage:  sc.memoryUsage,
		NumGoroutine: sc.numGoroutine,
		Timestamp:    time.Now(),
	}
}

// collect gathers system metrics
func (sc *SystemCollector) collect() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Collect CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		sc.cpuUsage = cpuPercent[0]
	}

	// Collect memory usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		sc.memoryUsage = memInfo.UsedPercent
	}

	// Collect Go runtime metrics
	sc.numGoroutine = runtime.NumGoroutine()
}

// ServiceCollector collects service-specific metrics
type ServiceCollector struct {
	mu sync.RWMutex

	// Service metrics
	services map[string]*ServiceMetrics

	// Collection configuration
	retentionPeriod time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
}

// NewServiceCollector creates a new service metrics collector
func NewServiceCollector(retentionPeriod, cleanupInterval time.Duration) *ServiceCollector {
	return &ServiceCollector{
		services:        make(map[string]*ServiceMetrics),
		retentionPeriod: retentionPeriod,
		cleanupInterval: cleanupInterval,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the service metrics collection and cleanup
func (sc *ServiceCollector) Start() {
	ticker := time.NewTicker(sc.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sc.cleanup()
		case <-sc.stopChan:
			return
		}
	}
}

// Stop stops the metrics collection
func (sc *ServiceCollector) Stop() {
	close(sc.stopChan)
}

// RecordServiceMetric records metrics for a service
func (sc *ServiceCollector) RecordServiceMetric(serviceID string, metrics *ServiceMetrics) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.services[serviceID] = metrics
}

// GetServiceMetrics returns metrics for a specific service
func (sc *ServiceCollector) GetServiceMetrics(serviceID string) *ServiceMetrics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.services[serviceID]
}

// GetAllServiceMetrics returns metrics for all services
func (sc *ServiceCollector) GetAllServiceMetrics() map[string]*ServiceMetrics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	metrics := make(map[string]*ServiceMetrics)
	for id, service := range sc.services {
		metrics[id] = service
	}
	return metrics
}

// cleanup removes old metrics data
func (sc *ServiceCollector) cleanup() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	threshold := time.Now().Add(-sc.retentionPeriod)
	for id, metrics := range sc.services {
		if metrics.LastSeen.Before(threshold) {
			delete(sc.services, id)
		}
	}
}

// RouteCollector collects proxy route metrics
type RouteCollector struct {
	mu sync.RWMutex

	// Route metrics
	routes map[string]*RouteMetrics

	// Collection configuration
	retentionPeriod time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
}

// NewRouteCollector creates a new route metrics collector
func NewRouteCollector(retentionPeriod, cleanupInterval time.Duration) *RouteCollector {
	return &RouteCollector{
		routes:          make(map[string]*RouteMetrics),
		retentionPeriod: retentionPeriod,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the route metrics collection and cleanup
func (rc *RouteCollector) Start() {
	ticker := time.NewTicker(rc.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rc.cleanup()
		case <-rc.stopChan:
			return
		}
	}
}

// Stop stops the metrics collection
func (rc *RouteCollector) Stop() {
	close(rc.stopChan)
}

// RecordRouteMetric records metrics for a route
func (rc *RouteCollector) RecordRouteMetric(routeID string, metrics *RouteMetrics) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.routes[routeID] = metrics
}

// GetRouteMetrics returns metrics for a specific route
func (rc *RouteCollector) GetRouteMetrics(routeID string) *RouteMetrics {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.routes[routeID]
}

// GetAllRouteMetrics returns metrics for all routes
func (rc *RouteCollector) GetAllRouteMetrics() map[string]*RouteMetrics {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	metrics := make(map[string]*RouteMetrics)
	for id, route := range rc.routes {
		metrics[id] = route
	}
	return metrics
}

// cleanup removes old metrics data
func (rc *RouteCollector) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	threshold := time.Now().Add(-rc.retentionPeriod)
	for id, metrics := range rc.routes {
		if metrics.LastAccessed.Before(threshold) {
			delete(rc.routes, id)
		}
	}
}
