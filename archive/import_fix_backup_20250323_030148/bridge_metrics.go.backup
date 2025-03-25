package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BridgeMetrics encapsulates all metrics for the bridge module
type BridgeMetrics struct {
	// Connection metrics
	ActiveConnections    prometheus.Gauge
	ConnectionsTotal     prometheus.Counter
	ConnectionFailures   prometheus.Counter
	
	// Message metrics
	MessagesReceivedTotal   *prometheus.CounterVec
	MessagesSentTotal       *prometheus.CounterVec
	MessageProcessingTime   *prometheus.HistogramVec
	MessageErrors           *prometheus.CounterVec
	
	// Token analysis metrics
	TokenAnalysisRequests   prometheus.Counter
	TokenAnalysisErrors     prometheus.Counter
	TokenAnalysisTime       prometheus.Histogram
	
	// Performance metrics
	GoroutinesCount      prometheus.Gauge
	MemoryUsage          prometheus.Gauge
	CPUUsage             prometheus.Gauge
	
	// Server metrics
	ServerUptime         prometheus.Counter
	ServerStartTime      time.Time
}

// NewBridgeMetrics initializes and registers bridge metrics with Prometheus
func NewBridgeMetrics(name string) *BridgeMetrics {
	metrics := &BridgeMetrics{
		ServerStartTime: time.Now(),
	}
	
	prefix := "bridge"
	if name != "" {
		prefix = name
	}
	
	// Connection metrics
	metrics.ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: prefix + "_active_connections",
		Help: "Number of active bridge connections",
	})
	
	metrics.ConnectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: prefix + "_connections_total",
		Help: "Total number of bridge connections established",
	})
	
	metrics.ConnectionFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: prefix + "_connection_failures_total",
		Help: "Total number of failed bridge connection attempts",
	})
	
	// Message metrics
	metrics.MessagesReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_messages_received_total",
			Help: "Total number of messages received by the bridge",
		},
		[]string{"type"},
	)
	
	metrics.MessagesSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_messages_sent_total",
			Help: "Total number of messages sent by the bridge",
		},
		[]string{"type"},
	)
	
	metrics.MessageProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_message_processing_time_seconds",
			Help:    "Time taken to process bridge messages",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // from 1ms to ~1s
		},
		[]string{"type"},
	)
	
	metrics.MessageErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_message_errors_total",
			Help: "Total number of message errors by type",
		},
		[]string{"error_type"},
	)
	
	// Token analysis metrics
	metrics.TokenAnalysisRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: prefix + "_token_analysis_requests_total",
		Help: "Total number of token analysis requests",
	})
	
	metrics.TokenAnalysisErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: prefix + "_token_analysis_errors_total",
		Help: "Total number of token analysis errors",
	})
	
	metrics.TokenAnalysisTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    prefix + "_token_analysis_time_seconds",
		Help:    "Time taken to analyze tokens",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
	})
	
	// Performance metrics
	metrics.GoroutinesCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: prefix + "_goroutines_count",
		Help: "Number of goroutines running in the bridge",
	})
	
	metrics.MemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: prefix + "_memory_usage_bytes",
		Help: "Memory usage of the bridge in bytes",
	})
	
	metrics.CPUUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: prefix + "_cpu_usage_percent",
		Help: "CPU usage of the bridge in percent",
	})
	
	// Server metrics
	metrics.ServerUptime = promauto.NewCounter(prometheus.CounterOpts{
		Name: prefix + "_server_uptime_seconds",
		Help: "Uptime of the bridge server in seconds",
	})
	
	return metrics
}

// RecordMessageReceived records a message received metric
func (m *BridgeMetrics) RecordMessageReceived(messageType string) {
	if m.MessagesReceivedTotal != nil {
		m.MessagesReceivedTotal.WithLabelValues(messageType).Inc()
	}
}

// RecordMessageSent records a message sent metric
func (m *BridgeMetrics) RecordMessageSent(messageType string) {
	if m.MessagesSentTotal != nil {
		m.MessagesSentTotal.WithLabelValues(messageType).Inc()
	}
}

// RecordMessageProcessingTime records message processing time
func (m *BridgeMetrics) RecordMessageProcessingTime(messageType string, duration time.Duration) {
	if m.MessageProcessingTime != nil {
		m.MessageProcessingTime.WithLabelValues(messageType).Observe(duration.Seconds())
	}
}

