// engine.go - Main risk assessment engine

package risk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Engine is the central risk assessment component that coordinates
// analysis across various risk assessment analyzers
type Engine struct {
	analyzers         map[string]Analyzer
	mutex             sync.RWMutex
	evaluationHistory map[string]*EvaluationResult
	config            *EngineConfig
	logger            Logger
	metrics           MetricsCollector
	riskThresholds    map[string]float64
	customFactors     map[string]interface{}
	contextProvider   ContextProvider
}

// EngineConfig defines configuration for the risk assessment engine
type EngineConfig struct {
	// Default configuration settings
	DefaultTimeout          time.Duration
	HistoryRetentionTime    time.Duration
	MaxHistoryItems         int
	DefaultRiskThreshold    float64
	EnableMetrics           bool
	DefaultAnalyzers        []string
	EnableParallelAnalysis  bool
	MaxConcurrentEvaluations int
	EnableContextEnrichment bool
	EvaluationCacheTTL      time.Duration
}

// NewEngine creates a new risk assessment engine
func NewEngine(config *EngineConfig) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}
	
	return &Engine{
		analyzers:         make(map[string]Analyzer),
		evaluationHistory: make(map[string]*EvaluationResult),
		config:            config,
		riskThresholds:    make(map[string]float64),
		customFactors:     make(map[string]interface{}),
		logger:            &defaultLogger{},
	}
}

// DefaultEngineConfig returns the default engine configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		DefaultTimeout:          30 * time.Second,
		HistoryRetentionTime:    24 * time.Hour,
		MaxHistoryItems:         1000,
		DefaultRiskThreshold:    0.7, // 70% threshold
		EnableMetrics:           true,
		EnableParallelAnalysis:  true,
		MaxConcurrentEvaluations: 50,
		EnableContextEnrichment: true,
		EvaluationCacheTTL:      5 * time.Minute,
	}
}

// SetLogger sets the logger for the engine
func (e *Engine) SetLogger(logger Logger) {
	e.logger = logger
}

// SetMetricsCollector sets the metrics collector
func (e *Engine) SetMetricsCollector(metrics MetricsCollector) {
	e.metrics = metrics
}

// SetContextProvider sets the context provider
func (e *Engine) SetContextProvider(provider ContextProvider) {
	e.contextProvider = provider
}

// RegisterAnalyzer registers a risk analyzer
func (e *Engine) RegisterAnalyzer(analyzer Analyzer) error {
	if analyzer == nil {
		return errors.New("analyzer cannot be nil")
	}
	
	name := analyzer.Name()
	if name == "" {
		return errors.New("analyzer must have a name")
	}
	
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if _, exists := e.analyzers[name]; exists {
		return fmt.Errorf("analyzer '%s' is already registered", name)
	}
	
	e.analyzers[name] = analyzer
	e.logger.Info("Analyzer registered", map[string]interface{}{
		"analyzer": name,
	})
	
	return nil
}

// SetRiskThreshold sets a risk threshold for a specific category
func (e *Engine) SetRiskThreshold(category string, threshold float64) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if threshold < 0 {
		threshold = 0
	} else if threshold > 1 {
		threshold = 1
	}
	
	e.riskThresholds[category] = threshold
}

// EvaluateRisk performs a comprehensive risk assessment
func (e *Engine) EvaluateRisk(
	ctx context.Context,
	target interface{},
	options *EvaluationOptions,
) (*EvaluationResult, error) {
	if options == nil {
		options = &EvaluationOptions{
			Analyzers: e.getDefaultAnalyzers(),
		}
	}
	
	if len(options.Analyzers) == 0 {
		options.Analyzers = e.getDefaultAnalyzers()
	}
	
	// Create evaluation ID if not provided
	if options.EvaluationID == "" {
		options.EvaluationID = uuid.New().String()
	}
	
	// Set up timeout if not already in context
	var cancel context.CancelFunc
	_, hasDeadline := ctx.Deadline()
	if !hasDeadline && e.config.DefaultTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, e.config.DefaultTimeout)
		defer cancel()
	}
	
	startTime := time.Now()
	
	// Initialize the evaluation result
	result := &EvaluationResult{
		EvaluationID:  options.EvaluationID,
		Target:        target,
		StartTime:     startTime,
		AnalyzerResults: make(map[string]*AnalyzerResult),
		Factors:       make(map[string]interface{}),
		RiskScores:    make(map[string]float64),
	}
	
	// Enrich context if enabled and provider is available
	if e.config.EnableContextEnrichment && e.contextProvider != nil {
		enrichedTarget, enrichedContext, err := e.contextProvider.EnrichContext(ctx, target)
		if err != nil {
			e.logger.Warn("Context enrichment failed", map[string]interface{}{
				"error":  err.Error(),
				"target": fmt.Sprintf("%v", target),
			})
		} else {
			target = enrichedTarget
			result.EnrichedContext = enrichedContext
			result.Target = enrichedTarget
		}
	}
	
	// Run the analyzers
	if e.config.EnableParallelAnalysis {
		err := e.runAnalyzersParallel(ctx, target, options, result)
		if err != nil {
			return result, err
		}
	} else {
		err := e.runAnalyzersSequential(ctx, target, options, result)
		if err != nil {
			return result, err
		}
	}
	
	// Calculate final risk scores
	e.calculateFinalRiskScores(result)
	
	// Add custom factors
	e.addCustomFactors(result)
	
	// Mark completion
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	
	// Store in history
	e.storeResult(result)
	
	// Record metrics if enabled
	if e.metrics != nil && e.config.EnableMetrics {
		e.recordMetrics(result)
	}
	
	return result, nil
}

