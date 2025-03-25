// collector.go - Unified metrics collection

package metrics

import (
    "context"
    "sync"
    "time"
)

// Collector provides centralized metrics collection and distribution
type Collector struct {
    mu           sync.RWMutex
    config       *Config
    storage      StorageEngine
    processors   []MetricsProcessor
    subscribers  map[chan<- Metric]bool
    metricsBatch []Metric
}

// Collect gathers metrics from a specific source
func (c *Collector) Collect(source string, name string, value float64, tags map[string]string) {
    metric := Metric{
        Source:    source,
        Name:      name,
        Value:     value,
        Tags:      tags,
        Timestamp: time.Now(),
    }
    
    // Process metric through pipeline
    for _, processor := range c.processors {
        metric = processor.Process(metric)
    }
    
    // Add to batch for storage
    c.mu.Lock()
    c.metricsBatch = append(c.metricsBatch, metric)
    if len(c.metricsBatch) >= c.config.BatchSize {
        c.flushBatchAsync()
    }
    c.mu.Unlock()
    
    // Distribute to subscribers
    c.publishMetric(metric)
}

// NewCollector creates a new metrics collector with the specified configuration
func NewCollector(config *Config) *Collector {
    if config == nil {
        config = DefaultConfig()
    }
    
    return &Collector{
        config:      config,
        subscribers: make(map[chan<- Metric]bool),
        metricsBatch: make([]Metric, 0, config.BatchSize),
    }
}

// Subscribe registers a channel to receive metrics
func (c *Collector) Subscribe(ch chan<- Metric) func() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.subscribers[ch] = true
    
    return func() {
        c.mu.Lock()
        defer c.mu.Unlock()
        delete(c.subscribers, ch)
    }
}

// publishMetric sends a metric to all subscribers
func (c *Collector) publishMetric(metric Metric) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    for ch := range c.subscribers {
        select {
        case ch <- metric:
            // Successfully sent
        default:
            // Channel is blocked, skip this subscriber
        }
    }
}

// flushBatchAsync asynchronously flushes the metrics batch to storage
func (c *Collector) flushBatchAsync() {
    batch := c.metricsBatch
    c.metricsBatch = make([]Metric, 0, c.config.BatchSize)
    
    go func() {
        if err := c.storage.Store(batch); err != nil {
            // Handle storage error
        }
    }()
}

// Start begins the metrics collection process
func (c *Collector) Start(ctx context.Context) error {
    // Initialize storage if needed
    if c.storage == nil {
        var err error
        c.storage, err = NewStorage(c.config.StorageConfig)
        if err != nil {
            return err
        }
    }
    
    // Start periodic flush
    go c.periodicFlush(ctx)
    
    return nil
}

// periodicFlush periodically flushes metrics to storage
func (c *Collector) periodicFlush(ctx context.Context) {
    ticker := time.NewTicker(c.config.FlushInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.mu.Lock()
            if len(c.metricsBatch) > 0 {
                c.flushBatchAsync()
            }
            c.mu.Unlock()
        }
    }
}

// DefaultConfig returns the default metrics configuration
func DefaultConfig() *Config {
    return &Config{
        BatchSize:     100,
        FlushInterval: 10 * time.Second,
    }
}
