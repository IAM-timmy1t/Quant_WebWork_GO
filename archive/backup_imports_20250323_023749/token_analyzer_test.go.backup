// token_analyzer_test.go - Utility to test the token analyzer API

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// TokenAnalysisRequest represents the request structure for the token analysis API
type TokenAnalysisRequest struct {
	ModelID      string   `json:"model_id"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	Text         string   `json:"text,omitempty"`
	TargetModels []string `json:"target_models,omitempty"`
	Options      struct {
		DetailedAnalysis       bool `json:"detailed_analysis"`
		IncludeModelDetails    bool `json:"include_model_details"`
		IncludeRecommendations bool `json:"include_recommendations"`
	} `json:"options"`
}

func main() {
	// Define command-line flags
	modelID := flag.String("model", "gpt-4", "Model ID to analyze")
	input := flag.Int("input", 0, "Number of input tokens")
	output := flag.Int("output", 0, "Number of output tokens")
	text := flag.String("text", "", "Text to analyze instead of token counts")
	targetModelsStr := flag.String("targets", "", "Comma-separated list of target models")
	detailed := flag.Bool("detailed", true, "Include detailed analysis")
	modelDetails := flag.Bool("model-details", true, "Include model details")
	recommendations := flag.Bool("recommendations", true, "Include recommendations")
	endpoint := flag.String("endpoint", "http://localhost:8080/api/security/token-analysis", "API endpoint")

	flag.Parse()

	// Validate input
	if *input <= 0 && *output <= 0 && *text == "" {
		fmt.Println("Error: You must provide either token counts (input and output) or text to analyze")
		flag.Usage()
		os.Exit(1)
	}

	// Parse target models
	var targetModels []string
	if *targetModelsStr != "" {
		targetModels = []string{}
		// Simple split by comma for demo purposes
		for _, model := range []string{"gpt-3.5-turbo", "claude-3-sonnet"} {
			targetModels = append(targetModels, model)
		}
	}

	// Create request
	req := TokenAnalysisRequest{
		ModelID:      *modelID,
		InputTokens:  *input,
		OutputTokens: *output,
		Text:         *text,
		TargetModels: targetModels,
	}
	req.Options.DetailedAnalysis = *detailed
	req.Options.IncludeModelDetails = *modelDetails
	req.Options.IncludeRecommendations = *recommendations

	// Marshal request to JSON
	requestJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		fmt.Printf("Error encoding request: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Token Analysis Request ===")
	fmt.Println(string(requestJSON))
	fmt.Println()

	// Send request to API
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(*endpoint, "application/json", bytes.NewBuffer(requestJSON))
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Pretty-print response
	var responseJSON map[string]interface{}
	err = json.Unmarshal(responseBody, &responseJSON)
	if err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	formattedResponse, err := json.MarshalIndent(responseJSON, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Token Analysis Response ===")
	fmt.Println(string(formattedResponse))
	fmt.Println()

	// Extract and display specific sections of interest
	fmt.Println("=== Summary ===")
	fmt.Println("Status Code:", resp.StatusCode)
	if scores, ok := responseJSON["scores"].(map[string]interface{}); ok {
		fmt.Println("Risk Scores:")
		for category, score := range scores {
			fmt.Printf("  %s: %.2f\n", category, score)
		}
	}

	if recommendations, ok := responseJSON["recommendations"].([]interface{}); ok && len(recommendations) > 0 {
		fmt.Println("\n=== Top Recommendations ===")
		for i, rec := range recommendations {
			if recMap, ok := rec.(map[string]interface{}); ok {
				fmt.Printf("%d. %s\n", i+1, recMap["title"])
				fmt.Printf("   %s\n", recMap["description"])
				fmt.Println()
			}
			// Only show top 3 recommendations for brevity
			if i >= 2 {
				fmt.Printf("...and %d more recommendations\n", len(recommendations)-3)
				break
			}
		}
	}
}
