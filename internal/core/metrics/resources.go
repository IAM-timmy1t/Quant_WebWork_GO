// resources.go - Resource usage monitoring implementation

package metrics

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// ResourceMetrics handles resource usage monitoring
type ResourceMetrics struct {
	// Core metrics
	cpuUsage        prometheus.Gauge
	memUsage        prometheus.Gauge
	memTotal        prometheus.Gauge
	diskUsage       *prometheus.GaugeVec
	diskIO          *prometheus.GaugeVec
	netIO           *prometheus.GaugeVec
	netConnections  prometheus.Gauge
	
	// Go runtime metrics
	goroutines      prometheus.Gauge
	gcPauses        prometheus.Histogram
	heapObjects     prometheus.Gauge
	heapAlloc       prometheus.Gauge
	
	// Process metrics
	processThreads  prometheus.Gauge
	processCPU      prometheus.Gauge
	processMemory   prometheus.Gauge
	processOpenFDs  prometheus.Gauge
	
	// Additional metrics
	fileDescriptors prometheus.Gauge
	uptimeSeconds   prometheus.Counter
	
	logger         *zap.SugaredLogger
	mutex          sync.RWMutex
	startTime      time.Time
	initialized    bool
	processID      int32
	basePath       string
}

// NewResourceMetrics creates a new resource metrics collector
func NewResourceMetrics(logger *zap.SugaredLogger, basePath string) *ResourceMetrics {
	rm := &ResourceMetrics{
		logger:     logger,
		startTime:  time.Now(),
		basePath:   basePath,
	}
	
	// Initialize the metrics
	rm.initializeMetrics()
	
	// Get current process ID
	proc, err := process.NewProcess(int32(process.GetCurrentProcessPid()))
	if err == nil {
		rm.processID = proc.Pid
	} else {
		rm.logger.Warnw("Failed to get current process", "error", err)
	}
	
	return rm
}

// initializeMetrics sets up all resource metrics
func (rm *ResourceMetrics) initializeMetrics() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	if rm.initialized {
		return
	}
	
	// Core system metrics
	rm.cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_cpu_usage_percent",
		Help: "Current CPU usage in percent across all cores",
	})
	
	rm.memUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_usage_percent",
		Help: "Current memory usage in percent",
	})
	
	rm.memTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_total_bytes",
		Help: "Total system memory in bytes",
	})
	
	rm.diskUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "system_disk_usage_percent",
		Help: "Current disk usage in percent",
	}, []string{"path", "fstype"})
	
	rm.diskIO = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "system_disk_io_operations",
		Help: "Disk IO operations per second",
	}, []string{"device", "type"}) // type: read or write
	
	rm.netIO = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "system_network_io_bytes",
		Help: "Network IO bytes per second",
	}, []string{"interface", "direction"}) // direction: sent or received
	
	rm.netConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_network_connections",
		Help: "Current number of network connections",
	})
	
	// Go runtime metrics
	rm.goroutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "go_goroutines",
		Help: "Current number of goroutines",
	})
	
	rm.gcPauses = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "go_gc_pause_seconds",
		Help:    "Garbage collection pause duration distribution",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})
	
	rm.heapObjects = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "go_heap_objects",
		Help: "Number of allocated heap objects",
	})
	
	rm.heapAlloc = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "go_heap_alloc_bytes",
		Help: "Heap memory allocated in bytes",
	})
	
	// Process-specific metrics
	rm.processThreads = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_threads",
		Help: "Current number of OS threads used by the process",
	})
	
	rm.processCPU = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_cpu_percent",
		Help: "CPU usage percentage of the current process",
	})
	
	rm.processMemory = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_memory_bytes",
		Help: "Memory usage of the current process in bytes",
	})
	
	rm.processOpenFDs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_open_fds",
		Help: "Number of open file descriptors by the process",
	})
	
	// Additional metrics
	rm.fileDescriptors = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_file_descriptors",
		Help: "Number of open file descriptors system-wide",
	})
	
	rm.uptimeSeconds = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "system_uptime_seconds",
		Help: "System uptime in seconds",
	})
	
	// Register metrics with Prometheus
	prometheus.MustRegister(
		rm.cpuUsage,
		rm.memUsage,
		rm.memTotal,
		rm.diskUsage,
		rm.diskIO,
		rm.netIO,
		rm.netConnections,
		rm.goroutines,
		rm.gcPauses,
		rm.heapObjects,
		rm.heapAlloc,
		rm.processThreads,
		rm.processCPU,
		rm.processMemory,
		rm.processOpenFDs,
		rm.fileDescriptors,
		rm.uptimeSeconds,
	)
	
	rm.initialized = true
}

