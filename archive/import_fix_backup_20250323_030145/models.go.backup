// models.go - Risk assessment models and interfaces

package risk

import (
	"context"
	"time"
)

// Analyzer is the interface that all risk analyzers must implement
type Analyzer interface {
	// Name returns the unique name of this analyzer
	Name() string
	
	// Description returns a human-readable description of this analyzer
	Description() string
	
	// Categories returns the risk categories this analyzer evaluates
	Categories() []string
	
	// Analyze performs risk analysis on the target and returns results
	Analyze(ctx context.Context, target interface{}, options map[string]interface{}) (*AnalyzerResult, error)
	
	// Capabilities returns the capabilities of this analyzer
	Capabilities() *AnalyzerCapabilities
}

// AnalyzerCapabilities describes what an analyzer can do
type AnalyzerCapabilities struct {
	// SupportedTargetTypes lists the types of targets this analyzer can process
	SupportedTargetTypes []string
	
	// SupportedOptions lists the options this analyzer accepts
	SupportedOptions []string
	
	// SupportedCategories lists the risk categories this analyzer can evaluate
	SupportedCategories []string
	
	// RequiresEnrichedContext indicates if context enrichment is required
	RequiresEnrichedContext bool
	
	// SupportsIncrementalAnalysis indicates if incremental analysis is supported
	SupportsIncrementalAnalysis bool
	
	// ProducesRecommendations indicates if this analyzer produces recommendations
	ProducesRecommendations bool
}

// AnalyzerResult contains the results of a risk analysis
type AnalyzerResult struct {
	// AnalyzerName is the name of the analyzer that produced this result
	AnalyzerName string
	
	// Timestamp is when the analysis was performed
	Timestamp time.Time
	
	// Duration is how long the analysis took
	Duration time.Duration
	
	// Scores maps risk categories to their scores (0.0 to 1.0)
	Scores map[string]float64
	
	// Findings contains detailed findings from the analysis
	Findings []*Finding
	
	// Recommendations contains suggested actions based on findings
	Recommendations []*Recommendation
	
	// Metadata contains additional information about the analysis
	Metadata map[string]interface{}
	
	// Error contains any error that occurred during analysis
	Error error
}

// Finding represents a specific risk finding from an analyzer
type Finding struct {
	// ID is a unique identifier for this finding
	ID string
	
	// Category is the risk category this finding belongs to
	Category string
	
	// Severity indicates how severe this finding is (0.0 to 1.0)
	Severity float64
	
	// Title is a short description of the finding
	Title string
	
	// Description is a detailed explanation of the finding
	Description string
	
	// Evidence contains data supporting this finding
	Evidence map[string]interface{}
	
	// Location identifies where in the target this finding was discovered
	Location string
	
	// Timestamp indicates when this finding was discovered
	Timestamp time.Time
	
	// Tags are labels associated with this finding
	Tags []string
	
	// CVSS contains Common Vulnerability Scoring System data if applicable
	CVSS *CVSSData
}

// CVSSData contains CVSS scoring information
type CVSSData struct {
	// Version is the CVSS version (e.g., "3.1")
	Version string
	
	// BaseScore is the base CVSS score (0.0 to 10.0)
	BaseScore float64
	
	// Vector is the CVSS vector string
	Vector string
	
	// Temporal factors that may affect the score over time
	TemporalScore float64
	
	// Environmental factors specific to the organization
	EnvironmentalScore float64
}

// Recommendation suggests actions to address findings
type Recommendation struct {
	// ID is a unique identifier for this recommendation
	ID string
	
	// Title is a short description of the recommendation
	Title string
	
	// Description is a detailed explanation of the recommendation
	Description string
	
	// RelatedFindingIDs links to the findings this addresses
	RelatedFindingIDs []string
	
	// Priority indicates how important this recommendation is (0.0 to 1.0)
	Priority float64
	
	// Effort estimates the effort required to implement (0.0 to 1.0)
	Effort float64
	
	// References contains links to additional information
	References []string
	
	// ActionItems lists specific actions to take
	ActionItems []string
}

