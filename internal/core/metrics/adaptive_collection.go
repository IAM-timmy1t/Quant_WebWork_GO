// adaptive_collection.go - Adaptive metrics collection based on system load

package metrics

import (
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// AdaptiveCollector provides adaptive collection of metrics based on system load
type AdaptiveCollector struct {
	// Configuration
	baseInterval       time.Duration
	minInterval        time.Duration
	maxInterval        time.Duration
	loadThresholdLow   float64
	loadThresholdHigh  float64
	adaptationRate     float64
	
	// Current state
	currentInterval    time.Duration
	loadHistory        []float64
	historyMaxSize     int
	
	// Metrics
	collectionInterval prometheus.Gauge
	collectionsMade    prometheus.Counter
	collectionsSkipped prometheus.Counter
	adaptationEvents   prometheus.Counter
	
	// Resources
	resourceMetrics    *ResourceMetrics
	
	// Control
	logger             *zap.SugaredLogger
	mutex              sync.RWMutex
	collectors         []func()
	stopCh             chan struct{}
	collectionActive   bool
}

// AdaptiveCollectorConfig holds the configuration for the adaptive collector
type AdaptiveCollectorConfig struct {
	BaseInterval       time.Duration
	MinInterval        time.Duration
	MaxInterval        time.Duration
	LoadThresholdLow   float64
	LoadThresholdHigh  float64
	AdaptationRate     float64
	HistorySize        int
}

// DefaultAdaptiveConfig returns a default configuration for the adaptive collector
func DefaultAdaptiveConfig() AdaptiveCollectorConfig {
	return AdaptiveCollectorConfig{
		BaseInterval:      time.Second * 15,
		MinInterval:       time.Second * 5,
		MaxInterval:       time.Minute * 2,
		LoadThresholdLow:  30.0, // CPU percentage
		LoadThresholdHigh: 70.0, // CPU percentage
		AdaptationRate:    0.2,  // How quickly to adapt (0-1)
		HistorySize:       10,   // Number of load measurements to keep
	}
}

// NewAdaptiveCollector creates a new adaptive metrics collector
func NewAdaptiveCollector(config AdaptiveCollectorConfig, resourceMetrics *ResourceMetrics, logger *zap.SugaredLogger) *AdaptiveCollector {
	ac := &AdaptiveCollector{
		baseInterval:      config.BaseInterval,
		minInterval:       config.MinInterval,
		maxInterval:       config.MaxInterval,
		loadThresholdLow:  config.LoadThresholdLow,
		loadThresholdHigh: config.LoadThresholdHigh,
		adaptationRate:    config.AdaptationRate,
		currentInterval:   config.BaseInterval,
		historyMaxSize:    config.HistorySize,
		loadHistory:       make([]float64, 0, config.HistorySize),
		resourceMetrics:   resourceMetrics,
		logger:            logger,
		stopCh:            make(chan struct{}),
		collectors:        make([]func(), 0),
	}
	
	// Initialize metrics
	ac.collectionInterval = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "metrics_collection_interval_seconds",
		Help: "Current interval between metrics collections in seconds",
	})
	
	ac.collectionsMade = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "metrics_collections_total",
		Help: "Total number of metric collections performed",
	})
	
	ac.collectionsSkipped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "metrics_collections_skipped_total",
		Help: "Total number of metric collections skipped due to high load",
	})
	
	ac.adaptationEvents = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "metrics_adaptation_events_total",
		Help: "Total number of times the collection interval was adapted",
	})
	
	// Register metrics
	prometheus.MustRegister(
		ac.collectionInterval,
		ac.collectionsMade,
		ac.collectionsSkipped,
		ac.adaptationEvents,
	)
	
	return ac
}

// AddCollector adds a collection function to be called on the adaptive schedule
func (ac *AdaptiveCollector) AddCollector(collector func()) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	
	ac.collectors = append(ac.collectors, collector)
}

// Start begins the adaptive collection process
func (ac *AdaptiveCollector) Start() {
	ac.mutex.Lock()
	if ac.collectionActive {
		ac.mutex.Unlock()
		return
	}
	ac.collectionActive = true
	ac.mutex.Unlock()
	
	go ac.runCollectionLoop()
}

// Stop halts the adaptive collection process
func (ac *AdaptiveCollector) Stop() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	
	if !ac.collectionActive {
		return
	}
	
	close(ac.stopCh)
	ac.collectionActive = false
}

