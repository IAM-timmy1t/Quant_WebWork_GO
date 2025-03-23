package dashboard

import (
	"sync"
	"time"

	"github.com/timot/Quant_WebWork_GO/internal/discovery"
	"github.com/timot/Quant_WebWork_GO/internal/proxy"
)

// MetricsConfig defines configuration for metrics collection
type MetricsConfig struct {
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	SampleInterval  time.Duration `json:"sampleInterval"`
	DetailedMetrics bool          `json:"detailedMetrics"`
}

// MetricsCollector manages system-wide metrics collection
type MetricsCollector struct {
	mu sync.RWMutex

	// Configuration
	config MetricsConfig

	// System metrics
	systemMetrics []SystemMetric
	serviceMetrics map[string]*ServiceMetrics
	routeMetrics   map[string]*RouteMetrics

	// Logs
	logs []LogEntry
}

// SystemMetric represents a point-in-time system metric
type SystemMetric struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpuUsage"`
	MemoryUsage   float64   `json:"memoryUsage"`
	TotalRequests uint64    `json:"totalRequests"`
	ErrorRate     float64   `json:"errorRate"`
	HealthScore   float64   `json:"healthScore"`
}

// ServiceMetrics tracks metrics for a specific service
type ServiceMetrics struct {
	ServiceID      string        `json:"serviceId"`
	ServiceName    string        `json:"serviceName"`
	Status         string        `json:"status"`
	LastSeen       time.Time     `json:"lastSeen"`
	ResponseTimes  []float64     `json:"responseTimes"`
	RequestCount   uint64        `json:"requestCount"`
	ErrorCount     uint64        `json:"errorCount"`
	AvailableTime  time.Duration `json:"availableTime"`
	DownTime       time.Duration `json:"downTime"`
	LastError      time.Time     `json:"lastError"`
	LastErrorMsg   string        `json:"lastErrorMsg"`
}

// RouteMetrics tracks metrics for a proxy route
type RouteMetrics struct {
	RouteID        string    `json:"routeId"`
	Path           string    `json:"path"`
	RequestCount   uint64    `json:"requestCount"`
	ErrorCount     uint64    `json:"errorCount"`
	ResponseTimes  []float64 `json:"responseTimes"`
	LastAccessed   time.Time `json:"lastAccessed"`
	LastError      time.Time `json:"lastError"`
	LastErrorMsg   string    `json:"lastErrorMsg"`
}

// LogEntry represents a system log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		config: MetricsConfig{
			RetentionPeriod: 24 * time.Hour,
			SampleInterval:  10 * time.Second,
			DetailedMetrics: true,
		},
		serviceMetrics: make(map[string]*ServiceMetrics),
		routeMetrics:   make(map[string]*RouteMetrics),
		logs:          make([]LogEntry, 0),
	}
}

// Start begins metrics collection
func (mc *MetricsCollector) Start() {
	ticker := time.NewTicker(mc.config.SampleInterval)
	defer ticker.Stop()

	for range ticker.C {
		mc.collectMetrics()
		mc.cleanupOldMetrics()
	}
}

// RecordServiceMetric records metrics for a service
func (mc *MetricsCollector) RecordServiceMetric(service *discovery.Service, responseTime float64, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.serviceMetrics[service.ID]
	if !exists {
		metrics = &ServiceMetrics{
			ServiceID:   service.ID,
			ServiceName: service.Name,
		}
		mc.serviceMetrics[service.ID] = metrics
	}

	metrics.LastSeen = time.Now()
	metrics.Status = service.Status
	metrics.RequestCount++
	metrics.ResponseTimes = append(metrics.ResponseTimes, responseTime)

	if err != nil {
		metrics.ErrorCount++
		metrics.LastError = time.Now()
		metrics.LastErrorMsg = err.Error()
	}

	// Maintain response time history within retention period
	mc.trimMetricsHistory(metrics)
}

