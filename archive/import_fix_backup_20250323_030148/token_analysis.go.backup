// token_analysis.go - Handler for token analysis API

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security/risk"
)

// TokenAnalysisRequest represents the request body for token analysis
type TokenAnalysisRequest struct {
	ModelID      string   `json:"model_id"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	Text         string   `json:"text,omitempty"`
	TargetModels []string `json:"target_models,omitempty"`
	Options      struct {
		DetailedAnalysis     bool `json:"detailed_analysis"`
		IncludeModelDetails  bool `json:"include_model_details"`
		IncludeRecommendations bool `json:"include_recommendations"`
	} `json:"options"`
}

// TokenAnalysisResponse represents the response for token analysis
type TokenAnalysisResponse struct {
	RequestID   string                 `json:"request_id"`
	Timestamp   string                 `json:"timestamp"`
	Duration    int64                  `json:"duration_ms"`
	Scores      map[string]float64     `json:"scores"`
	Findings    []risk.Finding         `json:"findings,omitempty"`
	Recommendations []risk.Recommendation `json:"recommendations,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TokenAnalysisHandler handles token analysis requests
type TokenAnalysisHandler struct {
	analyzer *risk.TokenAnalyzer
}

// NewTokenAnalysisHandler creates a new token analysis handler
func NewTokenAnalysisHandler(analyzer *risk.TokenAnalyzer) *TokenAnalysisHandler {
	return &TokenAnalysisHandler{
		analyzer: analyzer,
	}
}

// HandleTokenAnalysis is the HTTP handler for /api/security/token-analysis
func (h *TokenAnalysisHandler) HandleTokenAnalysis(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req TokenAnalysisRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ModelID == "" {
		req.ModelID = "default" // Use default model if not specified
	}
	if req.InputTokens <= 0 && req.OutputTokens <= 0 && req.Text == "" {
		http.Error(w, "At least one of input_tokens, output_tokens, or text must be provided", http.StatusBadRequest)
		return
	}

	// Prepare analysis options
	options := map[string]interface{}{
		"detailed_analysis":    req.Options.DetailedAnalysis,
		"include_model_details": req.Options.IncludeModelDetails,
		"include_recommendations": req.Options.IncludeRecommendations,
	}

	// Prepare analysis target
	var target interface{}
	if req.Text != "" {
		// If text is provided, use it as the target
		target = req.Text
	} else {
		// Otherwise, use token counts
		tokenData := map[string]interface{}{
			"model":         req.ModelID,
			"input_tokens":  float64(req.InputTokens),
			"output_tokens": float64(req.OutputTokens),
			"total_tokens":  float64(req.InputTokens + req.OutputTokens),
		}
		
		// Add target models if provided
		if len(req.TargetModels) > 0 {
			targetModelsInterface := make([]interface{}, len(req.TargetModels))
			for i, model := range req.TargetModels {
				targetModelsInterface[i] = model
			}
			tokenData["target_models"] = targetModelsInterface
		}
		
		target = tokenData
	}

	// Perform analysis
	startTime := time.Now()
	result, err := h.analyzer.Analyze(r.Context(), target, options)
	duration := time.Since(startTime)

	if err != nil {
		http.Error(w, "Analysis failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := TokenAnalysisResponse{
		RequestID:   r.Header.Get("X-Request-ID"),
		Timestamp:   time.Now().Format(time.RFC3339),
		Duration:    duration.Milliseconds(),
		Scores:      result.Scores,
		Metadata:    result.Metadata,
	}

	// Include findings and recommendations if requested
	if req.Options.DetailedAnalysis {
		// Convert slice of pointers to slice of values
		findings := make([]risk.Finding, len(result.Findings))
		for i, f := range result.Findings {
			findings[i] = *f
		}
		response.Findings = findings
	}

	if req.Options.IncludeRecommendations {
		// Convert slice of pointers to slice of values
		recommendations := make([]risk.Recommendation, len(result.Recommendations))
		for i, r := range result.Recommendations {
			recommendations[i] = *r
		}
		response.Recommendations = recommendations
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}




