// token_schema.go - GraphQL schema for token analysis

package schema

import (
	"fmt"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/QUANT_WW_GO/internal/security/risk"
)

// TokenAnalysisTypes defines GraphQL types for token analysis
type TokenAnalysisTypes struct {
	TokenAnalysisInput       *graphql.InputObject
	TokenAnalysisOptionsInput *graphql.InputObject
	TokenAnalysisResult      *graphql.Object
	Finding                  *graphql.Object
	Recommendation           *graphql.Object
	RecommendationAction     *graphql.Object
}

// NewTokenAnalysisTypes creates GraphQL types for token analysis
func NewTokenAnalysisTypes() *TokenAnalysisTypes {
	t := &TokenAnalysisTypes{}

	// Define the Finding type
	t.Finding = graphql.NewObject(graphql.ObjectConfig{
		Name: "Finding",
		Description: "A specific finding from token analysis",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.String,
				Description: "Unique identifier for the finding",
			},
			"category": &graphql.Field{
				Type:        graphql.String,
				Description: "Risk category this finding belongs to",
			},
			"severity": &graphql.Field{
				Type:        graphql.Float,
				Description: "Severity score (0.0 to 1.0)",
			},
			"title": &graphql.Field{
				Type:        graphql.String,
				Description: "Short description of the finding",
			},
			"description": &graphql.Field{
				Type:        graphql.String,
				Description: "Detailed explanation of the finding",
			},
			"location": &graphql.Field{
				Type:        graphql.String,
				Description: "Where in the target this finding was discovered",
			},
			"timestamp": &graphql.Field{
				Type:        graphql.String,
				Description: "When this finding was discovered",
			},
			"tags": &graphql.Field{
				Type:        graphql.NewList(graphql.String),
				Description: "Labels associated with this finding",
			},
			"evidence": &graphql.Field{
				Type:        graphql.NewScalar(graphql.ScalarConfig{
					Name:        "JSON",
					Description: "Arbitrary JSON data",
					Serialize:   SerializeJSON,
				}),
				Description: "Evidence supporting this finding",
			},
		},
	})

	// Define the RecommendationAction type
	t.RecommendationAction = graphql.NewObject(graphql.ObjectConfig{
		Name: "RecommendationAction",
		Description: "A specific action to implement a recommendation",
		Fields: graphql.Fields{
			"description": &graphql.Field{
				Type:        graphql.String,
				Description: "Description of the action",
			},
			"code": &graphql.Field{
				Type:        graphql.String,
				Description: "Example code for implementation",
			},
		},
	})

	// Define the Recommendation type
	t.Recommendation = graphql.NewObject(graphql.ObjectConfig{
		Name: "Recommendation",
		Description: "A recommendation based on token analysis findings",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.String,
				Description: "Unique identifier for the recommendation",
			},
			"title": &graphql.Field{
				Type:        graphql.String,
				Description: "Short description of the recommendation",
			},
			"description": &graphql.Field{
				Type:        graphql.String,
				Description: "Detailed explanation of the recommendation",
			},
			"priority": &graphql.Field{
				Type:        graphql.Float,
				Description: "Priority level (0.0 to 1.0)",
			},
			"effort": &graphql.Field{
				Type:        graphql.Float,
				Description: "Estimated effort to implement (0.0 to 1.0)",
			},
			"related_findings": &graphql.Field{
				Type:        graphql.NewList(graphql.String),
				Description: "IDs of related findings",
			},
			"action_items": &graphql.Field{
				Type:        graphql.NewList(graphql.String),
				Description: "Specific actions to take",
			},
			"references": &graphql.Field{
				Type:        graphql.NewList(graphql.String),
				Description: "Reference links for more information",
			},
		},
	})

	// Define the TokenAnalysisResult type
	t.TokenAnalysisResult = graphql.NewObject(graphql.ObjectConfig{
		Name: "TokenAnalysisResult",
		Description: "Results of token analysis",
		Fields: graphql.Fields{
			"request_id": &graphql.Field{
				Type:        graphql.String,
				Description: "Unique identifier for the request",
			},
			"timestamp": &graphql.Field{
				Type:        graphql.String,
				Description: "When the analysis was performed",
			},
			"duration_ms": &graphql.Field{
				Type:        graphql.Int,
				Description: "Duration of analysis in milliseconds",
			},
			"scores": &graphql.Field{
				Type: graphql.NewScalar(graphql.ScalarConfig{
					Name:        "JSONObject",
					Description: "Map of risk categories to scores",
					Serialize:   SerializeJSON,
				}),
				Description: "Risk scores by category",
			},
			"findings": &graphql.Field{
				Type:        graphql.NewList(t.Finding),
				Description: "Detailed findings from the analysis",
			},
			"recommendations": &graphql.Field{
				Type:        graphql.NewList(t.Recommendation),
				Description: "Suggested actions based on findings",
			},
			"metadata": &graphql.Field{
				Type: graphql.NewScalar(graphql.ScalarConfig{
					Name:        "JSONObject",
					Description: "Arbitrary JSON object",
					Serialize:   SerializeJSON,
				}),
				Description: "Additional metadata about the analysis",
			},
		},
	})

	// Define the TokenAnalysisOptionsInput type
	t.TokenAnalysisOptionsInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "TokenAnalysisOptionsInput",
		Description: "Options for token analysis",
		Fields: graphql.InputObjectConfigFieldMap{
			"detailed_analysis": &graphql.InputObjectFieldConfig{
				Type:         graphql.Boolean,
				Description:  "Whether to include detailed analysis",
				DefaultValue: false,
			},
			"include_model_details": &graphql.InputObjectFieldConfig{
				Type:         graphql.Boolean,
				Description:  "Whether to include model details",
				DefaultValue: false,
			},
			"include_recommendations": &graphql.InputObjectFieldConfig{
				Type:         graphql.Boolean,
				Description:  "Whether to include recommendations",
				DefaultValue: true,
			},
		},
	})

	// Define the TokenAnalysisInput type
	t.TokenAnalysisInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "TokenAnalysisInput",
		Description: "Input for token analysis",
		Fields: graphql.InputObjectConfigFieldMap{
			"model_id": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "Model ID to analyze",
			},
			"input_tokens": &graphql.InputObjectFieldConfig{
				Type:        graphql.Int,
				Description: "Number of input tokens",
			},
			"output_tokens": &graphql.InputObjectFieldConfig{
				Type:        graphql.Int,
				Description: "Number of output tokens",
			},
			"text": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "Text to analyze instead of token counts",
			},
			"target_models": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewList(graphql.String),
				Description: "Target models for cross-model analysis",
			},
			"options": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewInputObject(graphql.InputObjectConfig{
					Name: "TokenAnalysisOptionsInput",
					Fields: graphql.InputObjectConfigFieldMap{
						"detailed_analysis": &graphql.InputObjectFieldConfig{
							Type:         graphql.Boolean,
							DefaultValue: false,
						},
						"include_model_details": &graphql.InputObjectFieldConfig{
							Type:         graphql.Boolean,
							DefaultValue: false,
						},
						"include_recommendations": &graphql.InputObjectFieldConfig{
							Type:         graphql.Boolean,
							DefaultValue: true,
						},
					},
				}),
				Description: "Analysis options",
			},
		},
	})

	return t
}

