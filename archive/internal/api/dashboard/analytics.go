package dashboard

import (
	"math"
	"sort"
	"sync"
	"time"
)

// AnalyticsEngine handles advanced metrics analysis and anomaly detection
type AnalyticsEngine struct {
	mu sync.RWMutex

	// Configuration
	config AnalyticsConfig

	// Historical data
	systemHistory    []SystemMetrics
	serviceHistory   map[string][]ServiceMetrics
	anomalyHistory  map[string][]Anomaly
	
	// Analysis components
	detectors map[string]AnomalyDetector
}

// AnalyticsConfig defines configuration for analytics
type AnalyticsConfig struct {
	HistoryWindow      time.Duration `json:"historyWindow"`
	AnomalyThreshold   float64      `json:"anomalyThreshold"`
	MinDataPoints      int          `json:"minDataPoints"`
	UpdateInterval     time.Duration `json:"updateInterval"`
	EnablePrediction   bool         `json:"enablePrediction"`
	PredictionWindow   time.Duration `json:"predictionWindow"`
}

// Anomaly represents a detected system anomaly
type Anomaly struct {
	Timestamp    time.Time `json:"timestamp"`
	Source      string    `json:"source"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
}

// AnomalyDetector interface for different detection algorithms
type AnomalyDetector interface {
	Detect(data []float64) []Anomaly
	Train(data []float64)
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine(config AnalyticsConfig) *AnalyticsEngine {
	return &AnalyticsEngine{
		config:         config,
		serviceHistory: make(map[string][]ServiceMetrics),
		anomalyHistory: make(map[string][]Anomaly),
		detectors:      make(map[string]AnomalyDetector),
	}
}

// Start begins the analytics processing
func (ae *AnalyticsEngine) Start() {
	ticker := time.NewTicker(ae.config.UpdateInterval)
	defer ticker.Stop()

	for range ticker.C {
		ae.analyze()
	}
}

// AddSystemMetrics adds new system metrics for analysis
func (ae *AnalyticsEngine) AddSystemMetrics(metrics SystemMetrics) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ae.systemHistory = append(ae.systemHistory, metrics)
	ae.trimHistory()
}

// AddServiceMetrics adds new service metrics for analysis
func (ae *AnalyticsEngine) AddServiceMetrics(serviceID string, metrics ServiceMetrics) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if _, exists := ae.serviceHistory[serviceID]; !exists {
		ae.serviceHistory[serviceID] = make([]ServiceMetrics, 0)
	}
	ae.serviceHistory[serviceID] = append(ae.serviceHistory[serviceID], metrics)
}

// GetAnomalies returns detected anomalies for a given time range
func (ae *AnalyticsEngine) GetAnomalies(start, end time.Time) []Anomaly {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	var anomalies []Anomaly
	for _, serviceAnomalies := range ae.anomalyHistory {
		for _, anomaly := range serviceAnomalies {
			if anomaly.Timestamp.After(start) && anomaly.Timestamp.Before(end) {
				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies
}

// GetPredictions generates predictions for system metrics
func (ae *AnalyticsEngine) GetPredictions() map[string][]float64 {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	predictions := make(map[string][]float64)
	
	if !ae.config.EnablePrediction {
		return predictions
	}

	// CPU Usage prediction
	cpuData := make([]float64, len(ae.systemHistory))
	for i, metrics := range ae.systemHistory {
		cpuData[i] = metrics.CPUUsage
	}
	predictions["cpu"] = ae.predictMetric(cpuData)

	// Memory Usage prediction
	memData := make([]float64, len(ae.systemHistory))
	for i, metrics := range ae.systemHistory {
		memData[i] = metrics.MemoryUsage
	}
	predictions["memory"] = ae.predictMetric(memData)

	return predictions
}

// analyze performs the main analytics processing
func (ae *AnalyticsEngine) analyze() {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	// Analyze system metrics
	ae.analyzeSystemMetrics()

	// Analyze service metrics
	for serviceID := range ae.serviceHistory {
		ae.analyzeServiceMetrics(serviceID)
	}

	// Clean up old data
	ae.trimHistory()
}

// analyzeSystemMetrics analyzes system-wide metrics
func (ae *AnalyticsEngine) analyzeSystemMetrics() {
	if len(ae.systemHistory) < ae.config.MinDataPoints {
		return
	}

	// Analyze CPU usage
	cpuData := make([]float64, len(ae.systemHistory))
	for i, metrics := range ae.systemHistory {
		cpuData[i] = metrics.CPUUsage
	}
	anomalies := ae.detectAnomalies("system_cpu", cpuData)
	ae.addAnomalies("system", anomalies)

	// Analyze memory usage
	memData := make([]float64, len(ae.systemHistory))
	for i, metrics := range ae.systemHistory {
		memData[i] = metrics.MemoryUsage
	}
	anomalies = ae.detectAnomalies("system_memory", memData)
	ae.addAnomalies("system", anomalies)
}

// analyzeServiceMetrics analyzes service-specific metrics
func (ae *AnalyticsEngine) analyzeServiceMetrics(serviceID string) {
	metrics := ae.serviceHistory[serviceID]
	if len(metrics) < ae.config.MinDataPoints {
		return
	}

	// Analyze response times
	responseTimes := make([]float64, len(metrics))
	for i, m := range metrics {
		if len(m.ResponseTimes) > 0 {
			sum := 0.0
			for _, rt := range m.ResponseTimes {
				sum += rt
			}
			responseTimes[i] = sum / float64(len(m.ResponseTimes))
		}
	}
	anomalies := ae.detectAnomalies(serviceID+"_response_time", responseTimes)
	ae.addAnomalies(serviceID, anomalies)

	// Analyze error rates
	errorRates := make([]float64, len(metrics))
	for i, m := range metrics {
		if m.RequestCount > 0 {
			errorRates[i] = float64(m.ErrorCount) / float64(m.RequestCount)
		}
	}
	anomalies = ae.detectAnomalies(serviceID+"_error_rate", errorRates)
	ae.addAnomalies(serviceID, anomalies)
}

// detectAnomalies uses statistical analysis to detect anomalies
func (ae *AnalyticsEngine) detectAnomalies(metricName string, data []float64) []Anomaly {
	if len(data) < ae.config.MinDataPoints {
		return nil
	}

	// Calculate mean and standard deviation
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(data))
	stdDev := math.Sqrt(variance)

	// Detect anomalies using z-score
	var anomalies []Anomaly
	threshold := ae.config.AnomalyThreshold * stdDev
	for i, v := range data {
		diff := math.Abs(v - mean)
		if diff > threshold {
			severity := "warning"
			if diff > threshold*2 {
				severity = "critical"
			}

			anomaly := Anomaly{
				Timestamp:    time.Now().Add(-time.Duration(len(data)-i) * ae.config.UpdateInterval),
				Source:      metricName,
				Metric:      "value",
				Value:       v,
				Threshold:   mean + threshold,
				Severity:    severity,
				Description: "Value exceeds normal range",
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// predictMetric uses simple linear regression for prediction
func (ae *AnalyticsEngine) predictMetric(data []float64) []float64 {
	if len(data) < ae.config.MinDataPoints {
		return nil
	}

	// Calculate slope and intercept using linear regression
	n := float64(len(data))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, y := range data {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Generate predictions
	numPredictions := int(ae.config.PredictionWindow / ae.config.UpdateInterval)
	predictions := make([]float64, numPredictions)

	for i := range predictions {
		x := float64(len(data) + i)
		predictions[i] = slope*x + intercept
	}

	return predictions
}

// trimHistory removes old data points
func (ae *AnalyticsEngine) trimHistory() {
	cutoff := time.Now().Add(-ae.config.HistoryWindow)

	// Trim system metrics
	var newSystemHistory []SystemMetrics
	for _, metrics := range ae.systemHistory {
		if metrics.Timestamp.After(cutoff) {
			newSystemHistory = append(newSystemHistory, metrics)
		}
	}
	ae.systemHistory = newSystemHistory

	// Trim service metrics
	for serviceID, metrics := range ae.serviceHistory {
		var newMetrics []ServiceMetrics
		for _, m := range metrics {
			if m.LastSeen.After(cutoff) {
				newMetrics = append(newMetrics, m)
			}
		}
		if len(newMetrics) > 0 {
			ae.serviceHistory[serviceID] = newMetrics
		} else {
			delete(ae.serviceHistory, serviceID)
		}
	}

	// Trim anomaly history
	for source, anomalies := range ae.anomalyHistory {
		var newAnomalies []Anomaly
		for _, anomaly := range anomalies {
			if anomaly.Timestamp.After(cutoff) {
				newAnomalies = append(newAnomalies, anomaly)
			}
		}
		if len(newAnomalies) > 0 {
			ae.anomalyHistory[source] = newAnomalies
		} else {
			delete(ae.anomalyHistory, source)
		}
	}
}

// addAnomalies adds new anomalies to the history
func (ae *AnalyticsEngine) addAnomalies(source string, anomalies []Anomaly) {
	if _, exists := ae.anomalyHistory[source]; !exists {
		ae.anomalyHistory[source] = make([]Anomaly, 0)
	}
	ae.anomalyHistory[source] = append(ae.anomalyHistory[source], anomalies...)
}
