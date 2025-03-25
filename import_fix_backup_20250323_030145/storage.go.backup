// storage.go - Persistent storage for metrics

package metrics

import (
    "fmt"
    "sort"
    "sync"
    "time"

    "github.com/google/uuid"
)

// Storage implements the StorageEngine interface
type Storage struct {
    config     StorageConfig
    metrics    []Metric
    mu         sync.RWMutex
    retention  time.Duration
    lastPurge  time.Time
}

// NewStorage creates a new storage engine
func NewStorage(config StorageConfig) (StorageEngine, error) {
    retention := config.RetentionTime
    if retention <= 0 {
        retention = 24 * time.Hour // Default retention: 24 hours
    }
    
    return &Storage{
        config:    config,
        metrics:   make([]Metric, 0),
        retention: retention,
        lastPurge: time.Now(),
    }, nil
}

// Store persists a batch of metrics
func (s *Storage) Store(metrics []Metric) error {
    if len(metrics) == 0 {
        return nil
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Add unique IDs to metrics if they don't have one
    for i := range metrics {
        if metrics[i].ID == "" {
            metrics[i].ID = uuid.New().String()
        }
    }
    
    // Add metrics to storage
    s.metrics = append(s.metrics, metrics...)
    
    // Periodically purge old metrics
    if time.Since(s.lastPurge) > time.Hour {
        s.purgeOldMetrics()
        s.lastPurge = time.Now()
    }
    
    return nil
}

// Query retrieves metrics based on criteria
func (s *Storage) Query(query QueryParams) ([]Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    var results []Metric
    
    for _, metric := range s.metrics {
        if s.matchesQuery(metric, query) {
            results = append(results, metric)
        }
        
        // Apply limit if specified
        if query.Limit > 0 && len(results) >= query.Limit {
            break
        }
    }
    
    // Apply ordering if specified
    if query.OrderBy != "" {
        s.sortMetrics(results, query.OrderBy, query.OrderDir)
    }
    
    return results, nil
}

// Aggregate performs aggregation on metrics according to specified parameters
func (s *Storage) Aggregate(query QueryParams, aggregation AggregationParams) ([]AggregatedMetric, error) {
    // Get matching metrics first
    metrics, err := s.Query(query)
    if err != nil {
        return nil, err
    }
    
    if len(metrics) == 0 {
        return []AggregatedMetric{}, nil
    }
    
    // Group metrics by specified fields
    groups := make(map[string][]Metric)
    groupKeys := make(map[string]map[string]string)
    
    for _, metric := range metrics {
        // Create group key
        key := ""
        groupKey := make(map[string]string)
        
        for _, field := range aggregation.GroupBy {
            var value string
            switch field {
            case "type":
                value = metric.Type
            case "source":
                value = metric.Source
            case "name":
                value = metric.Name
            default:
                // Check for tag
                if tagVal, ok := metric.Tags[field]; ok {
                    value = tagVal
                } else if labelVal, ok := metric.Labels[field]; ok {
                    value = labelVal
                } else {
                    value = "unknown"
                }
            }
            
            key += value + "|"
            groupKey[field] = value
        }
        
        // Add time interval for time-based aggregation
        if aggregation.Interval > 0 {
            timeKey := metric.Timestamp.Truncate(aggregation.Interval).Format(time.RFC3339)
            key += timeKey
            groupKey["time"] = timeKey
        }
        
        if _, ok := groups[key]; !ok {
            groups[key] = make([]Metric, 0)
            groupKeys[key] = groupKey
        }
        
        groups[key] = append(groups[key], metric)
    }
    
    // Perform aggregation for each group
    var results []AggregatedMetric
    
    for key, groupMetrics := range groups {
        var value float64
        var startTime, endTime time.Time
        
        // Find start and end times
        if len(groupMetrics) > 0 {
            startTime = groupMetrics[0].Timestamp
            endTime = groupMetrics[0].Timestamp
            
            for _, m := range groupMetrics {
                if m.Timestamp.Before(startTime) {
                    startTime = m.Timestamp
                }
                if m.Timestamp.After(endTime) {
                    endTime = m.Timestamp
                }
            }
        }
        
        // Apply aggregation function
        switch aggregation.Function {
        case "avg":
            sum := 0.0
            for _, m := range groupMetrics {
                sum += m.Value
            }
            if len(groupMetrics) > 0 {
                value = sum / float64(len(groupMetrics))
            }
        case "sum":
            sum := 0.0
            for _, m := range groupMetrics {
                sum += m.Value
            }
            value = sum
        case "min":
            if len(groupMetrics) > 0 {
                value = groupMetrics[0].Value
                for _, m := range groupMetrics {
                    if m.Value < value {
                        value = m.Value
                    }
                }
            }
        case "max":
            if len(groupMetrics) > 0 {
                value = groupMetrics[0].Value
                for _, m := range groupMetrics {
                    if m.Value > value {
                        value = m.Value
                    }
                }
            }
        case "count":
            value = float64(len(groupMetrics))
        default:
            return nil, fmt.Errorf("unsupported aggregation function: %s", aggregation.Function)
        }
        
        results = append(results, AggregatedMetric{
            GroupKey:  groupKeys[key],
            Value:     value,
            Count:     len(groupMetrics),
            StartTime: startTime,
            EndTime:   endTime,
        })
    }
    
    return results, nil
}

// ApplyRetention applies the retention policy and removes old metrics
func (s *Storage) ApplyRetention() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.purgeOldMetrics()
    return nil
}