// RecordMessageError records a message error
func (m *BridgeMetrics) RecordMessageError(errorType string) {
	if m.MessageErrors != nil {
		m.MessageErrors.WithLabelValues(errorType).Inc()
	}
}

// RecordError records an error with the given error type
// This is a convenience method that maps to RecordMessageError
func (m *BridgeMetrics) RecordError(errorType string) {
	m.RecordMessageError(errorType)
}

// UpdateGoroutinesCount updates the goroutines count metric
func (m *BridgeMetrics) UpdateGoroutinesCount(count int) {
	if m.GoroutinesCount != nil {
		m.GoroutinesCount.Set(float64(count))
	}
}

// UpdateMemoryUsage updates the memory usage metric
func (m *BridgeMetrics) UpdateMemoryUsage(bytes float64) {
	if m.MemoryUsage != nil {
		m.MemoryUsage.Set(bytes)
	}
}

// UpdateCPUUsage updates the CPU usage metric
func (m *BridgeMetrics) UpdateCPUUsage(percent float64) {
	if m.CPUUsage != nil {
		m.CPUUsage.Set(percent)
	}
}

// UpdateUptime updates the server uptime metric
func (m *BridgeMetrics) UpdateUptime() {
	if m.ServerUptime != nil {
		m.ServerUptime.Inc()
	}
}

// RecordConnectionOpened increments the active connections and total connections
func (m *BridgeMetrics) RecordConnectionOpened() {
	if m.ActiveConnections != nil {
		m.ActiveConnections.Inc()
	}
	if m.ConnectionsTotal != nil {
		m.ConnectionsTotal.Inc()
	}
}

// RecordConnectionClosed decrements the active connections
func (m *BridgeMetrics) RecordConnectionClosed() {
	if m.ActiveConnections != nil {
		m.ActiveConnections.Dec()
	}
}

// RecordConnectionFailure increments the connection failures counter
func (m *BridgeMetrics) RecordConnectionFailure() {
	if m.ConnectionFailures != nil {
		m.ConnectionFailures.Inc()
	}
}

// StartTokenAnalysis starts timing a token analysis operation
// Returns a function that should be called when the operation completes
func (m *BridgeMetrics) StartTokenAnalysis() func(error) {
	if m.TokenAnalysisRequests != nil {
		m.TokenAnalysisRequests.Inc()
	}
	startTime := time.Now()
	
	return func(err error) {
		duration := time.Since(startTime)
		if m.TokenAnalysisTime != nil {
			m.TokenAnalysisTime.Observe(duration.Seconds())
		}
		
		if err != nil {
			if m.TokenAnalysisErrors != nil {
				m.TokenAnalysisErrors.Inc()
			}
		}
	}
}

// GetMetricsSnapshot returns a snapshot of current metrics for API responses
func (m *BridgeMetrics) GetMetricsSnapshot() map[string]interface{} {
	uptime := time.Since(m.ServerStartTime)
	
	// Note: Prometheus doesn't easily expose the current values of metrics
	// In a production environment, you'd use a collector or the Prometheus HTTP API
	return map[string]interface{}{
		"uptime":           uptime.String(),
		"uptimeSeconds":    uptime.Seconds(),
		// We can't directly access Prometheus metric values, but we can provide metadata
		"metricsRegistered": true,
	}
}

/*
Helper methods commented out as Prometheus doesn't provide direct access to metric values
For a real implementation, consider using a separate counter or the Prometheus HTTP API

func (m *BridgeMetrics) getActiveConnectionsValue() float64 {
	if m.ActiveConnections != nil {
		// Prometheus doesn't expose current values directly
		return 0
	}
	return 0
}

func (m *BridgeMetrics) getConnectionsTotalValue() float64 {
	if m.ConnectionsTotal != nil {
		// Prometheus doesn't expose current values directly
		return 0
	}
	return 0
}

func (m *BridgeMetrics) getConnectionFailuresValue() float64 {
	if m.ConnectionFailures != nil {
		// Prometheus doesn't expose current values directly
		return 0
	}
	return 0
}

func (m *BridgeMetrics) getTokenAnalysisRequestsValue() float64 {
	if m.TokenAnalysisRequests != nil {
		// Prometheus doesn't expose current values directly
		return 0
	}
	return 0
}

func (m *BridgeMetrics) getTokenAnalysisErrorsValue() float64 {
	if m.TokenAnalysisErrors != nil {
		// Prometheus doesn't expose current values directly
		return 0
	}
	return 0
}
*/
