package proxy

import (
	"sync"
	"time"
)

// MetricsCollector tracks proxy metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// Request metrics
	totalRequests     uint64
	successRequests   uint64
	failedRequests    uint64
	responseTimeTotal time.Duration
	responseTimeCount uint64

	// Target metrics
	targetMetrics map[string]*TargetMetrics
}

// TargetMetrics tracks per-target metrics
type TargetMetrics struct {
	mu sync.RWMutex

	requests          uint64
	errors            uint64
	responseTimeTotal time.Duration
	responseTimeCount uint64
	lastError         time.Time
	lastErrorMessage  string
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		targetMetrics: make(map[string]*TargetMetrics),
	}
}

// RecordRequest records metrics for a proxy request
func (m *MetricsCollector) RecordRequest(targetID string, duration time.Duration, err error) {
	m.mu.Lock()
	m.totalRequests++
	m.responseTimeTotal += duration
	m.responseTimeCount++
	if err != nil {
		m.failedRequests++
	} else {
		m.successRequests++
	}
	m.mu.Unlock()

	// Record target-specific metrics
	m.mu.RLock()
	metrics, exists := m.targetMetrics[targetID]
	m.mu.RUnlock()

	if !exists {
		m.mu.Lock()
		metrics = &TargetMetrics{}
		m.targetMetrics[targetID] = metrics
		m.mu.Unlock()
	}

	metrics.mu.Lock()
	metrics.requests++
	metrics.responseTimeTotal += duration
	metrics.responseTimeCount++
	if err != nil {
		metrics.errors++
		metrics.lastError = time.Now()
		metrics.lastErrorMessage = err.Error()
	}
	metrics.mu.Unlock()
}

// GetMetrics returns the current metrics
type Metrics struct {
	TotalRequests      uint64
	SuccessRequests    uint64
	FailedRequests     uint64
	AverageResponseTime time.Duration
	TargetMetrics      map[string]TargetMetricsData
}

type TargetMetricsData struct {
	Requests           uint64
	Errors             uint64
	AverageResponseTime time.Duration
	LastError          time.Time
	LastErrorMessage   string
}

func (m *MetricsCollector) GetMetrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgResponseTime time.Duration
	if m.responseTimeCount > 0 {
		avgResponseTime = m.responseTimeTotal / time.Duration(m.responseTimeCount)
	}

	metrics := Metrics{
		TotalRequests:      m.totalRequests,
		SuccessRequests:    m.successRequests,
		FailedRequests:     m.failedRequests,
		AverageResponseTime: avgResponseTime,
		TargetMetrics:      make(map[string]TargetMetricsData),
	}

	for id, target := range m.targetMetrics {
		target.mu.RLock()
		var targetAvgResponseTime time.Duration
		if target.responseTimeCount > 0 {
			targetAvgResponseTime = target.responseTimeTotal / time.Duration(target.responseTimeCount)
		}

		metrics.TargetMetrics[id] = TargetMetricsData{
			Requests:           target.requests,
			Errors:             target.errors,
			AverageResponseTime: targetAvgResponseTime,
			LastError:          target.lastError,
			LastErrorMessage:   target.lastErrorMessage,
		}
		target.mu.RUnlock()
	}

	return metrics
}

// ResetMetrics resets all metrics counters
func (m *MetricsCollector) ResetMetrics() {
	m.mu.Lock()
	m.totalRequests = 0
	m.successRequests = 0
	m.failedRequests = 0
	m.responseTimeTotal = 0
	m.responseTimeCount = 0
	m.targetMetrics = make(map[string]*TargetMetrics)
	m.mu.Unlock()
}