// SerializeJSON serializes data as JSON for GraphQL scalar types
func SerializeJSON(value interface{}) interface{} {
	return value
}

// GetTokenAnalysisQueries returns GraphQL queries for token analysis
func GetTokenAnalysisQueries(types *TokenAnalysisTypes, analyzer *risk.TokenAnalyzer) graphql.Fields {
	return graphql.Fields{
		"analyzeTokens": &graphql.Field{
			Type:        types.TokenAnalysisResult,
			Description: "Analyze token usage and generate recommendations",
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(types.TokenAnalysisInput),
				},
			},
			Resolve: resolveTokenAnalysis(analyzer),
		},
		"getModelProfiles": &graphql.Field{
			Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
				Name: "ModelProfile",
				Fields: graphql.Fields{
					"id": &graphql.Field{
						Type: graphql.String,
					},
					"max_context_tokens": &graphql.Field{
						Type: graphql.Int,
					},
					"optimal_token_range": &graphql.Field{
						Type: graphql.NewList(graphql.Int),
					},
					"token_cost_weighting": &graphql.Field{
						Type: graphql.Float,
					},
					"has_compression": &graphql.Field{
						Type: graphql.Boolean,
					},
				},
			})),
			Description: "Get available model profiles",
			Resolve: resolveModelProfiles(analyzer),
		},
	}
}

