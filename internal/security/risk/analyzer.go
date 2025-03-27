// analyzer.go - Advanced risk assessment framework

package risk

import (
    "context"
    "sync"
    "time"

    "github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security"
)

// Analyzer implements advanced risk assessment capabilities
type Analyzer struct {
    config         Config
    patterns       []Pattern
    factors        map[string]Factor
    contextData    map[string]interface{}
    mu             sync.RWMutex
    historicalRisk map[string][]RiskDataPoint
}

// NewAnalyzer creates a new risk analyzer
func NewAnalyzer(config Config) *Analyzer {
    return &Analyzer{
        config:         config,
        patterns:       make([]Pattern, 0),
        factors:        make(map[string]Factor),
        contextData:    make(map[string]interface{}),
        historicalRisk: make(map[string][]RiskDataPoint),
    }
}

// CalculateRisk determines the risk score for an event or context
func (a *Analyzer) CalculateRisk(ctx context.Context, event security.Event) int {
    a.mu.RLock()
    defer a.mu.RUnlock()
    
    // Start with baseline risk
    riskScore := a.config.BaselineRisk
    
    // Apply source-based risk factors
    if factor, ok := a.factors[event.Source]; ok {
        riskScore += factor.Weight
    }

    // Apply type-based risk factors - use the String() method to convert SecurityEventType to string
    if factor, ok := a.factors[event.Type.String()]; ok {
        riskScore += factor.Weight
    }

    // Apply severity-based multipliers
    severityMultiplier := 1
    switch event.Severity {
    case security.RiskCritical:
        severityMultiplier = 5
    case security.RiskHigh:
        severityMultiplier = 4
    case security.RiskMedium:
        severityMultiplier = 3
    case security.RiskLow:
        severityMultiplier = 2
    default:
        severityMultiplier = 1
    }
    riskScore *= severityMultiplier

    // Check for pattern matches
    for _, pattern := range a.patterns {
        if pattern.Matches(event) {
            riskScore += pattern.RiskBoost
        }
    }

    // Apply historical risk trend analysis
    if historyBoost := a.calculateHistoricalRiskBoost(event.Source, event.Type.String()); historyBoost > 0 {
        riskScore += historyBoost
    }

    // Cap maximum risk if needed
    if riskScore > a.config.MaxRiskScore {
        riskScore = a.config.MaxRiskScore
    }

    // Record this risk score in the historical data
    a.recordRiskScore(event.Source, event.Type.String(), riskScore)
    
    return riskScore
}

// AddFactor adds or updates a risk factor
func (a *Analyzer) AddFactor(category string, factor Factor) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    a.factors[category] = factor
}

// AddPattern adds a risk pattern
func (a *Analyzer) AddPattern(pattern Pattern) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    a.patterns = append(a.patterns, pattern)
}

// UpdateContextData updates the risk context data
func (a *Analyzer) UpdateContextData(key string, value interface{}) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    a.contextData[key] = value
}

// recordRiskScore adds a risk score to the historical data
func (a *Analyzer) recordRiskScore(source, eventType string, score int) {
    // Create a combined key for source+type
    key := source + ":" + eventType
    
    // Add the new data point
    dataPoint := RiskDataPoint{
        Timestamp: time.Now(),
        Score:     score,
    }
    
    // Add to history, keeping up to maxHistoryPoints
    history := a.historicalRisk[key]
    history = append(history, dataPoint)
    if len(history) > a.config.MaxHistoryPoints {
        history = history[len(history)-a.config.MaxHistoryPoints:]
    }
    
    a.historicalRisk[key] = history
}

// calculateHistoricalRiskBoost analyzes risk trends and returns additional risk
func (a *Analyzer) calculateHistoricalRiskBoost(source, eventType string) int {
    key := source + ":" + eventType
    history, ok := a.historicalRisk[key]
    if !ok || len(history) < 2 {
        return 0
    }
    
    // Calculate trend over recent points
    recentWindow := a.config.TrendWindowSize
    if recentWindow > len(history) {
        recentWindow = len(history)
    }
    
    recentPoints := history[len(history)-recentWindow:]
    
    // Calculate average of recent risk scores
    var sum int
    for _, point := range recentPoints {
        sum += point.Score
    }
    recentAvg := sum / len(recentPoints)
    
    // Calculate earlier average if enough history
    var earlierAvg int
    if len(history) > recentWindow*2 {
        earlierPoints := history[len(history)-recentWindow*2:len(history)-recentWindow]
        sum = 0
        for _, point := range earlierPoints {
            sum += point.Score
        }
        earlierAvg = sum / len(earlierPoints)
        
        // If there's a significant increase in risk, apply a boost
        if recentAvg > earlierAvg && 
           float64(recentAvg-earlierAvg)/float64(earlierAvg) > a.config.TrendThreshold {
            return a.config.TrendRiskBoost
        }
    }
    
    return 0
}

// GetRiskTrend returns risk trend data for visualization
func (a *Analyzer) GetRiskTrend(sources []string, duration time.Duration) map[string][]RiskDataPoint {
    a.mu.RLock()
    defer a.mu.RUnlock()
    
    result := make(map[string][]RiskDataPoint)
    cutoff := time.Now().Add(-duration)
    
    for key, history := range a.historicalRisk {
        // Filter by sources if specified
        if len(sources) > 0 {
            found := false
            for _, source := range sources {
                if key[:len(source)] == source {
                    found = true
                    break
                }
            }
            if !found {
                continue
            }
        }
        
        // Filter by time
        var filtered []RiskDataPoint
        for _, point := range history {
            if point.Timestamp.After(cutoff) {
                filtered = append(filtered, point)
            }
        }
        
        if len(filtered) > 0 {
            result[key] = filtered
        }
    }
    
    return result
}