// runCollectionLoop is the main collection loop
func (ac *AdaptiveCollector) runCollectionLoop() {
	ticker := time.NewTicker(ac.currentInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Check system load
			cpuLoad, err := GetCPUUsage()
			if err != nil {
				ac.logger.Warnw("Failed to get CPU usage for adaptive collection", "error", err)
				cpuLoad = 50.0 // Default to middle value if we can't get actual CPU
			}
			
			// Update load history
			ac.updateLoadHistory(cpuLoad)
			
			// If system is under heavy load, consider skipping collection
			if ac.shouldSkipCollection() {
				ac.collectionsSkipped.Inc()
				ac.logger.Debugw("Skipping metrics collection due to high system load",
					"avgLoad", ac.getAverageLoad(),
					"threshold", ac.loadThresholdHigh)
			} else {
				// Perform collection
				ac.performCollection()
				ac.collectionsMade.Inc()
			}
			
			// Adapt collection interval based on load
			newInterval := ac.adaptInterval(ac.getAverageLoad())
			if newInterval != ac.currentInterval {
				ac.mutex.Lock()
				ac.currentInterval = newInterval
				ac.mutex.Unlock()
				
				// Update the ticker
				ticker.Reset(newInterval)
				
				// Update metrics
				ac.collectionInterval.Set(float64(newInterval) / float64(time.Second))
				ac.adaptationEvents.Inc()
				
				ac.logger.Debugw("Adapted metrics collection interval",
					"newInterval", newInterval.String(),
					"avgLoad", ac.getAverageLoad())
			}
			
		case <-ac.stopCh:
			return
		}
	}
}

// updateLoadHistory adds a new load measurement to the history
func (ac *AdaptiveCollector) updateLoadHistory(load float64) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	
	// Add the new measurement
	ac.loadHistory = append(ac.loadHistory, load)
	
	// Trim history if it exceeds max size
	if len(ac.loadHistory) > ac.historyMaxSize {
		ac.loadHistory = ac.loadHistory[1:]
	}
}

// getAverageLoad calculates the average system load from history
func (ac *AdaptiveCollector) getAverageLoad() float64 {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	
	if len(ac.loadHistory) == 0 {
		return 50.0 // Default to middle value if no history
	}
	
	var sum float64
	for _, load := range ac.loadHistory {
		sum += load
	}
	
	return sum / float64(len(ac.loadHistory))
}

// shouldSkipCollection determines if collection should be skipped due to high load
func (ac *AdaptiveCollector) shouldSkipCollection() bool {
	avgLoad := ac.getAverageLoad()
	
	// Skip if average load is significantly above high threshold
	return avgLoad > (ac.loadThresholdHigh * 1.2)
}

// adaptInterval calculates a new collection interval based on system load
func (ac *AdaptiveCollector) adaptInterval(load float64) time.Duration {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	
	// Use simple linear function to adjust interval based on load:
	// - Below low threshold: decrease interval (collect more frequently)
	// - Above high threshold: increase interval (collect less frequently)
	// - Between thresholds: stay at base interval
	
	var targetInterval time.Duration
	
	if load < ac.loadThresholdLow {
		// System is lightly loaded, decrease interval
		loadFactor := math.Max(0, load/ac.loadThresholdLow)
		intervalRange := float64(ac.baseInterval - ac.minInterval)
		reduction := time.Duration(intervalRange * (1 - loadFactor))
		targetInterval = ac.baseInterval - reduction
	} else if load > ac.loadThresholdHigh {
		// System is heavily loaded, increase interval
		overloadFactor := math.Min(1, (load-ac.loadThresholdHigh)/(100-ac.loadThresholdHigh))
		intervalRange := float64(ac.maxInterval - ac.baseInterval)
		increase := time.Duration(intervalRange * overloadFactor)
		targetInterval = ac.baseInterval + increase
	} else {
		// System load is in the normal range, use base interval
		targetInterval = ac.baseInterval
	}
	
	// Gradually move toward the target (smooth adaptation)
	if targetInterval == ac.currentInterval {
		return ac.currentInterval
	}
	
	intervalDiff := float64(targetInterval - ac.currentInterval)
	adjustment := time.Duration(intervalDiff * ac.adaptationRate)
	
	// Ensure we make at least some minimal adjustment
	if adjustment == 0 {
		if targetInterval > ac.currentInterval {
			adjustment = 1 * time.Millisecond
		} else {
			adjustment = -1 * time.Millisecond
		}
	}
	
	newInterval := ac.currentInterval + adjustment
	
	// Ensure we stay within bounds
	if newInterval < ac.minInterval {
		newInterval = ac.minInterval
	} else if newInterval > ac.maxInterval {
		newInterval = ac.maxInterval
	}
	
	return newInterval
}

// performCollection executes all registered collectors
func (ac *AdaptiveCollector) performCollection() {
	ac.mutex.RLock()
	collectors := make([]func(), len(ac.collectors))
	copy(collectors, ac.collectors)
	ac.mutex.RUnlock()
	
	// Execute each collector
	for _, collector := range collectors {
		// We could run these in separate goroutines, but that might
		// defeat the purpose of adaptive collection by creating more load
		collector()
	}
}

// GetCurrentInterval returns the current collection interval
func (ac *AdaptiveCollector) GetCurrentInterval() time.Duration {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	return ac.currentInterval
}
