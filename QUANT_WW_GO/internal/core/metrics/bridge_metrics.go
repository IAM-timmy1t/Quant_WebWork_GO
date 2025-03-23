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
	MessagesReceivedTotal   prometheus.CounterVec
	MessagesSentTotal       prometheus.CounterVec
	MessageProcessingTime   prometheus.HistogramVec
	MessageErrors           prometheus.CounterVec
	
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
func NewBridgeMetrics() *BridgeMetrics {
	metrics := &BridgeMetrics{
		ServerStartTime: time.Now(),
	}
	
	// Connection metrics
	metrics.ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_active_connections",
		Help: "Number of active bridge connections",
	})
	
	metrics.ConnectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_connections_total",
		Help: "Total number of bridge connections established",
	})
	
	metrics.ConnectionFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_connection_failures_total",
		Help: "Total number of failed bridge connection attempts",
	})
	
	// Message metrics
	metrics.MessagesReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_messages_received_total",
			Help: "Total number of messages received by the bridge",
		},
		[]string{"type"},
	)
	
	metrics.MessagesSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_messages_sent_total",
			Help: "Total number of messages sent by the bridge",
		},
		[]string{"type"},
	)
	
	metrics.MessageProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_message_processing_time_seconds",
			Help:    "Time taken to process bridge messages",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // from 1ms to ~1s
		},
		[]string{"type"},
	)
	
	metrics.MessageErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_message_errors_total",
			Help: "Total number of message errors by type",
		},
		[]string{"error_type"},
	)
	
	// Token analysis metrics
	metrics.TokenAnalysisRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_token_analysis_requests_total",
		Help: "Total number of token analysis requests",
	})
	
	metrics.TokenAnalysisErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_token_analysis_errors_total",
		Help: "Total number of token analysis errors",
	})
	
	metrics.TokenAnalysisTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_token_analysis_time_seconds",
		Help:    "Time taken to analyze tokens",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // from 10ms to ~10s
	})
	
	// Performance metrics
	metrics.GoroutinesCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_goroutines_count",
		Help: "Number of goroutines in the bridge service",
	})
	
	metrics.MemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_memory_usage_bytes",
		Help: "Memory usage of the bridge service in bytes",
	})
	
	metrics.CPUUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_cpu_usage_percent",
		Help: "CPU usage of the bridge service as percentage",
	})
	
	// Server metrics
	metrics.ServerUptime = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_server_uptime_seconds",
		Help: "Uptime of the bridge server in seconds",
	})
	
	// Start a goroutine to update server metrics
	go metrics.startMetricsCollection()
	
	return metrics
}

// startMetricsCollection periodically updates metrics that need polling
func (m *BridgeMetrics) startMetricsCollection() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		// Update uptime
		m.ServerUptime.Add(10)
		
		// These would typically be populated with actual system metrics
		// For demonstration, we're using placeholder implementations
		m.updateResourceMetrics()
	}
}

// updateResourceMetrics collects and updates resource usage metrics
func (m *BridgeMetrics) updateResourceMetrics() {
	// In a real implementation, you would use runtime.NumGoroutine()
	// and get memory/CPU stats from runtime or from OS
	// This is a placeholder implementation
	
	// Example implementation:
	// m.GoroutinesCount.Set(float64(runtime.NumGoroutine()))
	
	// Memory and CPU stats would typically come from:
	// - runtime.ReadMemStats for Go memory stats
	// - os/process libraries for system-level CPU stats
}

// RecordMessageReceived records a message received by type
func (m *BridgeMetrics) RecordMessageReceived(messageType string) {
	m.MessagesReceivedTotal.WithLabelValues(messageType).Inc()
}

// RecordMessageSent records a message sent by type
func (m *BridgeMetrics) RecordMessageSent(messageType string) {
	m.MessagesSentTotal.WithLabelValues(messageType).Inc()
}

// ObserveMessageProcessingTime records the time taken to process a message
func (m *BridgeMetrics) ObserveMessageProcessingTime(messageType string, duration time.Duration) {
	m.MessageProcessingTime.WithLabelValues(messageType).Observe(duration.Seconds())
}

// RecordMessageError increments the error counter for a specific error type
func (m *BridgeMetrics) RecordMessageError(errorType string) {
	m.MessageErrors.WithLabelValues(errorType).Inc()
}

// RecordConnectionOpened increments the active connections and total connections
func (m *BridgeMetrics) RecordConnectionOpened() {
	m.ActiveConnections.Inc()
	m.ConnectionsTotal.Inc()
}

// RecordConnectionClosed decrements the active connections
func (m *BridgeMetrics) RecordConnectionClosed() {
	m.ActiveConnections.Dec()
}

// RecordConnectionFailure increments the connection failures counter
func (m *BridgeMetrics) RecordConnectionFailure() {
	m.ConnectionFailures.Inc()
}

// StartTokenAnalysis starts timing a token analysis operation
// Returns a function that should be called when the operation completes
func (m *BridgeMetrics) StartTokenAnalysis() func(error) {
	m.TokenAnalysisRequests.Inc()
	startTime := time.Now()
	
	return func(err error) {
		duration := time.Since(startTime)
		m.TokenAnalysisTime.Observe(duration.Seconds())
		
		if err != nil {
			m.TokenAnalysisErrors.Inc()
		}
	}
}

// GetMetricsSnapshot returns a snapshot of current metrics for API responses
func (m *BridgeMetrics) GetMetricsSnapshot() map[string]interface{} {
	// This would collect the current values of metrics
	// Note: Some metric types in Prometheus don't easily expose their current values
	// This is a simplified implementation
	
	uptime := time.Since(m.ServerStartTime)
	
	// Get total message counts
	var messagesReceived float64
	var messagesSent float64
	
	// In a real implementation, you would collect these from the actual metrics
	// For demonstration, we're using placeholder values
	
	return map[string]interface{}{
		"activeConnections":      m.getGaugeValue(m.ActiveConnections),
		"totalConnections":       m.getCounterValue(m.ConnectionsTotal),
		"connectionFailures":     m.getCounterValue(m.ConnectionFailures),
		"messagesReceived":       messagesReceived,
		"messagesSent":           messagesSent,
		"tokenAnalysisRequests":  m.getCounterValue(m.TokenAnalysisRequests),
		"tokenAnalysisErrors":    m.getCounterValue(m.TokenAnalysisErrors),
		"uptime":                 uptime.String(),
		"uptimeSeconds":          uptime.Seconds(),
	}
}

// Helper methods to extract values from Prometheus metrics
// Note: These are simplified implementations
func (m *BridgeMetrics) getGaugeValue(gauge prometheus.Gauge) float64 {
	// This is a simplification - in a real implementation you would need a different approach
	// as Prometheus metrics don't directly expose their current values in this way
	return 0
}

func (m *BridgeMetrics) getCounterValue(counter prometheus.Counter) float64 {
	// This is a simplification - in a real implementation you would need a different approach
	return 0
}
