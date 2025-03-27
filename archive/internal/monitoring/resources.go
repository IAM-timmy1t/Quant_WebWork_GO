package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/storage"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/tracing"
)

// ResourceConfig defines configuration for resource monitoring
type ResourceConfig struct {
	Enabled           bool          `json:"enabled"`
	CollectionInterval time.Duration `json:"collectionInterval"`
	RetentionPeriod   time.Duration `json:"retentionPeriod"`
	
	// CPU monitoring
	CPUAlertThreshold    float64 `json:"cpuAlertThreshold"`
	CPUSampleInterval    time.Duration `json:"cpuSampleInterval"`
	
	// Memory monitoring
	MemoryAlertThreshold float64 `json:"memoryAlertThreshold"`
	SwapAlertThreshold   float64 `json:"swapAlertThreshold"`
	
	// Disk monitoring
	DiskAlertThreshold   float64 `json:"diskAlertThreshold"`
	MonitoredPaths      []string `json:"monitoredPaths"`
	
	// Network monitoring
	NetworkAlertThreshold float64 `json:"networkAlertThreshold"`
	MonitoredInterfaces  []string `json:"monitoredInterfaces"`
	
	// Process monitoring
	ProcessAlertThreshold float64 `json:"processAlertThreshold"`
	MaxProcessCount      int     `json:"maxProcessCount"`
}

// ResourceMetrics represents comprehensive resource metrics
type ResourceMetrics struct {
	Timestamp time.Time `json:"timestamp"`
	
	// CPU metrics
	CPUUsage       float64   `json:"cpuUsage"`
	CPUTemperature float64   `json:"cpuTemperature"`
	LoadAverage    [3]float64 `json:"loadAverage"`
	
	// Memory metrics
	MemoryUsage    float64 `json:"memoryUsage"`
	MemoryTotal    uint64  `json:"memoryTotal"`
	MemoryFree     uint64  `json:"memoryFree"`
	SwapUsage      float64 `json:"swapUsage"`
	SwapTotal      uint64  `json:"swapTotal"`
	SwapFree       uint64  `json:"swapFree"`
	
	// Process metrics
	ProcessCount   int     `json:"processCount"`
	ThreadCount    int     `json:"threadCount"`
	HandleCount    int     `json:"handleCount"`
	
	// System metrics
	UptimeSeconds  uint64  `json:"uptimeSeconds"`
	BootTime       uint64  `json:"bootTime"`
}

// ResourceAlert represents a resource-related alert
type ResourceAlert struct {
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Message    string    `json:"message"`
	Severity   string    `json:"severity"`
}

// ResourceMonitor manages resource monitoring
type ResourceMonitor struct {
	mu sync.RWMutex

	config ResourceConfig
	tracer *tracing.Tracer
	storage *storage.MetricsStorage

	// Metrics storage
	metrics []ResourceMetrics
	alerts  []ResourceAlert

	// Aggregated metrics
	hourlyMetrics  []ResourceMetrics
	dailyMetrics   []ResourceMetrics
	monthlyMetrics []ResourceMetrics

	// Subscribers
	subscribers map[chan<- ResourceMetrics]bool
	alertSubscribers map[chan<- ResourceAlert]bool

	// Control channels
	stopChan chan struct{}
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(config ResourceConfig, storage *storage.MetricsStorage, tracer *tracing.Tracer) *ResourceMonitor {
	return &ResourceMonitor{
		config:      config,
		storage:     storage,
		tracer:      tracer,
		metrics:     make([]ResourceMetrics, 0),
		alerts:      make([]ResourceAlert, 0),
		hourlyMetrics: make([]ResourceMetrics, 0),
		dailyMetrics: make([]ResourceMetrics, 0),
		monthlyMetrics: make([]ResourceMetrics, 0),
		subscribers: make(map[chan<- ResourceMetrics]bool),
		alertSubscribers: make(map[chan<- ResourceAlert]bool),
		stopChan: make(chan struct{}),
	}
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start(ctx context.Context) error {
	if !rm.config.Enabled {
		return nil
	}

	// Start collection routine
	go rm.collectRoutine(ctx)

	// Start storage routine
	go rm.storageRoutine(ctx)

	return nil
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() {
	close(rm.stopChan)
}

// Subscribe adds a subscriber for real-time metrics updates
func (rm *ResourceMonitor) Subscribe(ch chan<- ResourceMetrics) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.subscribers[ch] = true
}

// Unsubscribe removes a subscriber
func (rm *ResourceMonitor) Unsubscribe(ch chan<- ResourceMetrics) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.subscribers, ch)
}

// SubscribeAlert adds a subscriber for real-time alert updates
func (rm *ResourceMonitor) SubscribeAlert(ch chan<- ResourceAlert) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.alertSubscribers[ch] = true
}

// UnsubscribeAlert removes an alert subscriber
func (rm *ResourceMonitor) UnsubscribeAlert(ch chan<- ResourceAlert) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.alertSubscribers, ch)
}