// resolveTokenAnalysis returns a resolver function for token analysis
func resolveTokenAnalysis(analyzer *risk.TokenAnalyzer) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		input, ok := p.Args["input"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid input")
		}

		// Extract input values
		modelID, _ := input["model_id"].(string)
		if modelID == "" {
			modelID = "default"
		}

		inputTokens, _ := input["input_tokens"].(int)
		outputTokens, _ := input["output_tokens"].(int)
		text, _ := input["text"].(string)
		
		// Validate input
		if inputTokens <= 0 && outputTokens <= 0 && text == "" {
			return nil, fmt.Errorf("at least one of input_tokens, output_tokens, or text must be provided")
		}

		// Extract target models
		var targetModels []string
		if targetModelsInput, ok := input["target_models"].([]interface{}); ok {
			for _, model := range targetModelsInput {
				if modelStr, ok := model.(string); ok {
					targetModels = append(targetModels, modelStr)
				}
			}
		}

		// Extract options
		options := map[string]interface{}{
			"detailed_analysis":     false,
			"include_model_details": false,
			"include_recommendations": true,
		}

		if optionsInput, ok := input["options"].(map[string]interface{}); ok {
			if detailedAnalysis, ok := optionsInput["detailed_analysis"].(bool); ok {
				options["detailed_analysis"] = detailedAnalysis
			}
			if includeModelDetails, ok := optionsInput["include_model_details"].(bool); ok {
				options["include_model_details"] = includeModelDetails
			}
			if includeRecommendations, ok := optionsInput["include_recommendations"].(bool); ok {
				options["include_recommendations"] = includeRecommendations
			}
		}

		// Prepare target
		var target interface{}
		if text != "" {
			target = text
		} else {
			tokenData := map[string]interface{}{
				"model":         modelID,
				"input_tokens":  float64(inputTokens),
				"output_tokens": float64(outputTokens),
				"total_tokens":  float64(inputTokens + outputTokens),
			}
			
			if len(targetModels) > 0 {
				targetModelsInterface := make([]interface{}, len(targetModels))
				for i, model := range targetModels {
					targetModelsInterface[i] = model
				}
				tokenData["target_models"] = targetModelsInterface
			}
			
			target = tokenData
		}

		// Perform analysis
		startTime := time.Now()
		result, err := analyzer.Analyze(p.Context, target, options)
		if err != nil {
			return nil, err
		}
		duration := time.Since(startTime)

		// Convert to response format
		response := map[string]interface{}{
			"request_id":   p.Context.Value("request_id"),
			"timestamp":    time.Now().Format(time.RFC3339),
			"duration_ms":  duration.Milliseconds(),
			"scores":       result.Scores,
			"metadata":     result.Metadata,
		}

		// Include findings if requested
		if options["detailed_analysis"].(bool) {
			findings := make([]interface{}, len(result.Findings))
			for i, f := range result.Findings {
				findings[i] = *f
			}
			response["findings"] = findings
		}

		// Include recommendations if requested
		if options["include_recommendations"].(bool) {
			recommendations := make([]interface{}, len(result.Recommendations))
			for i, r := range result.Recommendations {
				recommendations[i] = *r
			}
			response["recommendations"] = recommendations
		}

		return response, nil
	}
}

// resolveModelProfiles returns a resolver function for model profiles
func resolveModelProfiles(analyzer *risk.TokenAnalyzer) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		profiles := analyzer.GetModelProfiles()
		result := make([]map[string]interface{}, 0, len(profiles))
		
		for _, profile := range profiles {
			result = append(result, map[string]interface{}{
				"id":                   profile.ID,
				"max_context_tokens":   profile.MaxContextTokens,
				"optimal_token_range":  []int{profile.OptimalTokenRange[0], profile.OptimalTokenRange[1]},
				"token_cost_weighting": profile.TokenCostWeighting,
				"has_compression":      profile.HasCompression,
			})
		}
		
		return result, nil
	}
}



