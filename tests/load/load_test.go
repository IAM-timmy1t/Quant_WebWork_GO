package load

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
)

// TestConfig holds configuration for load tests
type TestConfig struct {
	// General settings
	Duration       time.Duration `json:"duration"`
	Connections    int           `json:"connections"`
	RampUpTime     time.Duration `json:"rampUpTime"`
	ReportInterval time.Duration `json:"reportInterval"`

	// Request settings
	RequestRate      float64 `json:"requestRate"` // Requests per second per connection
	RequestTimeout   time.Duration `json:"requestTimeout"`
	MaxConcurrentReq int      `json:"maxConcurrentRequests"`

	// Bridge settings
	Protocol         string   `json:"protocol"`
	Endpoints        []string `json:"endpoints"`
	ConnectionPoolSize int    `json:"connectionPoolSize"`

	// Output settings
	OutputFile       string   `json:"outputFile"`
	MetricsEnabled   bool     `json:"metricsEnabled"`
	DetailedStats    bool     `json:"detailedStats"`
	ResourceMonitoring bool    `json:"resourceMonitoring"`
}

// RequestResult holds the result of a single request
type RequestResult struct {
	ConnectionID int           `json:"connectionId"`
	StartTime    time.Time     `json:"startTime"`
	EndTime      time.Time     `json:"endTime"`
	Duration     time.Duration `json:"duration"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Endpoint     string        `json:"endpoint"`
}

// TestResult holds the aggregate results of a load test
type TestResult struct {
	// Test configuration
	Config TestConfig `json:"config"`

	// Basic metrics
	TotalRequests         int           `json:"totalRequests"`
	SuccessfulRequests    int           `json:"successfulRequests"`
	FailedRequests        int           `json:"failedRequests"`
	TotalDuration         time.Duration `json:"totalDuration"`
	RequestsPerSecond     float64       `json:"requestsPerSecond"`
	SuccessRate           float64       `json:"successRate"`

	// Latency metrics
	MinLatency            time.Duration `json:"minLatency"`
	MaxLatency            time.Duration `json:"maxLatency"`
	MeanLatency           time.Duration `json:"meanLatency"`
	MedianLatency         time.Duration `json:"medianLatency"`
	P95Latency            time.Duration `json:"p95Latency"`
	P99Latency            time.Duration `json:"p99Latency"`

	// Resource usage
	AvgCPUUsage           float64       `json:"avgCpuUsage,omitempty"`
	MaxCPUUsage           float64       `json:"maxCpuUsage,omitempty"`
	AvgMemoryUsage        uint64        `json:"avgMemoryUsage,omitempty"`
	MaxMemoryUsage        uint64        `json:"maxMemoryUsage,omitempty"`

	// Error distribution
	ErrorDistribution     map[string]int `json:"errorDistribution,omitempty"`

	// Time series data (if detailedStats is true)
	TimeSeriesData        []IntervalStat `json:"timeSeriesData,omitempty"`
}

// IntervalStat holds statistics for a reporting interval
type IntervalStat struct {
	Timestamp          time.Time     `json:"timestamp"`
	Duration           time.Duration `json:"duration"`
	Requests           int           `json:"requests"`
	Successes          int           `json:"successes"`
	Failures           int           `json:"failures"`
	AvgLatency         time.Duration `json:"avgLatency"`
	MaxLatency         time.Duration `json:"maxLatency"`
	CPUUsage           float64       `json:"cpuUsage,omitempty"`
	MemoryUsageBytes   uint64        `json:"memoryUsageBytes,omitempty"`
}

// ResourceUsage tracks system resource usage during the test
type ResourceUsage struct {
	CPUUsage       []float64
	MemoryUsage    []uint64
	Timestamps     []time.Time
	mutex          sync.Mutex
}

// NewDefaultConfig creates a default test configuration
func NewDefaultConfig() TestConfig {
	return TestConfig{
		Duration:           5 * time.Minute,
		Connections:        100,
		RampUpTime:         30 * time.Second,
		ReportInterval:     10 * time.Second,
		RequestRate:        10, // 10 requests per second per connection
		RequestTimeout:     5 * time.Second,
		MaxConcurrentReq:   500,
		Protocol:           "grpc",
		Endpoints:          []string{"/api/v1/bridge/status"},
		ConnectionPoolSize: 100,
		OutputFile:         "load_test_results.json",
		MetricsEnabled:     true,
		DetailedStats:      true,
		ResourceMonitoring: true,
	}
}

// LoadTestRunner manages the execution of load tests
type LoadTestRunner struct {
	Config       TestConfig
	Results      []RequestResult
	resourceUsage ResourceUsage
	metricsCollector *metrics.Collector
	logger       *zap.SugaredLogger
	startTime    time.Time
	endTime      time.Time
	wg           sync.WaitGroup
	resultsLock  sync.Mutex
	intervalData []IntervalStat
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewLoadTestRunner creates a new load test runner
func NewLoadTestRunner(cfg TestConfig, logger *zap.SugaredLogger) *LoadTestRunner {
	ctx, cancel := context.WithCancel(context.Background())
	
	var collector *metrics.Collector
	if cfg.MetricsEnabled {
		metricsConfig := config.MetricsConfig{
			Enabled: true,
			Prometheus: config.PrometheusConfig{
				Enabled: true,
			},
		}
		collector = metrics.NewCollector(metricsConfig, logger)
	}
	
	return &LoadTestRunner{
		Config:          cfg,
		Results:         make([]RequestResult, 0),
		resourceUsage:   ResourceUsage{
			CPUUsage:    make([]float64, 0),
			MemoryUsage: make([]uint64, 0),
			Timestamps:  make([]time.Time, 0),
		},
		metricsCollector: collector,
		logger:          logger,
		intervalData:    make([]IntervalStat, 0),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Run executes the load test
func (r *LoadTestRunner) Run() (*TestResult, error) {
	r.logger.Infow("Starting load test",
		"connections", r.Config.Connections,
		"duration", r.Config.Duration,
		"protocol", r.Config.Protocol)

	// Initialize the result channel and contexts
	resultCh := make(chan RequestResult, r.Config.Connections*10)
	r.startTime = time.Now()

	// Start resource monitoring if enabled
	if r.Config.ResourceMonitoring {
		go r.monitorResources()
	}

	// Start interval reporting
	go r.reportIntervals(resultCh)

	// Create connections and start sending requests
	for i := 0; i < r.Config.Connections; i++ {
		r.wg.Add(1)
		go func(connID int) {
			defer r.wg.Done()
			
			// Implement ramp-up delay
			if r.Config.RampUpTime > 0 && r.Config.Connections > 1 {
				delay := time.Duration(float64(r.Config.RampUpTime) * float64(connID) / float64(r.Config.Connections-1))
				select {
				case <-time.After(delay):
				case <-r.ctx.Done():
					return
				}
			}
			
			// Create client based on protocol
			var client interface{}
			var err error
			
			switch r.Config.Protocol {
			case "grpc":
				client, err = r.createGRPCClient()
			case "rest":
				client, err = r.createRESTClient()
			case "websocket":
				client, err = r.createWebSocketClient()
			default:
				r.logger.Errorw("Unsupported protocol", "protocol", r.Config.Protocol)
				return
			}
			
			if err != nil {
				r.logger.Errorw("Failed to create client",
					"connectionID", connID,
					"protocol", r.Config.Protocol,
					"error", err)
				return
			}
			
			// Send requests using this client
			r.sendRequests(client, connID, resultCh)
		}(i)
	}

	// Collect results in a separate goroutine
	go func() {
		for result := range resultCh {
			r.resultsLock.Lock()
			r.Results = append(r.Results, result)
			r.resultsLock.Unlock()
		}
	}()

	// Wait for test duration to complete
	testTimeout := time.After(r.Config.Duration)
	<-testTimeout
	
	// Cancel the context to stop all goroutines
	r.cancel()
	r.logger.Info("Test duration completed, waiting for all requests to finish...")
	
	// Wait for all connections to complete
	r.wg.Wait()
	close(resultCh)
	
	r.endTime = time.Now()
	r.logger.Infow("Load test completed",
		"totalRequests", len(r.Results),
		"duration", r.endTime.Sub(r.startTime))
	
	// Generate and return test results
	result := r.generateResults()
	
	// Save results to file if specified
	if r.Config.OutputFile != "" {
		if err := r.saveResults(result); err != nil {
			r.logger.Errorw("Failed to save test results", "error", err)
		}
	}
	
	return result, nil
}

// monitorResources collects CPU and memory usage during the test
func (r *LoadTestRunner) monitorResources() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case t := <-ticker.C:
			// Get CPU usage
			cpuPercent, err := cpu.Percent(0, false)
			if err == nil && len(cpuPercent) > 0 {
				r.resourceUsage.mutex.Lock()
				r.resourceUsage.CPUUsage = append(r.resourceUsage.CPUUsage, cpuPercent[0])
				r.resourceUsage.Timestamps = append(r.resourceUsage.Timestamps, t)
				r.resourceUsage.mutex.Unlock()
			}
			
			// Get memory usage
			memInfo, err := mem.VirtualMemory()
			if err == nil {
				r.resourceUsage.mutex.Lock()
				r.resourceUsage.MemoryUsage = append(r.resourceUsage.MemoryUsage, memInfo.Used)
				r.resourceUsage.mutex.Unlock()
			}
		}
	}
}

// reportIntervals collects and reports statistics at regular intervals
func (r *LoadTestRunner) reportIntervals(resultCh <-chan RequestResult) {
	ticker := time.NewTicker(r.Config.ReportInterval)
	defer ticker.Stop()
	
	var intervalResults []RequestResult
	intervalStart := time.Now()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case result := <-resultCh:
			intervalResults = append(intervalResults, result)
		case t := <-ticker.C:
			// Lock to copy the current results and reset for next interval
			r.resultsLock.Lock()
			currentResults := intervalResults
			intervalResults = make([]RequestResult, 0)
			r.resultsLock.Unlock()
			
			// Process current interval data
			if len(currentResults) > 0 {
				stat := r.processIntervalData(currentResults, intervalStart, t)
				r.intervalData = append(r.intervalData, stat)
				
				r.logger.Infow("Interval report",
					"requests", stat.Requests,
					"successes", stat.Successes, 
					"failures", stat.Failures,
					"avgLatency", stat.AvgLatency)
			}
			
			intervalStart = t
		}
	}
}

// processIntervalData calculates statistics for a reporting interval
func (r *LoadTestRunner) processIntervalData(results []RequestResult, start, end time.Time) IntervalStat {
	stat := IntervalStat{
		Timestamp: end,
		Duration:  end.Sub(start),
		Requests:  len(results),
	}
	
	var totalLatency time.Duration
	var maxLatency time.Duration
	
	// Calculate basic stats
	for _, result := range results {
		if result.Success {
			stat.Successes++
			totalLatency += result.Duration
			if result.Duration > maxLatency {
				maxLatency = result.Duration
			}
		} else {
			stat.Failures++
		}
	}
	
	// Calculate average latency
	if stat.Successes > 0 {
		stat.AvgLatency = totalLatency / time.Duration(stat.Successes)
		stat.MaxLatency = maxLatency
	}
	
	// Add resource usage if monitoring is enabled
	if r.Config.ResourceMonitoring {
		r.resourceUsage.mutex.Lock()
		
		// Find CPU and memory usage for this interval
		var totalCPU float64
		var count int
		
		for i, ts := range r.resourceUsage.Timestamps {
			if ts.After(start) && ts.Before(end) || ts.Equal(end) {
				totalCPU += r.resourceUsage.CPUUsage[i]
				count++
			}
		}
		
		if count > 0 {
			stat.CPUUsage = totalCPU / float64(count)
		}
		
		// Get the latest memory usage
		if len(r.resourceUsage.MemoryUsage) > 0 {
			stat.MemoryUsageBytes = r.resourceUsage.MemoryUsage[len(r.resourceUsage.MemoryUsage)-1]
		}
		
		r.resourceUsage.mutex.Unlock()
	}
	
	return stat
}

// createGRPCClient creates a gRPC client for load testing
func (r *LoadTestRunner) createGRPCClient() (*grpc.ClientConn, error) {
	// In a real implementation, this would create a gRPC connection with proper configuration
	return grpc.Dial("localhost:50051", grpc.WithInsecure())
}

// createRESTClient creates a REST client for load testing
func (r *LoadTestRunner) createRESTClient() (interface{}, error) {
	// In a real implementation, this would return an HTTP client
	return &adapters.RESTAdapter{}, nil
}

// createWebSocketClient creates a WebSocket client for load testing
func (r *LoadTestRunner) createWebSocketClient() (interface{}, error) {
	// In a real implementation, this would return a WebSocket client
	return &adapters.WebSocketAdapter{}, nil
}

// sendRequests sends requests at the configured rate using the given client
func (r *LoadTestRunner) sendRequests(client interface{}, connID int, resultCh chan<- RequestResult) {
	// Calculate interval based on request rate
	interval := time.Duration(float64(time.Second) / r.Config.RequestRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			// Select an endpoint randomly from the available endpoints
			endpoint := r.Config.Endpoints[0]
			if len(r.Config.Endpoints) > 1 {
				endpoint = r.Config.Endpoints[connID%len(r.Config.Endpoints)]
			}
			
			// Make the request and record the result
			result := RequestResult{
				ConnectionID: connID,
				StartTime:    time.Now(),
				Endpoint:     endpoint,
			}
			
			// Simulate a request - in a real implementation, this would actually call the service
			var err error
			switch c := client.(type) {
			case *grpc.ClientConn:
				// This would be a real gRPC call
				time.Sleep(50 * time.Millisecond) // Simulate processing time
			case *adapters.RESTAdapter:
				// This would be a real REST call
				time.Sleep(75 * time.Millisecond) // Simulate processing time
			case *adapters.WebSocketAdapter:
				// This would be a real WebSocket call
				time.Sleep(25 * time.Millisecond) // Simulate processing time
			}
			
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.Success = err == nil
			if err != nil {
				result.Error = err.Error()
			}
			
			resultCh <- result
			
			// Record in Prometheus metrics if enabled
			if r.metricsCollector != nil {
				var status string
				if result.Success {
					status = "success"
				} else {
					status = "error"
				}
				r.metricsCollector.RecordHTTPRequest("LOAD_TEST", endpoint, status, result.Duration.Seconds())
			}
		}
	}
}

// generateResults processes all collected results into final statistics
func (r *LoadTestRunner) generateResults() *TestResult {
	result := &TestResult{
		Config:            r.Config,
		TotalRequests:     len(r.Results),
		TotalDuration:     r.endTime.Sub(r.startTime),
		ErrorDistribution: make(map[string]int),
		TimeSeriesData:    r.intervalData,
	}
	
	if result.TotalRequests == 0 {
		return result
	}
	
	// Count successes and failures
	for _, req := range r.Results {
		if req.Success {
			result.SuccessfulRequests++
		} else {
			result.FailedRequests++
			result.ErrorDistribution[req.Error]++
		}
	}
	
	// Calculate success rate
	result.SuccessRate = float64(result.SuccessfulRequests) / float64(result.TotalRequests) * 100
	
	// Calculate requests per second
	result.RequestsPerSecond = float64(result.TotalRequests) / result.TotalDuration.Seconds()
	
	// Calculate latency statistics
	var latencies []time.Duration
	var totalLatency time.Duration
	result.MinLatency = time.Hour // Start with a large value
	
	for _, req := range r.Results {
		if !req.Success {
			continue
		}
		
		latencies = append(latencies, req.Duration)
		totalLatency += req.Duration
		
		if req.Duration < result.MinLatency {
			result.MinLatency = req.Duration
		}
		if req.Duration > result.MaxLatency {
			result.MaxLatency = req.Duration
		}
	}
	
	// Sort latencies for percentile calculations
	if len(latencies) > 0 {
		result.MeanLatency = totalLatency / time.Duration(len(latencies))
		
		// Calculate median (50th percentile)
		result.MedianLatency = calculatePercentile(latencies, 50)
		
		// Calculate 95th percentile
		result.P95Latency = calculatePercentile(latencies, 95)
		
		// Calculate 99th percentile
		result.P99Latency = calculatePercentile(latencies, 99)
	}
	
	// Calculate resource usage statistics if monitoring was enabled
	if r.Config.ResourceMonitoring {
		r.resourceUsage.mutex.Lock()
		defer r.resourceUsage.mutex.Unlock()
		
		if len(r.resourceUsage.CPUUsage) > 0 {
			var totalCPU float64
			var maxCPU float64
			
			for _, cpu := range r.resourceUsage.CPUUsage {
				totalCPU += cpu
				if cpu > maxCPU {
					maxCPU = cpu
				}
			}
			
			result.AvgCPUUsage = totalCPU / float64(len(r.resourceUsage.CPUUsage))
			result.MaxCPUUsage = maxCPU
		}
		
		if len(r.resourceUsage.MemoryUsage) > 0 {
			var totalMem uint64
			var maxMem uint64
			
			for _, mem := range r.resourceUsage.MemoryUsage {
				totalMem += mem
				if mem > maxMem {
					maxMem = mem
				}
			}
			
			result.AvgMemoryUsage = totalMem / uint64(len(r.resourceUsage.MemoryUsage))
			result.MaxMemoryUsage = maxMem
		}
	}
	
	return result
}

// saveResults saves the test results to a JSON file
func (r *LoadTestRunner) saveResults(result *TestResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	
	return os.WriteFile(r.Config.OutputFile, data, 0644)
}

// calculatePercentile calculates the specified percentile from sorted latency data
func calculatePercentile(sorted []time.Duration, percentile float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	
	index := int(math.Ceil(float64(len(sorted)) * percentile / 100.0)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}