// GetLatestMetrics returns the most recently collected metrics
func (rm *ResourceMonitor) GetLatestMetrics() (*ResourceMetrics, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if len(rm.metrics) == 0 {
		return nil, fmt.Errorf("no metrics available")
	}

	return &rm.metrics[len(rm.metrics)-1], nil
}

// collectRoutine periodically collects system metrics
func (rm *ResourceMonitor) collectRoutine(ctx context.Context) {
	ticker := time.NewTicker(rm.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := rm.collectResourceMetrics(ctx); err != nil {
				rm.tracer.RecordError(ctx, fmt.Errorf("error collecting metrics: %v", err))
			}

		case <-rm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// storageRoutine handles persisting metrics to storage
func (rm *ResourceMonitor) storageRoutine(ctx context.Context) {
	batch := make([]ResourceMetrics, 0, rm.config.RetentionPeriod/time.Second)
	ticker := time.NewTicker(time.Second * 10) // Flush every 10 seconds if batch not full
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if len(batch) > 0 {
				if err := rm.flushMetricsBatch(ctx, batch); err != nil {
					rm.tracer.RecordError(ctx, fmt.Errorf("error flushing metrics batch: %v", err))
				}
				batch = batch[:0]
			}

		case <-rm.stopChan:
			// Flush remaining metrics before stopping
			if len(batch) > 0 {
				if err := rm.flushMetricsBatch(ctx, batch); err != nil {
					rm.tracer.RecordError(ctx, fmt.Errorf("error flushing final metrics batch: %v", err))
				}
			}
			return

		case <-ctx.Done():
			return
		}
	}
}

// flushMetricsBatch persists a batch of metrics to storage
func (rm *ResourceMonitor) flushMetricsBatch(ctx context.Context, batch []ResourceMetrics) error {
	ctx, span := rm.tracer.StartSpan(ctx, "ResourceMonitor.flushMetricsBatch")
	defer span.End()

	for _, metrics := range batch {
		if err := rm.storage.StoreMetrics(ctx, metrics); err != nil {
			return fmt.Errorf("failed to store metrics: %v", err)
		}
	}

	return nil
}

// collectResourceMetrics gathers comprehensive resource metrics
func (rm *ResourceMonitor) collectResourceMetrics(ctx context.Context) error {
	ctx, span := rm.tracer.StartSpan(ctx, "ResourceMonitor.collectResourceMetrics")
	defer span.End()

	metrics := ResourceMetrics{
		Timestamp: time.Now(),
	}

	// Collect CPU metrics
	if cpuPercent, err := cpu.Percent(rm.config.CPUSampleInterval, false); err == nil {
		metrics.CPUUsage = cpuPercent[0]
		
		if metrics.CPUUsage > rm.config.CPUAlertThreshold {
			rm.recordAlert(ResourceAlert{
				Timestamp: time.Now(),
				Type:     "cpu",
				Metric:   "usage",
				Value:    metrics.CPUUsage,
				Threshold: rm.config.CPUAlertThreshold,
				Message:  fmt.Sprintf("CPU usage exceeded threshold: %.2f%%", metrics.CPUUsage),
				Severity: "high",
			})
		}
	} else {
		rm.tracer.RecordError(ctx, fmt.Errorf("error collecting CPU metrics: %v", err))
	}

	// Collect memory metrics
	if vmem, err := mem.VirtualMemory(); err == nil {
		metrics.MemoryUsage = vmem.UsedPercent
		metrics.MemoryTotal = vmem.Total
		metrics.MemoryFree = vmem.Free

		if metrics.MemoryUsage > rm.config.MemoryAlertThreshold {
			rm.recordAlert(ResourceAlert{
				Timestamp: time.Now(),
				Type:     "memory",
				Metric:   "usage",
				Value:    metrics.MemoryUsage,
				Threshold: rm.config.MemoryAlertThreshold,
				Message:  fmt.Sprintf("Memory usage exceeded threshold: %.2f%%", metrics.MemoryUsage),
				Severity: "high",
			})
		}
	} else {
		rm.tracer.RecordError(ctx, fmt.Errorf("error collecting memory metrics: %v", err))
	}

	// Collect swap metrics
	if swap, err := mem.SwapMemory(); err == nil {
		metrics.SwapUsage = swap.UsedPercent
		metrics.SwapTotal = swap.Total
		metrics.SwapFree = swap.Free

		if metrics.SwapUsage > rm.config.SwapAlertThreshold {
			rm.recordAlert(ResourceAlert{
				Timestamp: time.Now(),
				Type:     "swap",
				Metric:   "usage",
				Value:    metrics.SwapUsage,
				Threshold: rm.config.SwapAlertThreshold,
				Message:  fmt.Sprintf("Swap usage exceeded threshold: %.2f%%", metrics.SwapUsage),
				Severity: "medium",
			})
		}
	} else {
		rm.tracer.RecordError(ctx, fmt.Errorf("error collecting swap metrics: %v", err))
	}

	// Collect process metrics
	if procs, err := process.Processes(); err == nil {
		metrics.ProcessCount = len(procs)
		
		if metrics.ProcessCount > rm.config.MaxProcessCount {
			rm.recordAlert(ResourceAlert{
				Timestamp: time.Now(),
				Type:     "process",
				Metric:   "count",
				Value:    float64(metrics.ProcessCount),
				Threshold: float64(rm.config.MaxProcessCount),
				Message:  fmt.Sprintf("Process count exceeded threshold: %d", metrics.ProcessCount),
				Severity: "medium",
			})
		}

		// Collect thread and handle counts
		for _, p := range procs {
			if info, err := p.Info(); err == nil {
				metrics.ThreadCount += int(info.NumThreads)
				metrics.HandleCount += int(info.OpenFiles)
			}
		}
	} else {
		rm.tracer.RecordError(ctx, fmt.Errorf("error collecting process metrics: %v", err))
	}

	// Store metrics
	rm.mu.Lock()
	rm.metrics = append(rm.metrics, metrics)
	rm.mu.Unlock()

	// Update aggregated metrics
	rm.updateAggregatedMetrics(metrics)

	// Notify subscribers
	rm.notifySubscribers(metrics)

	return nil
}