// Start begins collecting resource metrics at the specified interval
func (rm *ResourceMetrics) Start(interval time.Duration) {
	go rm.collectMetrics(interval)
}

// collectMetrics periodically collects all resource metrics
func (rm *ResourceMetrics) collectMetrics(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Track last network IO stats for calculating rates
	var lastNetStats []net.IOCountersStat
	var lastNetTime time.Time
	
	// Track last disk IO stats for calculating rates
	var lastDiskStats map[string]disk.IOCountersStat
	var lastDiskTime time.Time
	
	for range ticker.C {
		// Update uptime
		rm.uptimeSeconds.Add(interval.Seconds())
		
		// Collect CPU usage
		if err := rm.collectCPUMetrics(); err != nil {
			rm.logger.Warnw("Failed to collect CPU metrics", "error", err)
		}
		
		// Collect memory metrics
		if err := rm.collectMemoryMetrics(); err != nil {
			rm.logger.Warnw("Failed to collect memory metrics", "error", err)
		}
		
		// Collect disk usage metrics
		if err := rm.collectDiskUsageMetrics(); err != nil {
			rm.logger.Warnw("Failed to collect disk usage metrics", "error", err)
		}
		
		// Collect disk IO metrics
		if err := rm.collectDiskIOMetrics(&lastDiskStats, &lastDiskTime); err != nil {
			rm.logger.Warnw("Failed to collect disk IO metrics", "error", err)
		}
		
		// Collect network metrics
		if err := rm.collectNetworkMetrics(&lastNetStats, &lastNetTime); err != nil {
			rm.logger.Warnw("Failed to collect network metrics", "error", err)
		}
		
		// Collect Go runtime metrics
		rm.collectRuntimeMetrics()
		
		// Collect process metrics
		if err := rm.collectProcessMetrics(); err != nil {
			rm.logger.Warnw("Failed to collect process metrics", "error", err)
		}
	}
}

// collectCPUMetrics collects CPU usage metrics
func (rm *ResourceMetrics) collectCPUMetrics() error {
	cpuPercent, err := cpu.Percent(0, false) // false = total CPU usage across all cores
	if err != nil {
		return err
	}
	
	if len(cpuPercent) > 0 {
		rm.cpuUsage.Set(cpuPercent[0])
	}
	
	return nil
}

// collectMemoryMetrics collects memory usage metrics
func (rm *ResourceMetrics) collectMemoryMetrics() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	
	rm.memUsage.Set(memInfo.UsedPercent)
	rm.memTotal.Set(float64(memInfo.Total))
	
	return nil
}

// collectDiskUsageMetrics collects disk usage metrics
func (rm *ResourceMetrics) collectDiskUsageMetrics() error {
	// Get all partitions
	partitions, err := disk.Partitions(false)
	if err != nil {
		return err
	}
	
	// Monitor the partition containing the base path and other important ones
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			rm.logger.Warnw("Failed to get disk usage", 
				"mountpoint", partition.Mountpoint, 
				"error", err)
			continue
		}
		
		rm.diskUsage.WithLabelValues(
			partition.Mountpoint,
			partition.Fstype,
		).Set(usage.UsedPercent)
	}
	
	return nil
}

// collectDiskIOMetrics collects disk IO metrics
func (rm *ResourceMetrics) collectDiskIOMetrics(lastStats *map[string]disk.IOCountersStat, lastTime *time.Time) error {
	// Get current disk IO counters
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return err
	}
	
	now := time.Now()
	
	// If we have previous measurements, calculate rates
	if *lastStats != nil && !lastTime.IsZero() {
		duration := now.Sub(*lastTime).Seconds()
		
		if duration > 0 {
			for device, stats := range ioCounters {
				if lastStat, ok := (*lastStats)[device]; ok {
					// Calculate reads per second
					readDiff := stats.ReadCount - lastStat.ReadCount
					rm.diskIO.WithLabelValues(device, "read").Set(float64(readDiff) / duration)
					
					// Calculate writes per second
					writeDiff := stats.WriteCount - lastStat.WriteCount
					rm.diskIO.WithLabelValues(device, "write").Set(float64(writeDiff) / duration)
				}
			}
		}
	}
	
	// Store current stats for next collection
	*lastStats = ioCounters
	*lastTime = now
	
	return nil
}

