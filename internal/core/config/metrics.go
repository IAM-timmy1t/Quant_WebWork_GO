// metrics.go - Metrics configuration types

package config

import "time"

// MetricsConfig contains metrics collection configuration
type MetricsConfig struct {
	// Enabled indicates if metrics collection is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Interval defines how often to collect resource metrics
	Interval time.Duration `json:"interval" yaml:"interval"`

	// PrometheusEndpoint is the endpoint for exposing Prometheus metrics
	PrometheusEndpoint string `json:"prometheus_endpoint" yaml:"prometheus_endpoint"`

	// CollectionNamespace is the namespace for metrics collection
	CollectionNamespace string `json:"collection_namespace" yaml:"collection_namespace"`

	// CollectionSubsystem is the subsystem for metrics collection
	CollectionSubsystem string `json:"collection_subsystem" yaml:"collection_subsystem"`

	// HistogramBuckets defines custom histogram buckets
	HistogramBuckets []float64 `json:"histogram_buckets" yaml:"histogram_buckets"`

	// Labels contains additional labels to add to all metrics
	Labels map[string]string `json:"labels" yaml:"labels"`
}

// DefaultMetricsConfig returns the default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled:            true,
		Interval:           15 * time.Second,
		PrometheusEndpoint: "/metrics",
		CollectionNamespace: "quant",
		CollectionSubsystem: "webwork",
		HistogramBuckets:   nil, // Use Prometheus defaults
		Labels: map[string]string{
			"service": "webwork_server",
		},
	}
}
