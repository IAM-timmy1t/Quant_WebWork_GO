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
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/tracing"
)

// MonitorConfig defines configuration for system monitoring
type MonitorConfig struct {
	Enabled           bool          `json:"enabled"`
	CollectionInterval time.Duration `json:"collectionInterval"`
	RetentionPeriod   time.Duration `json:"retentionPeriod"`
	DiskPaths         []string      `json:"diskPaths"`
	NetworkInterfaces []string      `json:"networkInterfaces"`
	EnableTracing     bool          `json:"enableTracing"`
	AlertConfig       AlertConfig   `json:"alertConfig"`
	MonitorProcesses  bool          `json:"monitorProcesses"`
}

// AlertConfig defines configuration for monitoring alerts
type AlertConfig struct {
	CPUThreshold    float64 `json:"cpuThreshold"`    // Percentage
	MemoryThreshold float64 `json:"memoryThreshold"` // Percentage
	DiskThreshold   float64 `json:"diskThreshold"`   // Percentage
	AlertInterval   time.Duration `json:"alertInterval"`
}

// SystemMetrics represents system-wide metrics
type SystemMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpuUsage"`
	MemoryUsage   float64   `json:"memoryUsage"`
	MemoryTotal   uint64    `json:"memoryTotal"`
	MemoryFree    uint64    `json:"memoryFree"`
	SwapUsage     float64   `json:"swapUsage"`
	SwapTotal     uint64    `json:"swapTotal"`
	SwapFree      uint64    `json:"swapFree"`
	LoadAvg1      float64   `json:"loadAvg1"`
	LoadAvg5      float64   `json:"loadAvg5"`
	LoadAvg15     float64   `json:"loadAvg15"`
	NumGoroutines int       `json:"numGoroutines"`
	NumThreads    int       `json:"numThreads"`
	NumCPUs       int       `json:"numCPUs"`
}

// DiskMetrics represents disk usage metrics
type DiskMetrics struct {
	Timestamp   time.Time `json:"timestamp"`
	Path        string    `json:"path"`
	Total       uint64    `json:"total"`
	Used        uint64    `json:"used"`
	Free        uint64    `json:"free"`
	UsagePercent float64   `json:"usagePercent"`
	IOPS        uint64    `json:"iops"`
	ReadBytes   uint64    `json:"readBytes"`
	WriteBytes  uint64    `json:"writeBytes"`
}

// NetworkMetrics represents network interface metrics
type NetworkMetrics struct {
	Timestamp    time.Time `json:"timestamp"`
	Interface    string    `json:"interface"`
	BytesSent    uint64    `json:"bytesSent"`
	BytesRecv    uint64    `json:"bytesRecv"`
	PacketsSent  uint64    `json:"packetsSent"`
	PacketsRecv  uint64    `json:"packetsRecv"`
	ErrorsIn     uint64    `json:"errorsIn"`
	ErrorsOut    uint64    `json:"errorsOut"`
	DroppedIn    uint64    `json:"droppedIn"`
	DroppedOut   uint64    `json:"droppedOut"`
}

// ProcessMetrics represents process-level metrics
type ProcessMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	PID          int32     `json:"pid"`
	Name         string    `json:"name"`
	CPUUsage     float64   `json:"cpuUsage"`
	MemoryUsage  uint64    `json:"memoryUsage"`
	ThreadCount  int32     `json:"threadCount"`
	OpenFiles    int32     `json:"openFiles"`
}

// AlertEvent represents a monitoring alert
type AlertEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	Type      string        `json:"type"`
	Metric    string        `json:"metric"`
	Value     float64       `json:"value"`
	Threshold float64       `json:"threshold"`
	Message   string        `json:"message"`
}