// EvaluationOptions configures a risk evaluation
type EvaluationOptions struct {
	// EvaluationID is a unique identifier for this evaluation
	EvaluationID string
	
	// Analyzers lists the analyzers to use (by name)
	Analyzers []string
	
	// AnalyzerOptions contains options for specific analyzers
	AnalyzerOptions map[string]interface{}
	
	// Tags are labels associated with this evaluation
	Tags []string
	
	// IncludeRecommendations indicates if recommendations should be included
	IncludeRecommendations bool
	
	// IncludeEvidence indicates if evidence should be included
	IncludeEvidence bool
	
	// ContextEnrichment indicates if context enrichment should be performed
	ContextEnrichment bool
	
	// BaselineEvaluationID references a baseline to compare against
	BaselineEvaluationID string
}

// EvaluationResult contains the complete results of a risk evaluation
type EvaluationResult struct {
	// EvaluationID is the unique identifier for this evaluation
	EvaluationID string
	
	// Target is the object that was evaluated
	Target interface{}
	
	// StartTime is when the evaluation started
	StartTime time.Time
	
	// EndTime is when the evaluation completed
	EndTime time.Time
	
	// Duration is how long the evaluation took
	Duration time.Duration
	
	// AnalyzerResults maps analyzer names to their results
	AnalyzerResults map[string]*AnalyzerResult
	
	// RiskScores maps risk categories to their scores (0.0 to 1.0)
	RiskScores map[string]float64
	
	// OverallRiskScore is the combined risk score (0.0 to 1.0)
	OverallRiskScore float64
	
	// HighRiskCategories lists categories that exceeded thresholds
	HighRiskCategories []string
	
	// Factors contains additional risk factors considered
	Factors map[string]interface{}
	
	// EnrichedContext contains additional context used in evaluation
	EnrichedContext map[string]interface{}
	
	// DifferentialResults contains comparison to baseline if available
	DifferentialResults *DifferentialResults
}

// DifferentialResults contains comparison to a baseline evaluation
type DifferentialResults struct {
	// BaselineEvaluationID references the baseline evaluation
	BaselineEvaluationID string
	
	// ScoreDifferences shows how scores changed from baseline
	ScoreDifferences map[string]float64
	
	// NewFindings lists findings not present in the baseline
	NewFindings []*Finding
	
	// ResolvedFindings lists findings from baseline not in current
	ResolvedFindings []*Finding
	
	// ChangedFindings lists findings present in both but changed
	ChangedFindings []*FindingChange
	
	// OverallTrend indicates the overall risk trend (-1.0 to 1.0)
	OverallTrend float64
}

// FindingChange represents how a finding changed between evaluations
type FindingChange struct {
	// FindingID identifies the finding that changed
	FindingID string
	
	// PreviousSeverity is the severity in the baseline
	PreviousSeverity float64
	
	// CurrentSeverity is the severity in the current evaluation
	CurrentSeverity float64
	
	// SeverityChange is the difference in severity
	SeverityChange float64
	
	// Changes lists specific changes to the finding
	Changes map[string]interface{}
}

// ContextProvider enriches evaluation targets with additional context
type ContextProvider interface {
	// EnrichContext adds additional context to a target
	EnrichContext(ctx context.Context, target interface{}) (interface{}, map[string]interface{}, error)
}

// RiskAggregator combines risk scores across multiple sources
type RiskAggregator interface {
	// Aggregate combines multiple risk scores into a final score
	Aggregate(scores map[string]float64, weights map[string]float64) float64
}

// WeightedAverageAggregator is a simple implementation of RiskAggregator
type WeightedAverageAggregator struct{}

// Aggregate implements RiskAggregator.Aggregate
func (wa *WeightedAverageAggregator) Aggregate(scores map[string]float64, weights map[string]float64) float64 {
	var totalScore, totalWeight float64
	
	for category, score := range scores {
		weight := 1.0 // Default weight
		if w, exists := weights[category]; exists {
			weight = w
		}
		
		totalScore += score * weight
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return 0
	}
	
	return totalScore / totalWeight
}

// MaximumAggregator is an implementation that uses the maximum score
type MaximumAggregator struct{}

// Aggregate implements RiskAggregator.Aggregate
func (ma *MaximumAggregator) Aggregate(scores map[string]float64, weights map[string]float64) float64 {
	var maxScore float64
	
	for _, score := range scores {
		if score > maxScore {
			maxScore = score
		}
	}
	
	return maxScore
}