// runAnalyzersParallel runs analyzers in parallel
func (e *Engine) runAnalyzersParallel(
	ctx context.Context,
	target interface{},
	options *EvaluationOptions,
	result *EvaluationResult,
) error {
	e.mutex.RLock()
	analyzers := make(map[string]Analyzer, len(options.Analyzers))
	for _, name := range options.Analyzers {
		if analyzer, exists := e.analyzers[name]; exists {
			analyzers[name] = analyzer
		}
	}
	e.mutex.RUnlock()
	
	// Set up worker pool
	var wg sync.WaitGroup
	resultCh := make(chan *analyzerWorkResult, len(analyzers))
	errorCh := make(chan error, len(analyzers))
	
	// Create semaphore for limiting concurrency
	sem := make(chan struct{}, e.config.MaxConcurrentEvaluations)
	
	// Launch analyzer goroutines
	for name, analyzer := range analyzers {
		wg.Add(1)
		go func(name string, analyzer Analyzer) {
			defer wg.Done()
			
			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()
			
			// Run the analyzer
			analyzerResult, err := analyzer.Analyze(ctx, target, options.AnalyzerOptions)
			if err != nil {
				errorCh <- fmt.Errorf("analyzer '%s' failed: %w", name, err)
				return
			}
			
			// Send result
			resultCh <- &analyzerWorkResult{
				name:   name,
				result: analyzerResult,
			}
		}(name, analyzer)
	}
	
	// Close channels when all analyzers are done
	go func() {
		wg.Wait()
		close(resultCh)
		close(errorCh)
	}()
	
	// Collect results
	var analysisError error
	for {
		select {
		case res, ok := <-resultCh:
			if !ok {
				// Channel closed, all results collected
				return analysisError
			}
			result.AnalyzerResults[res.name] = res.result
			
		case err, ok := <-errorCh:
			if !ok {
				// Channel closed
				continue
			}
			// Record error but don't fail the entire evaluation
			e.logger.Error("Analyzer error", map[string]interface{}{
				"error": err.Error(),
			})
			
			// Only keep the first error
			if analysisError == nil {
				analysisError = err
			}
		}
	}
}

// runAnalyzersSequential runs analyzers sequentially
func (e *Engine) runAnalyzersSequential(
	ctx context.Context,
	target interface{},
	options *EvaluationOptions,
	result *EvaluationResult,
) error {
	e.mutex.RLock()
	analyzerNames := make([]string, 0, len(options.Analyzers))
	for _, name := range options.Analyzers {
		if _, exists := e.analyzers[name]; exists {
			analyzerNames = append(analyzerNames, name)
		}
	}
	e.mutex.RUnlock()
	
	var firstError error
	
	for _, name := range analyzerNames {
		// Check context cancelation
		if ctx.Err() != nil {
			return ctx.Err()
		}
		
		e.mutex.RLock()
		analyzer := e.analyzers[name]
		e.mutex.RUnlock()
		
		// Run the analyzer
		analyzerResult, err := analyzer.Analyze(ctx, target, options.AnalyzerOptions)
		if err != nil {
			e.logger.Error("Analyzer error", map[string]interface{}{
				"analyzer": name,
				"error":    err.Error(),
			})
			
			if firstError == nil {
				firstError = fmt.Errorf("analyzer '%s' failed: %w", name, err)
			}
			
			// Continue with other analyzers
			continue
		}
		
		// Store the result
		result.AnalyzerResults[name] = analyzerResult
	}
	
	return firstError
}

// calculateFinalRiskScores calculates the final risk scores
func (e *Engine) calculateFinalRiskScores(result *EvaluationResult) {
	// Collect all categories
	categories := make(map[string]bool)
	for _, analyzerResult := range result.AnalyzerResults {
		for category := range analyzerResult.Scores {
			categories[category] = true
		}
	}
	
	// Calculate scores for each category
	for category := range categories {
		var totalScore float64
		var count int
		
		for _, analyzerResult := range result.AnalyzerResults {
			if score, exists := analyzerResult.Scores[category]; exists {
				totalScore += score
				count++
			}
		}
		
		// Calculate average score if we have values
		if count > 0 {
			result.RiskScores[category] = totalScore / float64(count)
		}
	}
	
	// Calculate overall risk score (average of all categories)
	var totalScore float64
	var count int
	
	for _, score := range result.RiskScores {
		totalScore += score
		count++
	}
	
	if count > 0 {
		result.OverallRiskScore = totalScore / float64(count)
	}
	
	// Determine high-risk categories
	result.HighRiskCategories = make([]string, 0)
	
	e.mutex.RLock()
	for category, score := range result.RiskScores {
		threshold := e.config.DefaultRiskThreshold
		if customThreshold, exists := e.riskThresholds[category]; exists {
			threshold = customThreshold
		}
		
		if score >= threshold {
			result.HighRiskCategories = append(result.HighRiskCategories, category)
		}
	}
	e.mutex.RUnlock()
}