// SystemMonitor manages system monitoring functionality
type SystemMonitor struct {
	mu sync.RWMutex

	config MonitorConfig
	tracer *tracing.Tracer

	// Metrics storage
	metrics        []SystemMetrics
	diskMetrics    map[string][]DiskMetrics
	netMetrics     map[string][]NetworkMetrics
	processMetrics map[int32][]ProcessMetrics

	// Aggregated metrics
	hourlyMetrics  []SystemMetrics
	dailyMetrics   []SystemMetrics
	monthlyMetrics []SystemMetrics

	// Alert channels
	alertSubscribers map[chan<- AlertEvent]bool

	// Subscribers for real-time updates
	subscribers map[chan<- SystemMetrics]bool

	// Control channels
	stopChan chan struct{}
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor(config MonitorConfig, tracer *tracing.Tracer) *SystemMonitor {
	return &SystemMonitor{
		config:           config,
		tracer:           tracer,
		metrics:          make([]SystemMetrics, 0),
		diskMetrics:      make(map[string][]DiskMetrics),
		netMetrics:       make(map[string][]NetworkMetrics),
		processMetrics:   make(map[int32][]ProcessMetrics),
		hourlyMetrics:    make([]SystemMetrics, 0),
		dailyMetrics:     make([]SystemMetrics, 0),
		monthlyMetrics:   make([]SystemMetrics, 0),
		alertSubscribers: make(map[chan<- AlertEvent]bool),
		subscribers:      make(map[chan<- SystemMetrics]bool),
		stopChan:         make(chan struct{}),
	}
}

// Start begins system monitoring
func (sm *SystemMonitor) Start(ctx context.Context) error {
	if !sm.config.Enabled {
		return nil
	}

	ticker := time.NewTicker(sm.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sm.collect(ctx); err != nil {
				// Log error but continue monitoring
				fmt.Printf("Error collecting metrics: %v\n", err)
			}
		case <-sm.stopChan:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Stop stops system monitoring
func (sm *SystemMonitor) Stop() {
	close(sm.stopChan)
}

// Subscribe adds a subscriber channel for real-time metrics
func (sm *SystemMonitor) Subscribe(ch chan<- SystemMetrics) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.subscribers[ch] = true
}

// Unsubscribe removes a subscriber channel
func (sm *SystemMonitor) Unsubscribe(ch chan<- SystemMetrics) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.subscribers, ch)
}

// SubscribeAlerts adds a subscriber channel for monitoring alerts
func (sm *SystemMonitor) SubscribeAlerts(ch chan<- AlertEvent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.alertSubscribers[ch] = true
}

// UnsubscribeAlerts removes an alert subscriber channel
func (sm *SystemMonitor) UnsubscribeAlerts(ch chan<- AlertEvent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.alertSubscribers, ch)
}

