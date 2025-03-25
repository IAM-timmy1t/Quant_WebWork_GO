// monitor.go - Advanced security monitoring

package security

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
)

// Config defines configuration for the security monitor
type Config struct {
    MaxEventsHistory    int           // Maximum number of events to keep in history
    AlertThreshold      int           // Risk score threshold to trigger alerts
    DetectorInterval    time.Duration // How often to run detectors
    DetectorTimeout     time.Duration // Maximum time for detection cycle
    RiskFactors         map[string]float64 // Risk factors by category
    ThresholdLevels     map[RiskLevel]float64 // Thresholds for different risk levels
    EnabledDetectors    []string      // List of enabled detectors
}

// DefaultConfig returns a default security monitor configuration
func DefaultConfig() Config {
    return Config{
        MaxEventsHistory: 1000,
        AlertThreshold:   70,
        DetectorInterval: 5 * time.Minute,
        DetectorTimeout:  30 * time.Second,
        RiskFactors: map[string]float64{
            "authentication": 1.5,
            "authorization":  1.2,
            "data_access":    1.3,
            "api_abuse":      1.4,
        },
        ThresholdLevels: map[RiskLevel]float64{
            RiskLow:      30.0,
            RiskMedium:   50.0,
            RiskHigh:     70.0,
            RiskCritical: 90.0,
        },
        EnabledDetectors: []string{
            "brute_force",
            "anomaly",
            "vulnerability",
        },
    }
}

// Monitor implements security monitoring and event processing
type Monitor struct {
    config         Config
    events         []Event
    detectors      map[string]Detector
    riskAnalyzer   RiskAnalyzerImpl
    eventProcessor EventProcessorImpl
    alertManager   AlertManagerImpl
    subscribers    []chan<- Alert
    mu             sync.RWMutex
    ctx            context.Context
    cancel         context.CancelFunc
}

// NewMonitor creates a new security monitor
func NewMonitor(config Config) (*Monitor, error) {
    ctx, cancel := context.WithCancel(context.Background())
    
    m := &Monitor{
        config:         config,
        events:         make([]Event, 0, config.MaxEventsHistory),
        detectors:      make(map[string]Detector),
        riskAnalyzer:   NewRiskAnalyzer(config.RiskFactors, config.ThresholdLevels),
        eventProcessor: NewEventProcessor(config.MaxEventsHistory),
        alertManager:   NewAlertManager(),
        subscribers:    make([]chan<- Alert, 0),
        ctx:            ctx,
        cancel:         cancel,
    }
    
    // Register default detectors
    if err := m.registerDefaultDetectors(); err != nil {
        cancel()
        return nil, err
    }
    
    // Start background processing
    go m.runDetectors()
    
    return m, nil
}

// ProcessEvent processes a security event
func (m *Monitor) ProcessEvent(ctx context.Context, event Event) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Ensure event has an ID and timestamp
    if event.Type == "" {
        event.Type = "unknown"
    }
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }
    
    // Process the event with the event processor
    enhanced, err := m.eventProcessor.Process(ctx, event)
    if err != nil {
        return fmt.Errorf("failed to process event: %w", err)
    }
    
    // Analyze risk
    analysisResult, err := m.riskAnalyzer.AnalyzeRisk(ctx, event, enhanced.UserHistory)
    if err != nil {
        return fmt.Errorf("failed to analyze risk: %w", err)
    }
    
    // Update enhanced event with analysis result
    enhanced.Analysis = analysisResult
    
    // Store the event
    m.eventProcessor.StoreEvent(enhanced)
    
    // Add to local history
    m.events = append(m.events, event)
    if len(m.events) > m.config.MaxEventsHistory {
        m.events = m.events[1:]
    }
    
    // Create metrics for monitoring
    m.recordMetrics(enhanced)
    
    // Check if alert threshold is reached
    if analysisResult.Score >= float64(m.config.AlertThreshold) {
        alert, err := m.alertManager.CreateAlert(ctx, enhanced)
        if err != nil {
            return fmt.Errorf("failed to create alert: %w", err)
        }
        
        // Notify subscribers
        m.notifySubscribers(alert)
    }
    
    return nil
}

// Subscribe adds a subscriber channel for alerts
func (m *Monitor) Subscribe(ch chan<- Alert) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.subscribers = append(m.subscribers, ch)
}

// Unsubscribe removes a subscriber channel
func (m *Monitor) Unsubscribe(ch chan<- Alert) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    for i, sub := range m.subscribers {
        if sub == ch {
            m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
            return
        }
    }
}

// Close stops the security monitor
func (m *Monitor) Close() error {
    m.cancel()
    return nil
}

// RegisterDetector registers a custom detector
func (m *Monitor) RegisterDetector(name string, detector Detector) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.detectors[name] = detector
}

// GetEventHistory returns recent security events
func (m *Monitor) GetEventHistory(limit int) []Event {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if limit <= 0 || limit > len(m.events) {
        limit = len(m.events)
    }
    
    result := make([]Event, limit)
    start := len(m.events) - limit
    copy(result, m.events[start:])
    
    return result
}

// registerDefaultDetectors registers the default security detectors
func (m *Monitor) registerDefaultDetectors() error {
    for _, name := range m.config.EnabledDetectors {
        var detector Detector
        
        switch name {
        case "brute_force":
            detector = NewBruteForceDetector(BruteForceDetectorConfig{
                MaxAttempts:        5,
                WindowDuration:     15 * time.Minute,
                LockoutDuration:    30 * time.Minute,
                ResetAfterSuccess:  true,
            })
        case "anomaly":
            detector = NewAnomalyDetector()
        case "vulnerability":
            detector = NewVulnerabilityDetector()
        default:
            return fmt.Errorf("unknown detector type: %s", name)
        }
        
        m.detectors[name] = detector
    }
    
    return nil
}

// runDetectors runs security detectors at scheduled intervals
func (m *Monitor) runDetectors() {
    ticker := time.NewTicker(m.config.DetectorInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            m.executeDetectors()
        case <-m.ctx.Done():
            return
        }
    }
}

// executeDetectors runs all registered detectors
func (m *Monitor) executeDetectors() {
    ctx, cancel := context.WithTimeout(m.ctx, m.config.DetectorTimeout)
    defer cancel()
    
    var wg sync.WaitGroup
    
    m.mu.RLock()
    detectors := make(map[string]Detector, len(m.detectors))
    for name, detector := range m.detectors {
        detectors[name] = detector
    }
    m.mu.RUnlock()
    
    for name, detector := range detectors {
        wg.Add(1)
        
        go func(name string, d Detector) {
            defer wg.Done()
            
            events, err := d.Detect(ctx)
            if err != nil {
                log.Printf("Detector %s failed: %v", name, err)
                return
            }
            
            // Process detected events
            for _, event := range events {
                event.Source = name
                if err := m.ProcessEvent(ctx, event); err != nil {
                    log.Printf("Failed to process detected event: %v", err)
                }
            }
        }(name, detector)
    }
    
    wg.Wait()
}

// notifySubscribers sends an alert to all subscribers
func (m *Monitor) notifySubscribers(alert Alert) {
    for _, ch := range m.subscribers {
        select {
        case ch <- alert:
            // Alert sent successfully
        default:
            // Channel is full or closed, log and continue
            log.Printf("Failed to send alert to subscriber")
        }
    }
}

// recordMetrics records security metrics
func (m *Monitor) recordMetrics(event EnhancedEvent) {
    // This would integrate with the metrics system
    // For example: metrics.RecordMetric("security_events", 1.0, map[string]string{"type": string(event.Event.Type)})
}