// RecordRouteMetric records metrics for a proxy route
func (mc *MetricsCollector) RecordRouteMetric(route *proxy.ProxyRoute, responseTime float64, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.routeMetrics[route.ID]
	if !exists {
		metrics = &RouteMetrics{
			RouteID: route.ID,
			Path:    route.Path,
		}
		mc.routeMetrics[route.ID] = metrics
	}

	metrics.LastAccessed = time.Now()
	metrics.RequestCount++
	metrics.ResponseTimes = append(metrics.ResponseTimes, responseTime)

	if err != nil {
		metrics.ErrorCount++
		metrics.LastError = time.Now()
		metrics.LastErrorMsg = err.Error()
	}

	// Maintain response time history within retention period
	mc.trimRouteMetricsHistory(metrics)
}

// AddLog adds a new log entry
func (mc *MetricsCollector) AddLog(level, source, message string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Source:    source,
		Message:   message,
	}

	mc.logs = append(mc.logs, entry)
}

// GetMetrics returns current system metrics
func (mc *MetricsCollector) GetMetrics() SystemMetric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.systemMetrics) == 0 {
		return SystemMetric{}
	}
	return mc.systemMetrics[len(mc.systemMetrics)-1]
}

// GetServiceMetrics returns metrics for a specific service
func (mc *MetricsCollector) GetServiceMetrics(serviceID string) *ServiceMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.serviceMetrics[serviceID]
}

// GetLogs returns system logs filtered by level
func (mc *MetricsCollector) GetLogs(level string, limit int) []LogEntry {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if level == "" {
		if limit > 0 && limit < len(mc.logs) {
			return mc.logs[len(mc.logs)-limit:]
		}
		return mc.logs
	}

	var filtered []LogEntry
	for i := len(mc.logs) - 1; i >= 0 && len(filtered) < limit; i-- {
		if mc.logs[i].Level == level {
			filtered = append(filtered, mc.logs[i])
		}
	}
	return filtered
}

// UpdateConfig updates the metrics configuration
func (mc *MetricsCollector) UpdateConfig(config MetricsConfig) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.config = config
}

// GetConfig returns the current metrics configuration
func (mc *MetricsCollector) GetConfig() MetricsConfig {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.config
}

// ClearLogs clears all system logs
func (mc *MetricsCollector) ClearLogs() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.logs = make([]LogEntry, 0)
}

// Internal helper methods

func (mc *MetricsCollector) collectMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Collect system metrics
	metric := SystemMetric{
		Timestamp: time.Now(),
		// Add system metric collection here
	}

	mc.systemMetrics = append(mc.systemMetrics, metric)
}

func (mc *MetricsCollector) cleanupOldMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.config.RetentionPeriod)

	// Clean up system metrics
	var newMetrics []SystemMetric
	for _, metric := range mc.systemMetrics {
		if metric.Timestamp.After(cutoff) {
			newMetrics = append(newMetrics, metric)
		}
	}
	mc.systemMetrics = newMetrics

	// Clean up service metrics
	for _, metrics := range mc.serviceMetrics {
		mc.trimMetricsHistory(metrics)
	}

	// Clean up route metrics
	for _, metrics := range mc.routeMetrics {
		mc.trimRouteMetricsHistory(metrics)
	}

	// Clean up logs
	var newLogs []LogEntry
	for _, log := range mc.logs {
		if log.Timestamp.After(cutoff) {
			newLogs = append(newLogs, log)
		}
	}
	mc.logs = newLogs
}

func (mc *MetricsCollector) trimMetricsHistory(metrics *ServiceMetrics) {
	cutoff := time.Now().Add(-mc.config.RetentionPeriod)
	if metrics.LastSeen.Before(cutoff) {
		metrics.ResponseTimes = nil
		return
	}
}

func (mc *MetricsCollector) trimRouteMetricsHistory(metrics *RouteMetrics) {
	cutoff := time.Now().Add(-mc.config.RetentionPeriod)
	if metrics.LastAccessed.Before(cutoff) {
		metrics.ResponseTimes = nil
		return
	}
}