// GetMetrics returns collected system metrics
func (sm *SystemMonitor) GetMetrics(duration time.Duration) []SystemMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if duration == 0 {
		return sm.metrics
	}

	cutoff := time.Now().Add(-duration)
	var filtered []SystemMetrics
	for _, m := range sm.metrics {
		if m.Timestamp.After(cutoff) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// GetDiskMetrics returns collected disk metrics
func (sm *SystemMonitor) GetDiskMetrics(path string, duration time.Duration) []DiskMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.diskMetrics[path]
	if !exists {
		return nil
	}

	if duration == 0 {
		return metrics
	}

	cutoff := time.Now().Add(-duration)
	var filtered []DiskMetrics
	for _, m := range metrics {
		if m.Timestamp.After(cutoff) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// GetNetworkMetrics returns collected network metrics
func (sm *SystemMonitor) GetNetworkMetrics(iface string, duration time.Duration) []NetworkMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.netMetrics[iface]
	if !exists {
		return nil
	}

	if duration == 0 {
		return metrics
	}

	cutoff := time.Now().Add(-duration)
	var filtered []NetworkMetrics
	for _, m := range metrics {
		if m.Timestamp.After(cutoff) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// GetProcessMetrics returns metrics for a specific process
func (sm *SystemMonitor) GetProcessMetrics(pid int32, duration time.Duration) []ProcessMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if metrics, ok := sm.processMetrics[pid]; ok {
		cutoff := time.Now().Add(-duration)
		result := make([]ProcessMetrics, 0)
		for _, m := range metrics {
			if m.Timestamp.After(cutoff) {
				result = append(result, m)
			}
		}
		return result
	}
	return nil
}

// GetAggregatedMetrics returns aggregated metrics for the specified timeframe
func (sm *SystemMonitor) GetAggregatedMetrics(timeframe string) []SystemMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	switch timeframe {
	case "hourly":
		return sm.hourlyMetrics
	case "daily":
		return sm.dailyMetrics
	case "monthly":
		return sm.monthlyMetrics
	default:
		return nil
	}
}

// collect gathers system metrics
func (sm *SystemMonitor) collect(ctx context.Context) error {
	ctx, span := sm.tracer.StartSpan(ctx, "SystemMonitor.collect")
	defer span.End()

	// Collect CPU metrics
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		sm.tracer.RecordError(ctx, fmt.Errorf("error collecting CPU metrics: %v", err))
		return err
	}

	// Collect memory metrics
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		sm.tracer.RecordError(ctx, fmt.Errorf("error collecting memory metrics: %v", err))
		return err
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		sm.tracer.RecordError(ctx, fmt.Errorf("error collecting swap metrics: %v", err))
		return err
	}

	// Create metrics
	metrics := SystemMetrics{
		Timestamp:     time.Now(),
		CPUUsage:     cpuPercent[0],
		MemoryUsage:  memInfo.UsedPercent,
		MemoryTotal:  memInfo.Total,
		MemoryFree:   memInfo.Free,
		SwapUsage:    swapInfo.UsedPercent,
		SwapTotal:    swapInfo.Total,
		SwapFree:     swapInfo.Free,
		NumGoroutines: runtime.NumGoroutine(),
		NumThreads:    runtime.NumCPU(),
		NumCPUs:      runtime.NumCPU(),
	}

	// Collect disk metrics
	for _, path := range sm.config.DiskPaths {
		usage, err := disk.Usage(path)
		if err != nil {
			sm.tracer.RecordError(ctx, fmt.Errorf("error collecting disk metrics for %s: %v", path, err))
			continue
		}

		diskMetric := DiskMetrics{
			Timestamp:    time.Now(),
			Path:        path,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsagePercent: usage.UsedPercent,
		}

		sm.mu.Lock()
		sm.diskMetrics[path] = append(sm.diskMetrics[path], diskMetric)
		sm.mu.Unlock()
	}

	// Collect network metrics
	netStats, err := net.IOCounters(true)
	if err != nil {
		sm.tracer.RecordError(ctx, fmt.Errorf("error collecting network metrics: %v", err))
	} else {
		for _, stat := range netStats {
			if len(sm.config.NetworkInterfaces) > 0 {
				found := false
				for _, iface := range sm.config.NetworkInterfaces {
					if stat.Name == iface {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			netMetric := NetworkMetrics{
				Timestamp:   time.Now(),
				Interface:   stat.Name,
				BytesSent:   stat.BytesSent,
				BytesRecv:   stat.BytesRecv,
				PacketsSent: stat.PacketsSent,
				PacketsRecv: stat.PacketsRecv,
				ErrorsIn:    stat.Errin,
				ErrorsOut:   stat.Errout,
				DroppedIn:   stat.Dropin,
				DroppedOut:  stat.Dropout,
			}

			sm.mu.Lock()
			sm.netMetrics[stat.Name] = append(sm.netMetrics[stat.Name], netMetric)
			sm.mu.Unlock()
		}
	}

	// Collect process metrics
	if sm.config.MonitorProcesses {
		procs, err := process.Processes()
		if err != nil {
			sm.tracer.RecordError(ctx, fmt.Errorf("error collecting process metrics: %v", err))
		} else {
			for _, p := range procs {
				pInfo, err := p.Info()
				if err != nil {
					sm.tracer.RecordError(ctx, fmt.Errorf("error collecting process info for %d: %v", p.Pid, err))
					continue
				}

				processMetric := ProcessMetrics{
					Timestamp:     time.Now(),
					PID:          p.Pid,
					Name:         pInfo.Name,
					CPUUsage:     pInfo.CPUPercent,
					MemoryUsage:  pInfo.MemoryPercent,
					ThreadCount:  pInfo.NumThreads,
					OpenFiles:    pInfo.OpenFiles,
				}

				sm.mu.Lock()
				sm.processMetrics[p.Pid] = append(sm.processMetrics[p.Pid], processMetric)
				sm.mu.Unlock()
			}
		}
	}

	// Store metrics
	sm.mu.Lock()
	sm.metrics = append(sm.metrics, metrics)
	sm.mu.Unlock()

	// Notify subscribers
	sm.notifySubscribers(metrics)

	// Cleanup old metrics
	sm.cleanup()

	// Check for alerts
	sm.checkAlerts(metrics)

	return nil
}

// cleanup removes old metrics data
func (sm *SystemMonitor) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-sm.config.RetentionPeriod)

	// Clean system metrics
	var newMetrics []SystemMetrics
	for _, m := range sm.metrics {
		if m.Timestamp.After(cutoff) {
			newMetrics = append(newMetrics, m)
		}
	}
	sm.metrics = newMetrics

	// Clean disk metrics
	for path, metrics := range sm.diskMetrics {
		var newDiskMetrics []DiskMetrics
		for _, m := range metrics {
			if m.Timestamp.After(cutoff) {
				newDiskMetrics = append(newDiskMetrics, m)
			}
		}
		if len(newDiskMetrics) > 0 {
			sm.diskMetrics[path] = newDiskMetrics
		} else {
			delete(sm.diskMetrics, path)
		}
	}

	// Clean network metrics
	for iface, metrics := range sm.netMetrics {
		var newNetMetrics []NetworkMetrics
		for _, m := range metrics {
			if m.Timestamp.After(cutoff) {
				newNetMetrics = append(newNetMetrics, m)
			}
		}
		if len(newNetMetrics) > 0 {
			sm.netMetrics[iface] = newNetMetrics
		} else {
			delete(sm.netMetrics, iface)
		}
	}

	// Clean process metrics
	for pid, metrics := range sm.processMetrics {
		var newProcessMetrics []ProcessMetrics
		for _, m := range metrics {
			if m.Timestamp.After(cutoff) {
				newProcessMetrics = append(newProcessMetrics, m)
			}
		}
		if len(newProcessMetrics) > 0 {
			sm.processMetrics[pid] = newProcessMetrics
		} else {
			delete(sm.processMetrics, pid)
		}
	}
}