// recordAlert records and broadcasts a resource alert
func (rm *ResourceMonitor) recordAlert(alert ResourceAlert) {
	rm.mu.Lock()
	rm.alerts = append(rm.alerts, alert)
	rm.mu.Unlock()

	rm.notifyAlertSubscribers(alert)
}

// notifyAlertSubscribers sends an alert to all subscribers
func (rm *ResourceMonitor) notifyAlertSubscribers(alert ResourceAlert) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	for ch := range rm.alertSubscribers {
		select {
		case ch <- alert:
		default:
			// Skip if channel is blocked
		}
	}
}

// updateAggregatedMetrics updates hourly, daily, and monthly metrics
func (rm *ResourceMonitor) updateAggregatedMetrics(metrics ResourceMetrics) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Update hourly metrics
	rm.hourlyMetrics = append(rm.hourlyMetrics, metrics)
	if len(rm.hourlyMetrics) > 3600/int(rm.config.CollectionInterval.Seconds()) {
		rm.hourlyMetrics = rm.hourlyMetrics[1:]
	}

	// Update daily metrics
	if len(rm.hourlyMetrics) > 0 && time.Since(rm.hourlyMetrics[0].Timestamp) >= time.Hour {
		dailyMetric := rm.aggregateMetrics(rm.hourlyMetrics)
		rm.dailyMetrics = append(rm.dailyMetrics, dailyMetric)
		if len(rm.dailyMetrics) > 24 {
			rm.dailyMetrics = rm.dailyMetrics[1:]
		}
	}

	// Update monthly metrics
	if len(rm.dailyMetrics) > 0 && time.Since(rm.dailyMetrics[0].Timestamp) >= 24*time.Hour {
		monthlyMetric := rm.aggregateMetrics(rm.dailyMetrics)
		rm.monthlyMetrics = append(rm.monthlyMetrics, monthlyMetric)
		if len(rm.monthlyMetrics) > 30 {
			rm.monthlyMetrics = rm.monthlyMetrics[1:]
		}
	}
}

// aggregateMetrics aggregates a slice of metrics into a single metric
func (rm *ResourceMonitor) aggregateMetrics(metrics []ResourceMetrics) ResourceMetrics {
	if len(metrics) == 0 {
		return ResourceMetrics{}
	}

	var result ResourceMetrics
	result.Timestamp = metrics[len(metrics)-1].Timestamp

	// Calculate averages
	for _, m := range metrics {
		result.CPUUsage += m.CPUUsage
		result.MemoryUsage += m.MemoryUsage
		result.SwapUsage += m.SwapUsage
		result.ProcessCount += m.ProcessCount
		result.ThreadCount += m.ThreadCount
		result.HandleCount += m.HandleCount
	}

	count := float64(len(metrics))
	result.CPUUsage /= count
	result.MemoryUsage /= count
	result.SwapUsage /= count
	result.ProcessCount = int(float64(result.ProcessCount) / count)
	result.ThreadCount = int(float64(result.ThreadCount) / count)
	result.HandleCount = int(float64(result.HandleCount) / count)

	// Use the latest values for totals
	latest := metrics[len(metrics)-1]
	result.MemoryTotal = latest.MemoryTotal
	result.MemoryFree = latest.MemoryFree
	result.SwapTotal = latest.SwapTotal
	result.SwapFree = latest.SwapFree
	result.UptimeSeconds = latest.UptimeSeconds
	result.BootTime = latest.BootTime

	return result
}

// notifySubscribers sends metrics to all subscribers
func (rm *ResourceMonitor) notifySubscribers(metrics ResourceMetrics) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	for ch := range rm.subscribers {
		select {
		case ch <- metrics:
		default:
			// Skip if channel is blocked
		}
	}
}

