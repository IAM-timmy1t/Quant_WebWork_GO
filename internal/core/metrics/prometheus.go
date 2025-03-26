// prometheus.go - Prometheus integration for metrics

package metrics

import (
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusBridge connects the metrics system to Prometheus
type PrometheusBridge struct {
	counters        map[string]*prometheus.CounterVec
	gauges          map[string]*prometheus.GaugeVec
	histograms      map[string]*prometheus.HistogramVec
	summaries       map[string]*prometheus.SummaryVec
	mutex           sync.RWMutex
	defaultBuckets  []float64
	namespace       string
	subsystem       string
	collectors      map[string]prometheus.Collector
	internalRegister bool
}

// NewPrometheusBridge creates a new Prometheus bridge
func NewPrometheusBridge(namespace, subsystem string) *PrometheusBridge {
	return &PrometheusBridge{
		counters:        make(map[string]*prometheus.CounterVec),
		gauges:          make(map[string]*prometheus.GaugeVec),
		histograms:      make(map[string]*prometheus.HistogramVec),
		summaries:       make(map[string]*prometheus.SummaryVec),
		collectors:      make(map[string]prometheus.Collector),
		defaultBuckets:  prometheus.DefBuckets,
		namespace:       namespace,
		subsystem:       subsystem,
		internalRegister: true,
	}
}

// SetDefaultBuckets sets the default histogram buckets
func (pb *PrometheusBridge) SetDefaultBuckets(buckets []float64) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	pb.defaultBuckets = buckets
}

// RecordMetric sends a metric to Prometheus
func (pb *PrometheusBridge) RecordMetric(metric *Metric) error {
	switch metric.Type {
	case MetricTypeCounter:
		return pb.recordCounter(metric)
	case MetricTypeGauge:
		return pb.recordGauge(metric)
	case MetricTypeHistogram:
		return pb.recordHistogram(metric)
	case MetricTypeSummary:
		return pb.recordSummary(metric)
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.Type)
	}
}

// normalizeMetricName converts a metric name to Prometheus-compatible format
func (pb *PrometheusBridge) normalizeMetricName(name string) string {
	// Replace non-alphanumeric chars with underscores
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	
	// Ensure it starts with a letter
	if len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
		name = "m_" + name
	}
	
	return name
}

// normalizeLabels converts labels to Prometheus-compatible format
func (pb *PrometheusBridge) normalizeLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}
	
	result := make(map[string]string, len(labels))
	for k, v := range labels {
		normalizedKey := pb.normalizeMetricName(k)
		result[normalizedKey] = v
	}
	
	return result
}

// getLabelsFromMetric extracts label names from a metric's labels
func (pb *PrometheusBridge) getLabelsFromMetric(metric *Metric) []string {
	if metric.Labels == nil || len(metric.Labels) == 0 {
		return []string{}
	}
	
	labels := make([]string, 0, len(metric.Labels))
	for k := range metric.Labels {
		normalizedKey := pb.normalizeMetricName(k)
		labels = append(labels, normalizedKey)
	}
	
	return labels
}

// recordCounter handles Counter metrics
func (pb *PrometheusBridge) recordCounter(metric *Metric) error {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	name := pb.normalizeMetricName(metric.Name)
	counterKey := fmt.Sprintf("%s_%s", metric.Source, name)
	
	counter, exists := pb.counters[counterKey]
	if !exists {
		labelNames := pb.getLabelsFromMetric(metric)
		counter = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: pb.namespace,
				Subsystem: pb.subsystem,
				Name:      name,
				Help:      fmt.Sprintf("Counter metric: %s", name),
			},
			labelNames,
		)
		pb.counters[counterKey] = counter
		pb.collectors[counterKey] = counter
	}
	
	normalizedLabels := pb.normalizeLabels(metric.Labels)
	labelValues := make([]string, 0, len(normalizedLabels))
	for _, labelName := range pb.getLabelsFromMetric(metric) {
		labelValues = append(labelValues, normalizedLabels[labelName])
	}
	
	// Increment the counter by the metric value
	if len(labelValues) > 0 {
		counter.WithLabelValues(labelValues...).Add(metric.Value)
	} else {
		// If no labels, use MustCurry for efficiency
		counter.WithLabelValues().Add(metric.Value)
	}
	
	return nil
}

