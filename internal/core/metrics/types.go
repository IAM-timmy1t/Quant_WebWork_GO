// types.go - Common type definitions for metrics system

package metrics

import (
    "time"
)

// Metric type constants
const (
    MetricTypeCounter   = "counter"
    MetricTypeGauge     = "gauge"
    MetricTypeHistogram = "histogram"
    MetricTypeSummary   = "summary"
)

// Metric represents a single data point with metadata
type Metric struct {
    ID        string            `json:"id"`          // Unique identifier for the metric
    Source    string            `json:"source"`      // Source component/module that generated the metric
    Type      string            `json:"type"`        // Metric type/category
    Name      string            `json:"name"`        // Metric name (e.g., "system.cpu.usage")
    Value     float64           `json:"value"`       // Metric value
    Tags      map[string]string `json:"tags"`        // Additional metadata tags
    Labels    map[string]string `json:"labels,omitempty"` // Additional labels for filtering
    Timestamp time.Time         `json:"timestamp"`   // Time when the metric was collected
    Unit      string            `json:"unit,omitempty"` // Optional unit of measurement
    Metadata  map[string]interface{} `json:"metadata,omitempty"` // Additional metadata for the metric
}

// Config defines configuration for the metrics system
type Config struct {
    BatchSize         int               // Number of metrics to batch before storage
    FlushInterval     time.Duration     // Interval for periodic flush operations
    StorageConfig     StorageConfig     // Configuration for metrics storage
    Processors        []ProcessorConfig // Configuration for metric processors
    MaxResponseTimeHistory int          // Maximum number of response times to keep in history
    HistogramBuckets  []float64         // Buckets for Prometheus histograms
    EnablePrometheus  bool              // Whether to enable Prometheus integration
    RetentionPolicy   RetentionPolicy   // Policy for retention of metrics
}

// StorageConfig defines configuration for metrics storage
type StorageConfig struct {
    Type          string            // Storage type (e.g., "memory", "time-series", "prometheus")
    RetentionTime time.Duration     // How long to keep metrics
    Options       map[string]string // Additional storage-specific options
}

// ProcessorConfig defines configuration for a metric processor
type ProcessorConfig struct {
    Type    string            // Processor type (e.g., "aggregator", "filter", "transformer")
    Options map[string]string // Processor-specific options
}

// RetentionPolicy defines how long to keep different types of metrics
type RetentionPolicy struct {
    HighResolution time.Duration // Retention period for high-resolution metrics
    MediumResolution time.Duration // Retention period for medium-resolution metrics
    LowResolution time.Duration // Retention period for low-resolution metrics
}

// MetricsProcessor defines interface for components that process metrics
type MetricsProcessor interface {
    // Process transforms a metric and returns the processed version
    Process(metric Metric) Metric
}

// StorageEngine defines interface for metric storage backends
type StorageEngine interface {
    // Store persists a batch of metrics
    Store(metrics []Metric) error
    
    // Query retrieves metrics based on criteria
    Query(query QueryParams) ([]Metric, error)
    
    // Aggregate performs aggregation on metrics
    Aggregate(query QueryParams, aggregation AggregationParams) ([]AggregatedMetric, error)
    
    // ApplyRetention applies retention policy
    ApplyRetention() error
    
    // Close releases resources
    Close() error
}

// QueryParams defines criteria for querying metrics
type QueryParams struct {
    Types      []string          // Filter by metric types
    Sources    []string          // Filter by sources
    Names      []string          // Filter by metric names
    Tags       map[string]string // Filter by tags
    StartTime  time.Time         // Start of time range
    EndTime    time.Time         // End of time range
    Limit      int               // Maximum number of results
    OrderBy    string            // Field to order by
    OrderDir   string            // Order direction (asc, desc)
}

// AggregationParams defines parameters for metric aggregation
type AggregationParams struct {
    Function string        // Aggregation function (avg, sum, min, max, count)
    Interval time.Duration // Time interval for aggregation
    GroupBy  []string      // Fields to group by
}

// AggregatedMetric represents an aggregated metric result
type AggregatedMetric struct {
    GroupKey    map[string]string `json:"groupKey"`    // Grouping key
    Value       float64           `json:"value"`       // Aggregated value
    Count       int               `json:"count"`       // Number of data points
    StartTime   time.Time         `json:"startTime"`   // Start of aggregation period
    EndTime     time.Time         `json:"endTime"`     // End of aggregation period
}

// SystemMetrics tracks system-wide metrics
type SystemMetrics struct {
    CPU        CPUMetrics    `json:"cpu"`
    Memory     MemoryMetrics `json:"memory"`
    Disk       DiskMetrics   `json:"disk"`
    Network    NetworkMetrics `json:"network"`
    LoadAvg    []float64     `json:"loadAvg"`
    Goroutines int           `json:"goroutines"`
    Timestamp  time.Time     `json:"timestamp"`
}

// CPUMetrics contains CPU-related metrics
type CPUMetrics struct {
    Usage       float64 `json:"usage"`       // CPU usage percentage
    LoadAverage float64 `json:"loadAverage"` // System load average
    NumCPU      int     `json:"numCPU"`      // Number of CPUs
}

