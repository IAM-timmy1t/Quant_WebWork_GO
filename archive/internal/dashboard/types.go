package dashboard

import (
	"time"

	"github.com/timot/Quant_WebWork_GO/internal/monitoring"
)

// MessageType represents different types of dashboard messages
type MessageType string

const (
	MessageTypeMetrics       MessageType = "metrics"
	MessageTypeSecurityEvent MessageType = "security_event"
	MessageTypeConfiguration MessageType = "configuration"
	MessageTypeError         MessageType = "error"
)

// Message represents a structured message for dashboard communication
type Message struct {
	Type    MessageType  `json:"type"`
	Payload interface{} `json:"payload"`
	Time    time.Time   `json:"time"`
}

// MetricsExportFormat represents supported export formats
type MetricsExportFormat string

const (
	FormatCSV  MetricsExportFormat = "csv"
	FormatJSON MetricsExportFormat = "json"
)

// MetricsAggregation represents time-based aggregation periods
type MetricsAggregation string

const (
	AggregationHourly  MetricsAggregation = "hourly"
	AggregationDaily   MetricsAggregation = "daily"
	AggregationMonthly MetricsAggregation = "monthly"
)

// ClientConfig represents client configuration options
type ClientConfig struct {
	Filters    MetricFilters `json:"filters"`
	Compressed bool          `json:"compressed"`
	BatchSize  int           `json:"batchSize"`
	BatchDelay time.Duration `json:"batchDelay"`
}

// MetricFilters defines client-specific metric filtering
type MetricFilters struct {
	Types       []string          `json:"types"`      // e.g., "cpu", "memory", "disk"
	Sources     []string          `json:"sources"`    // specific sources to monitor
	MinSeverity string            `json:"minSeverity"` // minimum security event severity
	Thresholds  map[string]float64 `json:"thresholds"` // metric-specific thresholds
}

// FilterMetrics checks if metrics pass the filter criteria
func (f *MetricFilters) FilterMetrics(metrics monitoring.ResourceMetrics) bool {
	if len(f.Types) > 0 {
		typeMatch := false
		for _, t := range f.Types {
			if metrics.Type == t {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	if len(f.Sources) > 0 {
		sourceMatch := false
		for _, s := range f.Sources {
			if metrics.Source == s {
				sourceMatch = true
				break
			}
		}
		if !sourceMatch {
			return false
		}
	}

	if threshold, ok := f.Thresholds[metrics.Type]; ok {
		if metrics.Value < threshold {
			return false
		}
	}

	return true
}

// FilterSecurityEvent checks if a security event passes the filter criteria
func (f *MetricFilters) FilterSecurityEvent(event monitoring.SecurityEvent) bool {
	if f.MinSeverity != "" {
		severityLevels := map[string]int{
			"low":    1,
			"medium": 2,
			"high":   3,
			"critical": 4,
		}

		eventLevel := severityLevels[event.Severity]
		minLevel := severityLevels[f.MinSeverity]

		if eventLevel < minLevel {
			return false
		}
	}

	return true
}
