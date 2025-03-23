// analyzer_result.go - Extensions for the AnalyzerResult structure

package risk

import (
	"fmt"
	"time"
)

// NewAnalyzerResult creates a new AnalyzerResult with default values
func NewAnalyzerResult(analyzerName string) *AnalyzerResult {
	return &AnalyzerResult{
		AnalyzerName:    analyzerName,
		Timestamp:       time.Now(),
		Scores:          make(map[string]float64),
		Findings:        make([]*Finding, 0),
		Recommendations: make([]*Recommendation, 0),
		Metadata:        make(map[string]interface{}),
	}
}

// AddFinding adds a new finding to the result
func (r *AnalyzerResult) AddFinding(finding *Finding) {
	if finding == nil {
		return
	}
	if finding.Timestamp.IsZero() {
		finding.Timestamp = time.Now()
	}
	r.Findings = append(r.Findings, finding)
}

// AddRecommendation adds a new recommendation to the result
func (r *AnalyzerResult) AddRecommendation(rec *Recommendation) {
	if rec == nil {
		return
	}
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("rec-%d", len(r.Recommendations)+1)
	}
	r.Recommendations = append(r.Recommendations, rec)
}

// AddScore adds or updates a risk score for a category
func (r *AnalyzerResult) AddScore(category string, score float64) {
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}
	r.Scores[category] = score
}

// AddMetadata adds or updates a metadata entry
func (r *AnalyzerResult) AddMetadata(key string, value interface{}) {
	r.Metadata[key] = value
}

// AverageScore calculates the average score across all categories
func (r *AnalyzerResult) AverageScore() float64 {
	if len(r.Scores) == 0 {
		return 0.0
	}
	
	total := 0.0
	for _, score := range r.Scores {
		total += score
	}
	
	return total / float64(len(r.Scores))
}

// MaxScore returns the highest score across all categories
func (r *AnalyzerResult) MaxScore() float64 {
	max := 0.0
	for _, score := range r.Scores {
		if score > max {
			max = score
		}
	}
	
	return max
}

// HasHighRisk returns true if any category has a score above the threshold
func (r *AnalyzerResult) HasHighRisk(threshold float64) bool {
	if threshold <= 0.0 || threshold > 1.0 {
		threshold = 0.7 // Default threshold for "high" risk
	}
	
	for _, score := range r.Scores {
		if score >= threshold {
			return true
		}
	}
	
	return false
}

// Merge combines this result with another result
func (r *AnalyzerResult) Merge(other *AnalyzerResult) {
	if other == nil {
		return
	}
	
	// Merge scores
	for category, score := range other.Scores {
		r.Scores[category] = score
	}
	
	// Merge findings
	r.Findings = append(r.Findings, other.Findings...)
	
	// Merge recommendations
	r.Recommendations = append(r.Recommendations, other.Recommendations...)
	
	// Merge metadata
	for key, value := range other.Metadata {
		r.Metadata[key] = value
	}
}

// NewRecommendationFromTemplate creates a recommendation from a template with implementation details
func NewRecommendationFromTemplate(title, description, implementation string, relatedFindings []string) *Recommendation {
	rec := &Recommendation{
		Title:            title,
		Description:      description,
		RelatedFindingIDs: relatedFindings,
		Priority:         0.5, // Default medium priority
		Effort:           0.5, // Default medium effort
		References:       []string{},
	}
	
	// Convert implementation details to action items
	if implementation != "" {
		rec.ActionItems = []string{implementation}
	}
	
	return rec
}
