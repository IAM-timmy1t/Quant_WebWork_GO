package monitoring

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/storage"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/tracing"
)

// SecurityMonitor manages security event monitoring and risk assessment
type SecurityMonitor struct {
	mu sync.RWMutex

	config SecurityConfig
	tracer *tracing.Tracer
	storage *storage.MetricsStorage

	// Event processing
	eventChan chan SecurityEvent
	stopChan  chan struct{}

	// Subscribers for real-time events
	subscribers map[chan<- SecurityEvent]bool

	// Risk tracking
	riskScores     map[string]float64
	riskThresholds map[string]float64
}

// SecurityConfig defines configuration for security monitoring
type SecurityConfig struct {
	Enabled bool `json:"enabled"`

	// Event processing
	MaxEventBuffer int           `json:"maxEventBuffer"`
	BatchSize     int           `json:"batchSize"`
	FlushInterval time.Duration `json:"flushInterval"`

	// Risk assessment
	BaseRiskScore     float64 `json:"baseRiskScore"`
	RiskDecayRate     float64 `json:"riskDecayRate"`
	RiskDecayInterval time.Duration `json:"riskDecayInterval"`
	
	// Thresholds for different severity levels
	CriticalThreshold float64 `json:"criticalThreshold"`
	HighThreshold     float64 `json:"highThreshold"`
	MediumThreshold   float64 `json:"mediumThreshold"`
	LowThreshold      float64 `json:"lowThreshold"`

	// Process security
	MonitorProcessSecurity bool          `json:"monitorProcessSecurity"`
	SuspiciousProcesses   []string      `json:"suspiciousProcesses"`
	MaxFileDescriptors    int           `json:"maxFileDescriptors"`
	
	// Network security
	MonitorNetworkSecurity bool     `json:"monitorNetworkSecurity"`
	BlockedPorts          []int     `json:"blockedPorts"`
	BlockedIPs           []string   `json:"blockedIPs"`
	
	// File system security
	MonitorFileSystem     bool     `json:"monitorFileSystem"`
	SensitivePaths       []string `json:"sensitivePaths"`
	FileChangeInterval   time.Duration `json:"fileChangeInterval"`
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	RiskScore float64   `json:"riskScore"`
	Timestamp time.Time `json:"timestamp"`
	Category  string    `json:"category"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessSecurityEvent represents a process-related security event
type ProcessSecurityEvent struct {
	SecurityEvent
	ProcessID   int32  `json:"processId"`
	ProcessName string `json:"processName"`
	CommandLine string `json:"commandLine"`
	User        string `json:"user"`
	CPU         float64 `json:"cpu"`
	Memory      uint64  `json:"memory"`
}

// NetworkSecurityEvent represents a network-related security event
type NetworkSecurityEvent struct {
	SecurityEvent
	Protocol    string `json:"protocol"`
	LocalAddr   string `json:"localAddr"`
	RemoteAddr  string `json:"remoteAddr"`
	LocalPort   int    `json:"localPort"`
	RemotePort  int    `json:"remotePort"`
	BytesSent   uint64 `json:"bytesSent"`
	BytesRecv   uint64 `json:"bytesRecv"`
}

// FileSystemEvent represents a file system security event
type FileSystemEvent struct {
	SecurityEvent
	Path        string    `json:"path"`
	Operation   string    `json:"operation"`
	Size        int64     `json:"size"`
	Permissions string    `json:"permissions"`
	ModTime     time.Time `json:"modTime"`
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor(config SecurityConfig, storage *storage.MetricsStorage, tracer *tracing.Tracer) *SecurityMonitor {
	if config.MaxEventBuffer <= 0 {
		config.MaxEventBuffer = 1000
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	sm := &SecurityMonitor{
		config:     config,
		storage:    storage,
		tracer:     tracer,
		eventChan:  make(chan SecurityEvent, config.MaxEventBuffer),
		stopChan:   make(chan struct{}),
		subscribers: make(map[chan<- SecurityEvent]bool),
		riskScores: make(map[string]float64),
		riskThresholds: map[string]float64{
			"critical": config.CriticalThreshold,
			"high":     config.HighThreshold,
			"medium":   config.MediumThreshold,
			"low":      config.LowThreshold,
		},
	}

	return sm
}

// Start begins security monitoring
func (sm *SecurityMonitor) Start(ctx context.Context) error {
	if !sm.config.Enabled {
		return nil
	}

	// Start event processing
	go sm.processEvents(ctx)

	// Start risk score decay
	go sm.decayRiskScores(ctx)

	// Start process monitoring
	go sm.MonitorProcesses(ctx)

	// Start network monitoring
	go sm.MonitorNetwork(ctx)

	return nil
}

// Stop stops security monitoring
func (sm *SecurityMonitor) Stop() {
	close(sm.stopChan)
}

// Subscribe adds a subscriber for security events
func (sm *SecurityMonitor) Subscribe(ch chan<- SecurityEvent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.subscribers[ch] = true
}

// Unsubscribe removes a subscriber
func (sm *SecurityMonitor) Unsubscribe(ch chan<- SecurityEvent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.subscribers, ch)
}

// RecordEvent records a security event
func (sm *SecurityMonitor) RecordEvent(ctx context.Context, event SecurityEvent) error {
	// Calculate risk score if not provided
	if event.RiskScore == 0 {
		event.RiskScore = sm.calculateRiskScore(event)
	}

	// Update risk scores
	sm.updateRiskScore(event.Source, event.RiskScore)

	// Send to processing channel
	select {
	case sm.eventChan <- event:
		// Broadcast to subscribers
		sm.broadcastEvent(event)
		return nil
	default:
		return fmt.Errorf("event channel full")
	}
}

// GetRiskScore returns the current risk score for a source
func (sm *SecurityMonitor) GetRiskScore(source string) float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.riskScores[source]
}

// processEvents handles the processing and storage of security events
func (sm *SecurityMonitor) processEvents(ctx context.Context) {
	batch := make([]SecurityEvent, 0, sm.config.BatchSize)
	ticker := time.NewTicker(sm.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event := <-sm.eventChan:
			batch = append(batch, event)
			if len(batch) >= sm.config.BatchSize {
				if err := sm.flushEventBatch(ctx, batch); err != nil {
					sm.tracer.RecordError(ctx, fmt.Errorf("error flushing event batch: %v", err))
				}
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				if err := sm.flushEventBatch(ctx, batch); err != nil {
					sm.tracer.RecordError(ctx, fmt.Errorf("error flushing event batch: %v", err))
				}
				batch = batch[:0]
			}

		case <-sm.stopChan:
			if len(batch) > 0 {
				if err := sm.flushEventBatch(ctx, batch); err != nil {
					sm.tracer.RecordError(ctx, fmt.Errorf("error flushing final event batch: %v", err))
				}
			}
			return

		case <-ctx.Done():
			return
		}
	}
}

// flushEventBatch stores a batch of security events
func (sm *SecurityMonitor) flushEventBatch(ctx context.Context, batch []SecurityEvent) error {
	for _, event := range batch {
		if err := sm.storage.StoreSecurityEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to store security event: %v", err)
		}
	}
	return nil
}

// broadcastEvent sends an event to all subscribers
func (sm *SecurityMonitor) broadcastEvent(event SecurityEvent) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for ch := range sm.subscribers {
		select {
		case ch <- event:
		default:
			// Skip if channel is blocked
		}
	}
}

// updateRiskScore updates the risk score for a source
func (sm *SecurityMonitor) updateRiskScore(source string, score float64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	currentScore := sm.riskScores[source]
	sm.riskScores[source] = currentScore + score
}

// calculateRiskScore calculates a risk score for an event
func (sm *SecurityMonitor) calculateRiskScore(event SecurityEvent) float64 {
	baseScore := sm.config.BaseRiskScore

	// Adjust score based on severity
	switch event.Severity {
	case "critical":
		baseScore *= 4.0
	case "high":
		baseScore *= 2.0
	case "medium":
		baseScore *= 1.5
	case "low":
		baseScore *= 1.0
	}

	return baseScore
}

// decayRiskScores periodically reduces risk scores
func (sm *SecurityMonitor) decayRiskScores(ctx context.Context) {
	ticker := time.NewTicker(sm.config.RiskDecayInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.mu.Lock()
			for source := range sm.riskScores {
				sm.riskScores[source] *= (1.0 - sm.config.RiskDecayRate)
				if sm.riskScores[source] < sm.riskThresholds["low"] {
					delete(sm.riskScores, source)
				}
			}
			sm.mu.Unlock()

		case <-sm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// GetSecurityStatus returns the current security status
func (sm *SecurityMonitor) GetSecurityStatus() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := make(map[string]interface{})
	highestRisk := 0.0
	criticalSources := make([]string, 0)
	highSources := make([]string, 0)

	for source, score := range sm.riskScores {
		if score > highestRisk {
			highestRisk = score
		}

		if score >= sm.riskThresholds["critical"] {
			criticalSources = append(criticalSources, source)
		} else if score >= sm.riskThresholds["high"] {
			highSources = append(highSources, source)
		}
	}

	status["highest_risk_score"] = highestRisk
	status["critical_sources"] = criticalSources
	status["high_risk_sources"] = highSources
	status["total_sources"] = len(sm.riskScores)

	return status
}

// MonitorProcesses monitors process-related security events
func (sm *SecurityMonitor) MonitorProcesses(ctx context.Context) {
	if !sm.config.MonitorProcessSecurity {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processes, err := process.Processes()
			if err != nil {
				sm.tracer.RecordError(ctx, fmt.Errorf("error monitoring processes: %v", err))
				continue
			}

			for _, p := range processes {
				info, err := p.Info()
				if err != nil {
					continue
				}

				// Check for suspicious processes
				for _, suspicious := range sm.config.SuspiciousProcesses {
					if info.Name == suspicious {
						event := ProcessSecurityEvent{
							SecurityEvent: SecurityEvent{
								ID:        fmt.Sprintf("process_%d_%s", p.Pid, time.Now().Format(time.RFC3339)),
								Source:    "process_monitor",
								Type:      "suspicious_process",
								Severity:  "high",
								Message:   fmt.Sprintf("Suspicious process detected: %s", info.Name),
								Timestamp: time.Now(),
								Category:  "process",
								Action:    "detect",
								Status:    "active",
							},
							ProcessID:   p.Pid,
							ProcessName: info.Name,
							CommandLine: info.CommandLine,
							User:       info.Username,
							CPU:        info.CPUPercent,
							Memory:     info.MemoryPercent,
						}
						sm.RecordEvent(ctx, event.SecurityEvent)
					}
				}

				// Check for excessive resource usage
				if info.CPUPercent > 90 || info.MemoryPercent > 90 {
					event := ProcessSecurityEvent{
						SecurityEvent: SecurityEvent{
							ID:        fmt.Sprintf("process_resource_%d_%s", p.Pid, time.Now().Format(time.RFC3339)),
							Source:    "process_monitor",
							Type:      "excessive_resource_usage",
							Severity:  "medium",
							Message:   fmt.Sprintf("Process %s using excessive resources", info.Name),
							Timestamp: time.Now(),
							Category:  "process",
							Action:    "alert",
							Status:    "active",
						},
						ProcessID:   p.Pid,
						ProcessName: info.Name,
						CommandLine: info.CommandLine,
						User:       info.Username,
						CPU:        info.CPUPercent,
						Memory:     info.MemoryPercent,
					}
					sm.RecordEvent(ctx, event.SecurityEvent)
				}
			}
		}
	}
}

// MonitorNetwork monitors network-related security events
func (sm *SecurityMonitor) MonitorNetwork(ctx context.Context) {
	if !sm.config.MonitorNetworkSecurity {
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			connections, err := net.Connections("all")
			if err != nil {
				sm.tracer.RecordError(ctx, fmt.Errorf("error monitoring network: %v", err))
				continue
			}

			for _, conn := range connections {
				// Check for blocked ports
				for _, port := range sm.config.BlockedPorts {
					if int(conn.Laddr.Port) == port || int(conn.Raddr.Port) == port {
						event := NetworkSecurityEvent{
							SecurityEvent: SecurityEvent{
								ID:        fmt.Sprintf("network_%d_%s", conn.Pid, time.Now().Format(time.RFC3339)),
								Source:    "network_monitor",
								Type:      "blocked_port_access",
								Severity:  "high",
								Message:   fmt.Sprintf("Access attempt to blocked port %d", port),
								Timestamp: time.Now(),
								Category:  "network",
								Action:    "block",
								Status:    "detected",
							},
							Protocol:    conn.Type,
							LocalAddr:   conn.Laddr.IP,
							RemoteAddr:  conn.Raddr.IP,
							LocalPort:   int(conn.Laddr.Port),
							RemotePort:  int(conn.Raddr.Port),
						}
						sm.RecordEvent(ctx, event.SecurityEvent)
					}
				}

				// Check for blocked IPs
				for _, ip := range sm.config.BlockedIPs {
					if conn.Raddr.IP == ip {
						event := NetworkSecurityEvent{
							SecurityEvent: SecurityEvent{
								ID:        fmt.Sprintf("network_ip_%s_%s", ip, time.Now().Format(time.RFC3339)),
								Source:    "network_monitor",
								Type:      "blocked_ip_access",
								Severity:  "critical",
								Message:   fmt.Sprintf("Access attempt from blocked IP %s", ip),
								Timestamp: time.Now(),
								Category:  "network",
								Action:    "block",
								Status:    "detected",
							},
							Protocol:    conn.Type,
							LocalAddr:   conn.Laddr.IP,
							RemoteAddr:  conn.Raddr.IP,
							LocalPort:   int(conn.Laddr.Port),
							RemotePort:  int(conn.Raddr.Port),
						}
						sm.RecordEvent(ctx, event.SecurityEvent)
					}
				}
			}
		}
	}
}

// MonitorFileSystem monitors file system-related security events
func (sm *SecurityMonitor) MonitorFileSystem(ctx context.Context) {
	if !sm.config.MonitorFileSystem {
		return
	}

	ticker := time.NewTicker(sm.config.FileChangeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, path := range sm.config.SensitivePaths {
				fi, err := os.Stat(path)
				if err != nil {
					sm.tracer.RecordError(ctx, fmt.Errorf("error monitoring file system: %v", err))
					continue
				}

				// Check for file changes
				if fi.ModTime().After(time.Now().Add(-sm.config.FileChangeInterval)) {
					event := FileSystemEvent{
						SecurityEvent: SecurityEvent{
							ID:        fmt.Sprintf("file_system_%s_%s", path, time.Now().Format(time.RFC3339)),
							Source:    "file_system_monitor",
							Type:      "file_change",
							Severity:  "medium",
							Message:   fmt.Sprintf("File change detected: %s", path),
							Timestamp: time.Now(),
							Category:  "file_system",
							Action:    "alert",
							Status:    "active",
						},
						Path:        path,
						Operation:   "modify",
						Size:        fi.Size(),
						Permissions: fi.Mode().String(),
						ModTime:     fi.ModTime(),
					}
					sm.RecordEvent(ctx, event.SecurityEvent)
				}
			}
		}
	}
}