// notifySubscribers sends metrics to all subscribers
func (sm *SystemMonitor) notifySubscribers(metrics SystemMetrics) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for ch := range sm.subscribers {
		select {
		case ch <- metrics:
		default:
			// Skip if channel is blocked
		}
	}
}

// checkAlerts checks for monitoring alerts
func (sm *SystemMonitor) checkAlerts(metrics SystemMetrics) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if metrics.CPUUsage > sm.config.AlertConfig.CPUThreshold {
		alert := AlertEvent{
			Timestamp: time.Now(),
			Type:      "CPU",
			Metric:    "Usage",
			Value:     metrics.CPUUsage,
			Threshold: sm.config.AlertConfig.CPUThreshold,
			Message:   fmt.Sprintf("CPU usage exceeded threshold of %.2f%%", sm.config.AlertConfig.CPUThreshold),
		}
		sm.notifyAlertSubscribers(alert)
	}

	if metrics.MemoryUsage > sm.config.AlertConfig.MemoryThreshold {
		alert := AlertEvent{
			Timestamp: time.Now(),
			Type:      "Memory",
			Metric:    "Usage",
			Value:     metrics.MemoryUsage,
			Threshold: sm.config.AlertConfig.MemoryThreshold,
			Message:   fmt.Sprintf("Memory usage exceeded threshold of %.2f%%", sm.config.AlertConfig.MemoryThreshold),
		}
		sm.notifyAlertSubscribers(alert)
	}

	// Add more alert checks as needed
}

// notifyAlertSubscribers sends alerts to all subscribers
func (sm *SystemMonitor) notifyAlertSubscribers(alert AlertEvent) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for ch := range sm.alertSubscribers {
		select {
		case ch <- alert:
		default:
			// Skip if channel is blocked
		}
	}
}