// addCustomFactors adds custom risk factors to the result
func (e *Engine) addCustomFactors(result *EvaluationResult) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	// Add any global custom factors
	for key, value := range e.customFactors {
		result.Factors[key] = value
	}
}

// storeResult stores an evaluation result in history
func (e *Engine) storeResult(result *EvaluationResult) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	// Store the result
	e.evaluationHistory[result.EvaluationID] = result
	
	// Clean up old history if we exceed the maximum
	if e.config.MaxHistoryItems > 0 && len(e.evaluationHistory) > e.config.MaxHistoryItems {
		e.cleanupHistory()
	}
}

// cleanupHistory removes old evaluation results
func (e *Engine) cleanupHistory() {
	now := time.Now()
	retention := e.config.HistoryRetentionTime
	
	// Collect IDs to remove
	idsToRemove := make([]string, 0)
	
	for id, result := range e.evaluationHistory {
		if retention > 0 && now.Sub(result.EndTime) > retention {
			idsToRemove = append(idsToRemove, id)
		}
	}
	
	// Remove old results
	for _, id := range idsToRemove {
		delete(e.evaluationHistory, id)
	}
	
	// If we still have too many items, remove oldest based on end time
	if len(e.evaluationHistory) > e.config.MaxHistoryItems {
		// Convert map to slice for sorting
		results := make([]*EvaluationResult, 0, len(e.evaluationHistory))
		for _, result := range e.evaluationHistory {
			results = append(results, result)
		}
		
		// Sort by end time (oldest first)
		sort.Slice(results, func(i, j int) bool {
			return results[i].EndTime.Before(results[j].EndTime)
		})
		
		// Remove oldest results
		for i := 0; i < len(results)-(e.config.MaxHistoryItems); i++ {
			delete(e.evaluationHistory, results[i].EvaluationID)
		}
	}
}

// GetEvaluationResult retrieves a stored evaluation result
func (e *Engine) GetEvaluationResult(evaluationID string) (*EvaluationResult, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	result, exists := e.evaluationHistory[evaluationID]
	if !exists {
		return nil, fmt.Errorf("evaluation result '%s' not found", evaluationID)
	}
	
	return result, nil
}

// getDefaultAnalyzers returns the list of default analyzers
func (e *Engine) getDefaultAnalyzers() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	// If default analyzers are configured, use them
	if len(e.config.DefaultAnalyzers) > 0 {
		return append([]string{}, e.config.DefaultAnalyzers...)
	}
	
	// Otherwise, use all registered analyzers
	analyzers := make([]string, 0, len(e.analyzers))
	for name := range e.analyzers {
		analyzers = append(analyzers, name)
	}
	
	return analyzers
}

// recordMetrics records metrics for an evaluation
func (e *Engine) recordMetrics(result *EvaluationResult) {
	// Record overall risk score
	e.metrics.RecordGauge("risk.overall_score", result.OverallRiskScore, map[string]string{
		"evaluation_id": result.EvaluationID,
	})
	
	// Record category-specific risk scores
	for category, score := range result.RiskScores {
		e.metrics.RecordGauge("risk.category_score", score, map[string]string{
			"evaluation_id": result.EvaluationID,
			"category":      category,
		})
	}
	
	// Record evaluation duration
	e.metrics.RecordLatency("risk.evaluation_duration", float64(result.Duration.Milliseconds()), map[string]string{
		"evaluation_id": result.EvaluationID,
	})
	
	// Record analyzer counts
	e.metrics.RecordGauge("risk.analyzer_count", float64(len(result.AnalyzerResults)), map[string]string{
		"evaluation_id": result.EvaluationID,
	})
	
	// Record high risk categories count
	e.metrics.RecordGauge("risk.high_risk_categories", float64(len(result.HighRiskCategories)), map[string]string{
		"evaluation_id": result.EvaluationID,
	})
}

// analyzerWorkResult holds the result from an analyzer goroutine
type analyzerWorkResult struct {
	name   string
	result *AnalyzerResult
}

// MetricsCollector interface for risk engine metrics
type MetricsCollector interface {
	RecordGauge(name string, value float64, tags map[string]string)
	RecordLatency(name string, valueMs float64, tags map[string]string)
}

// Logger interface for risk engine
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// defaultLogger is a basic implementation if none is provided
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *defaultLogger) Info(msg string, fields map[string]interface{})  {}
func (l *defaultLogger) Warn(msg string, fields map[string]interface{})  {}
func (l *defaultLogger) Error(msg string, fields map[string]interface{}) {}

// Required imports for this file
import (
	"sort"
)