// recordGauge handles Gauge metrics
func (pb *PrometheusBridge) recordGauge(metric *Metric) error {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	name := pb.normalizeMetricName(metric.Name)
	gaugeKey := fmt.Sprintf("%s_%s", metric.Source, name)
	
	gauge, exists := pb.gauges[gaugeKey]
	if !exists {
		labelNames := pb.getLabelsFromMetric(metric)
		gauge = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: pb.namespace,
				Subsystem: pb.subsystem,
				Name:      name,
				Help:      fmt.Sprintf("Gauge metric: %s", name),
			},
			labelNames,
		)
		pb.gauges[gaugeKey] = gauge
		pb.collectors[gaugeKey] = gauge
	}
	
	normalizedLabels := pb.normalizeLabels(metric.Labels)
	labelValues := make([]string, 0, len(normalizedLabels))
	for _, labelName := range pb.getLabelsFromMetric(metric) {
		labelValues = append(labelValues, normalizedLabels[labelName])
	}
	
	// Set the gauge to the metric value
	if len(labelValues) > 0 {
		gauge.WithLabelValues(labelValues...).Set(metric.Value)
	} else {
		gauge.WithLabelValues().Set(metric.Value)
	}
	
	return nil
}

// recordHistogram handles Histogram metrics
func (pb *PrometheusBridge) recordHistogram(metric *Metric) error {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	name := pb.normalizeMetricName(metric.Name)
	histogramKey := fmt.Sprintf("%s_%s", metric.Source, name)
	
	histogram, exists := pb.histograms[histogramKey]
	if !exists {
		labelNames := pb.getLabelsFromMetric(metric)
		
		// Get custom buckets from metadata or use defaults
		buckets := pb.defaultBuckets
		if metric.Metadata != nil {
			if customBuckets, ok := metric.Metadata["buckets"].([]float64); ok && len(customBuckets) > 0 {
				buckets = customBuckets
			}
		}
		
		histogram = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: pb.namespace,
				Subsystem: pb.subsystem,
				Name:      name,
				Help:      fmt.Sprintf("Histogram metric: %s", name),
				Buckets:   buckets,
			},
			labelNames,
		)
		pb.histograms[histogramKey] = histogram
		pb.collectors[histogramKey] = histogram
	}
	
	normalizedLabels := pb.normalizeLabels(metric.Labels)
	labelValues := make([]string, 0, len(normalizedLabels))
	for _, labelName := range pb.getLabelsFromMetric(metric) {
		labelValues = append(labelValues, normalizedLabels[labelName])
	}
	
	// Observe the metric value in the histogram
	if len(labelValues) > 0 {
		histogram.WithLabelValues(labelValues...).Observe(metric.Value)
	} else {
		histogram.WithLabelValues().Observe(metric.Value)
	}
	
	return nil
}

// recordSummary handles Summary metrics
func (pb *PrometheusBridge) recordSummary(metric *Metric) error {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	name := pb.normalizeMetricName(metric.Name)
	summaryKey := fmt.Sprintf("%s_%s", metric.Source, name)
	
	summary, exists := pb.summaries[summaryKey]
	if !exists {
		labelNames := pb.getLabelsFromMetric(metric)
		
		// Default objectives
		objectives := map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
		
		// Get custom objectives from metadata
		if metric.Metadata != nil {
			if customObjectives, ok := metric.Metadata["objectives"].(map[float64]float64); ok && len(customObjectives) > 0 {
				objectives = customObjectives
			}
		}
		
		summary = promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  pb.namespace,
				Subsystem:  pb.subsystem,
				Name:       name,
				Help:       fmt.Sprintf("Summary metric: %s", name),
				Objectives: objectives,
			},
			labelNames,
		)
		pb.summaries[summaryKey] = summary
		pb.collectors[summaryKey] = summary
	}
	
	normalizedLabels := pb.normalizeLabels(metric.Labels)
	labelValues := make([]string, 0, len(normalizedLabels))
	for _, labelName := range pb.getLabelsFromMetric(metric) {
		labelValues = append(labelValues, normalizedLabels[labelName])
	}
	
	// Observe the metric value in the summary
	if len(labelValues) > 0 {
		summary.WithLabelValues(labelValues...).Observe(metric.Value)
	} else {
		summary.WithLabelValues().Observe(metric.Value)
	}
	
	return nil
}