// collectNetworkMetrics collects network metrics
func (rm *ResourceMetrics) collectNetworkMetrics(lastStats *[]net.IOCountersStat, lastTime *time.Time) error {
	// Get current network IO counters
	ioCounters, err := net.IOCounters(true) // true = per interface
	if err != nil {
		return err
	}
	
	now := time.Now()
	
	// If we have previous measurements, calculate rates
	if len(*lastStats) > 0 && !lastTime.IsZero() {
		duration := now.Sub(*lastTime).Seconds()
		
		if duration > 0 {
			for i, stats := range ioCounters {
				if i < len(*lastStats) {
					lastStat := (*lastStats)[i]
					
					if stats.Name == lastStat.Name {
						// Calculate bytes received per second
						recvDiff := stats.BytesRecv - lastStat.BytesRecv
						rm.netIO.WithLabelValues(stats.Name, "received").Set(float64(recvDiff) / duration)
						
						// Calculate bytes sent per second
						sentDiff := stats.BytesSent - lastStat.BytesSent
						rm.netIO.WithLabelValues(stats.Name, "sent").Set(float64(sentDiff) / duration)
					}
				}
			}
		}
	}
	
	// Get connection count
	connections, err := net.Connections("all")
	if err == nil {
		rm.netConnections.Set(float64(len(connections)))
	}
	
	// Store current stats for next collection
	*lastStats = ioCounters
	*lastTime = now
	
	return nil
}

// collectRuntimeMetrics collects Go runtime metrics
func (rm *ResourceMetrics) collectRuntimeMetrics() {
	// Get Go runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Update metrics
	rm.goroutines.Set(float64(runtime.NumGoroutine()))
	rm.heapObjects.Set(float64(memStats.HeapObjects))
	rm.heapAlloc.Set(float64(memStats.HeapAlloc))
	
	// GC pause times are in nanoseconds, convert to seconds
	for _, pause := range memStats.PauseNs {
		if pause > 0 {
			rm.gcPauses.Observe(float64(pause) / 1e9)
		}
	}
}

// collectProcessMetrics collects process-specific metrics
func (rm *ResourceMetrics) collectProcessMetrics() error {
	if rm.processID == 0 {
		return fmt.Errorf("process ID not available")
	}
	
	proc, err := process.NewProcess(rm.processID)
	if err != nil {
		return err
	}
	
	// Number of threads
	numThreads, err := proc.NumThreads()
	if err == nil {
		rm.processThreads.Set(float64(numThreads))
	}
	
	// CPU usage
	cpuPercent, err := proc.CPUPercent()
	if err == nil {
		rm.processCPU.Set(cpuPercent)
	}
	
	// Memory usage
	memInfo, err := proc.MemoryInfo()
	if err == nil {
		rm.processMemory.Set(float64(memInfo.RSS))
	}
	
	// Open file descriptors
	numFDs, err := proc.NumFDs()
	if err == nil {
		rm.processOpenFDs.Set(float64(numFDs))
	}
	
	return nil
}

// GetDiskUsage returns the disk usage for the specified path
func GetDiskUsage(path string) (float64, error) {
	usage, err := disk.Usage(path)
	if err != nil {
		return 0, err
	}
	return usage.UsedPercent, nil
}

// GetCPUUsage returns the current CPU usage percentage
func GetCPUUsage() (float64, error) {
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}
	
	if len(cpuPercent) > 0 {
		return cpuPercent[0], nil
	}
	
	return 0, fmt.Errorf("no CPU usage data available")
}

// GetMemoryUsage returns the current memory usage percentage
func GetMemoryUsage() (float64, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	
	return memInfo.UsedPercent, nil
}