// Close releases resources
func (s *Storage) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Clear metrics to free memory
    s.metrics = nil
    
    return nil
}

// matchesQuery checks if a metric matches query parameters
func (s *Storage) matchesQuery(metric Metric, query QueryParams) bool {
    // Check time range
    if !query.StartTime.IsZero() && metric.Timestamp.Before(query.StartTime) {
        return false
    }
    if !query.EndTime.IsZero() && metric.Timestamp.After(query.EndTime) {
        return false
    }
    
    // Check types
    if len(query.Types) > 0 {
        matched := false
        for _, t := range query.Types {
            if t == metric.Type {
                matched = true
                break
            }
        }
        if !matched {
            return false
        }
    }
    
    // Check sources
    if len(query.Sources) > 0 {
        matched := false
        for _, source := range query.Sources {
            if source == metric.Source {
                matched = true
                break
            }
        }
        if !matched {
            return false
        }
    }
    
    // Check names
    if len(query.Names) > 0 {
        matched := false
        for _, name := range query.Names {
            if name == metric.Name {
                matched = true
                break
            }
        }
        if !matched {
            return false
        }
    }
    
    // Check tags
    if len(query.Tags) > 0 {
        for k, v := range query.Tags {
            if metricValue, ok := metric.Tags[k]; !ok || metricValue != v {
                return false
            }
        }
    }
    
    return true
}

// sortMetrics sorts metrics based on the specified field and direction
func (s *Storage) sortMetrics(metrics []Metric, orderBy, orderDir string) {
    sort.Slice(metrics, func(i, j int) bool {
        var isLess bool
        
        switch orderBy {
        case "timestamp":
            isLess = metrics[i].Timestamp.Before(metrics[j].Timestamp)
        case "value":
            isLess = metrics[i].Value < metrics[j].Value
        case "name":
            isLess = metrics[i].Name < metrics[j].Name
        case "source":
            isLess = metrics[i].Source < metrics[j].Source
        case "type":
            isLess = metrics[i].Type < metrics[j].Type
        default:
            isLess = metrics[i].Timestamp.Before(metrics[j].Timestamp)
        }
        
        if orderDir == "desc" {
            return !isLess
        }
        return isLess
    })
}

// purgeOldMetrics removes metrics older than retention period
func (s *Storage) purgeOldMetrics() {
    cutoff := time.Now().Add(-s.retention)
    newMetrics := make([]Metric, 0, len(s.metrics))
    
    for _, metric := range s.metrics {
        if metric.Timestamp.After(cutoff) {
            newMetrics = append(newMetrics, metric)
        }
    }
    
    s.metrics = newMetrics
}

// GetStorageImplementation returns concrete storage implementation based on type
func GetStorageImplementation(config StorageConfig) (StorageEngine, error) {
    switch config.Type {
    case "memory":
        return NewStorage(config)
    case "time-series":
        // This would implement a time-series database connector
        return nil, fmt.Errorf("time-series storage not implemented yet")
    case "prometheus":
        // This would implement a Prometheus adapter
        return nil, fmt.Errorf("prometheus storage not implemented yet")
    default:
        return NewStorage(config) // Default to in-memory
    }
}