// MemoryMetrics contains memory-related metrics
type MemoryMetrics struct {
    Total        uint64  `json:"total"`        // Total memory
    Used         uint64  `json:"used"`         // Used memory
    Free         uint64  `json:"free"`         // Free memory
    UsagePercent float64 `json:"usagePercent"` // Memory usage percentage
}

// DiskMetrics contains disk-related metrics
type DiskMetrics struct {
    Total        uint64  `json:"total"`        // Total disk space
    Used         uint64  `json:"used"`         // Used disk space
    Free         uint64  `json:"free"`         // Free disk space
    UsagePercent float64 `json:"usagePercent"` // Disk usage percentage
}

// NetworkMetrics contains network-related metrics
type NetworkMetrics struct {
    BytesSent     uint64 `json:"bytesSent"`     // Bytes sent
    BytesRecv     uint64 `json:"bytesRecv"`     // Bytes received
    PacketsSent   uint64 `json:"packetsSent"`   // Packets sent
    PacketsRecv   uint64 `json:"packetsRecv"`   // Packets received
    ErrorsIn      uint64 `json:"errorsIn"`      // Input errors
    ErrorsOut     uint64 `json:"errorsOut"`     // Output errors
}

// ServiceMetrics tracks service-specific metrics
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

// RouteMetrics tracks HTTP route-specific metrics
type RouteMetrics struct {
    Method         string        `json:"method"`
    Path           string        `json:"path"`
    RequestCount   uint64        `json:"requestCount"`
    ErrorCount     uint64        `json:"errorCount"`
    ResponseTimes  []float64     `json:"responseTimes"`
    StatusCodes    map[int]uint64 `json:"statusCodes"`
    LastSeen       time.Time     `json:"lastSeen"`
}

// BridgeMetricsData provides metrics data specifically for bridge components
type BridgeMetricsData struct {
    BridgeID       string        `json:"bridgeId"`
    BridgeName     string        `json:"bridgeName"`
    Status         string        `json:"status"`
    LastSeen       time.Time     `json:"lastSeen"`
    CallCount      uint64        `json:"callCount"`
    ErrorCount     uint64        `json:"errorCount"`
    ResponseTimes  []float64     `json:"responseTimes"`
    LastError      time.Time     `json:"lastError"`
    LastErrorMsg   string        `json:"lastErrorMsg"`
    
    collector *Collector
    tags map[string]string
}

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
    Events         map[string]uint64 `json:"events"`       // Count of events by type
    Severity       map[string]uint64 `json:"severity"`     // Count of events by severity
    AvgRiskScore   float64           `json:"avgRiskScore"` // Average risk score
    MaxRiskScore   float64           `json:"maxRiskScore"` // Maximum risk score
    LastUpdateTime time.Time         `json:"lastUpdateTime"`
}

// Anomaly represents a detected anomaly in metrics
type Anomaly struct {
    ID             string    `json:"id"`
    MetricType     string    `json:"metricType"`
    Source         string    `json:"source"`
    Value          float64   `json:"value"`
    ExpectedValue  float64   `json:"expectedValue"`
    Score          float64   `json:"score"`          // Anomaly score (higher = more anomalous)
    Description    string    `json:"description"`
    Timestamp      time.Time `json:"timestamp"`
}

// Alert represents a generated alert from metrics
type Alert struct {
    ID          string    `json:"id"`
    Source      string    `json:"source"`
    Severity    string    `json:"severity"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    MetricType  string    `json:"metricType"`
    MetricValue float64   `json:"metricValue"`
    Timestamp   time.Time `json:"timestamp"`
    Resolved    bool      `json:"resolved"`
    ResolvedAt  time.Time `json:"resolvedAt,omitempty"`
}

// NewBridgeMetricsData creates a new bridge metrics data container
func NewBridgeMetricsData(bridgeName string) *BridgeMetricsData {
    return &BridgeMetricsData{
        BridgeName: bridgeName,
        tags: map[string]string{
            "component": "bridge",
            "bridge": bridgeName,
        },
    }
}

// SetCollector sets the metrics collector
func (bm *BridgeMetricsData) SetCollector(collector *Collector) {
    bm.collector = collector
}

// RecordRequest records a bridge request metric
func (bm *BridgeMetricsData) RecordRequest(operation string, durationMs float64, success bool) {
    if bm.collector == nil {
        return
    }
    
    tags := make(map[string]string)
    for k, v := range bm.tags {
        tags[k] = v
    }
    tags["operation"] = operation
    tags["success"] = boolToString(success)
    
    bm.collector.Collect("bridge", "request.duration", durationMs, tags)
}

// RecordError records a bridge error metric
func (bm *BridgeMetricsData) RecordError(errorType string) {
    if bm.collector == nil {
        return
    }
    
    tags := make(map[string]string)
    for k, v := range bm.tags {
        tags[k] = v
    }
    tags["error_type"] = errorType
    
    bm.collector.Collect("bridge", "error", 1.0, tags)
}

// Helper function to convert bool to string
func boolToString(b bool) string {
    if b {
        return "true"
    }
    return "false"
}