// GetCollectors returns all Prometheus collectors
func (pb *PrometheusBridge) GetCollectors() map[string]prometheus.Collector {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	// Return a copy to prevent modifications
	collectors := make(map[string]prometheus.Collector, len(pb.collectors))
	for k, v := range pb.collectors {
		collectors[k] = v
	}
	
	return collectors
}

// RegisterWithPrometheus registers all collectors with Prometheus
func (pb *PrometheusBridge) RegisterWithPrometheus() error {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	for name, collector := range pb.collectors {
		if err := prometheus.Register(collector); err != nil {
			return fmt.Errorf("failed to register %s: %w", name, err)
		}
	}
	
	return nil
}

// UnregisterFromPrometheus unregisters all collectors from Prometheus
func (pb *PrometheusBridge) UnregisterFromPrometheus() {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	for _, collector := range pb.collectors {
		prometheus.Unregister(collector)
	}
}

// Custom metric types for bridge-related metrics

// BridgeRequestCounter counts bridge requests
type BridgeRequestCounter struct {
	counter *prometheus.CounterVec
}

// NewBridgeRequestCounter creates a new bridge request counter
func NewBridgeRequestCounter() *BridgeRequestCounter {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_requests_total",
			Help: "Total number of bridge requests",
		},
		[]string{"protocol", "service", "status"},
	)
	
	prometheus.MustRegister(counter)
	
	return &BridgeRequestCounter{
		counter: counter,
	}
}

// Inc increments the counter for a specific protocol, service, and status
func (c *BridgeRequestCounter) Inc(protocol, service, status string) {
	c.counter.WithLabelValues(protocol, service, status).Inc()
}

// BridgeLatencyHistogram measures bridge request latency
type BridgeLatencyHistogram struct {
	histogram *prometheus.HistogramVec
}

// NewBridgeLatencyHistogram creates a new bridge latency histogram
func NewBridgeLatencyHistogram() *BridgeLatencyHistogram {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_request_duration_seconds",
			Help:    "Bridge request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"protocol", "service"},
	)
	
	prometheus.MustRegister(histogram)
	
	return &BridgeLatencyHistogram{
		histogram: histogram,
	}
}

// Observe records a latency observation for a specific protocol and service
func (h *BridgeLatencyHistogram) Observe(protocol, service string, duration float64) {
	h.histogram.WithLabelValues(protocol, service).Observe(duration)
}

// BridgeConnectionGauge tracks active bridge connections
type BridgeConnectionGauge struct {
	gauge *prometheus.GaugeVec
}

// NewBridgeConnectionGauge creates a new bridge connection gauge
func NewBridgeConnectionGauge() *BridgeConnectionGauge {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_connections_active",
			Help: "Number of active bridge connections",
		},
		[]string{"protocol", "service"},
	)
	
	prometheus.MustRegister(gauge)
	
	return &BridgeConnectionGauge{
		gauge: gauge,
	}
}

// Set sets the gauge value for a specific protocol and service
func (g *BridgeConnectionGauge) Set(protocol, service string, value float64) {
	g.gauge.WithLabelValues(protocol, service).Set(value)
}

// Inc increments the gauge for a specific protocol and service
func (g *BridgeConnectionGauge) Inc(protocol, service string) {
	g.gauge.WithLabelValues(protocol, service).Inc()
}

// Dec decrements the gauge for a specific protocol and service
func (g *BridgeConnectionGauge) Dec(protocol, service string) {
	g.gauge.WithLabelValues(protocol, service).Dec()
}
