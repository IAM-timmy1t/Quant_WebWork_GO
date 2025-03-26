// collector.go - Unified metrics collection

package metrics

import (
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"
    "go.uber.org/zap"
)

// Collector handles metrics collection
type Collector struct {
    config      config.MetricsConfig
    logger      *zap.SugaredLogger
    httpCounter *prometheus.CounterVec
    httpLatency *prometheus.HistogramVec
    
    // Resource metrics
    cpuUsage    prometheus.Gauge
    memoryUsage prometheus.Gauge
    diskUsage   prometheus.Gauge
    
    // Network metrics
    networkIn   prometheus.Counter
    networkOut  prometheus.Counter
}

// NewCollector creates a new metrics collector
func NewCollector(config config.MetricsConfig, logger *zap.SugaredLogger) *Collector {
    collector := &Collector{
        config: config,
        logger: logger,
        
        // HTTP metrics
        httpCounter: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "http_requests_total",
                Help: "Total number of HTTP requests",
            },
            []string{"method", "path", "status"},
        ),
        httpLatency: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "http_request_duration_seconds",
                Help:    "HTTP request latency in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "path"},
        ),
        
        // Resource metrics
        cpuUsage: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "system_cpu_usage_percent",
                Help: "Current CPU usage in percent",
            },
        ),
        memoryUsage: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "system_memory_usage_percent",
                Help: "Current memory usage in percent",
            },
        ),
        diskUsage: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "system_disk_usage_percent",
                Help: "Current disk usage in percent",
            },
        ),
        
        // Network metrics
        networkIn: prometheus.NewCounter(
            prometheus.CounterOpts{
                Name: "network_bytes_received",
                Help: "Total number of bytes received",
            },
        ),
        networkOut: prometheus.NewCounter(
            prometheus.CounterOpts{
                Name: "network_bytes_sent",
                Help: "Total number of bytes sent",
            },
        ),
    }
    
    // Register the metrics
    prometheus.MustRegister(
        collector.httpCounter,
        collector.httpLatency,
        collector.cpuUsage,
        collector.memoryUsage,
        collector.diskUsage,
        collector.networkIn,
        collector.networkOut,
    )
    
    // Start the resource metrics collection
    if config.Enabled {
        go collector.collectResourceMetrics(config.Interval)
    }
    
    return collector
}

// RecordHTTPRequest records metrics for an HTTP request
func (c *Collector) RecordHTTPRequest(method, path string, status int, duration float64) {
    statusStr := string(status)
    c.httpCounter.WithLabelValues(method, path, statusStr).Inc()
    c.httpLatency.WithLabelValues(method, path).Observe(duration)
}

// RecordNetworkActivity records network activity
func (c *Collector) RecordNetworkActivity(bytesIn, bytesOut float64) {
    c.networkIn.Add(bytesIn)
    c.networkOut.Add(bytesOut)
}

// Collect provides backwards compatibility with existing code that uses the Collect method
func (c *Collector) Collect(source string, name string, value float64, tags map[string]string) {
    // Convert tags to a slice of string values for Prometheus labels
    if name == "" || source == "" {
        c.logger.Warnw("Invalid metric data", "source", source, "name", name)
        return
    }
    
    // Handle different metric types based on source
    switch source {
    case "http":
        if method, ok := tags["method"]; ok {
            if path, ok := tags["path"]; ok {
                if status, ok := tags["status"]; ok {
                    c.httpCounter.WithLabelValues(method, path, status).Inc()
                }
            }
        }
    case "latency":
        if method, ok := tags["method"]; ok {
            if path, ok := tags["path"]; ok {
                c.httpLatency.WithLabelValues(method, path).Observe(value)
            }
        }
    case "network":
        if direction, ok := tags["direction"]; ok {
            if direction == "in" {
                c.networkIn.Add(value)
            } else if direction == "out" {
                c.networkOut.Add(value)
            }
        }
    default:
        c.logger.Warnw("Unknown metric source", "source", source)
    }
}

// collectResourceMetrics periodically collects resource metrics
func (c *Collector) collectResourceMetrics(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // Collect CPU usage
            cpuUsage, err := getCPUUsage()
            if err != nil {
                c.logger.Warnw("Failed to collect CPU usage", "error", err)
            } else {
                c.cpuUsage.Set(cpuUsage)
            }
            
            // Collect memory usage
            memoryUsage, err := getMemoryUsage()
            if err != nil {
                c.logger.Warnw("Failed to collect memory usage", "error", err)
            } else {
                c.memoryUsage.Set(memoryUsage)
            }
            
            // Collect disk usage
            diskUsage, err := getDiskUsage()
            if err != nil {
                c.logger.Warnw("Failed to collect disk usage", "error", err)
            } else {
                c.diskUsage.Set(diskUsage)
            }
        }
    }
}

// getCPUUsage returns the current CPU usage in percent
func getCPUUsage() (float64, error) {
    // Implementation depends on the platform
    // This is a placeholder for now
    return 0, nil
}

// getMemoryUsage returns the current memory usage in percent
func getMemoryUsage() (float64, error) {
    // Implementation depends on the platform
    // This is a placeholder for now
    return 0, nil
}

// getDiskUsage returns the current disk usage in percent
func getDiskUsage() (float64, error) {
    // Implementation depends on the platform
    // This is a placeholder for now
    return 0, nil
}
