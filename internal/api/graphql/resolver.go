// resolver.go - Main GraphQL resolver implementation

package graphql

import (
	"context"
	"errors"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/api/graphql/schema"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security/risk"
)

// Resolver handles GraphQL query resolution
type Resolver struct {
	schema        *graphql.Schema
	tokenAnalyzer *risk.TokenAnalyzer
	riskEngine    *risk.Engine
}

// NewResolver creates a new GraphQL resolver
func NewResolver(tokenAnalyzer *risk.TokenAnalyzer, riskEngine *risk.Engine) (*Resolver, error) {
	if tokenAnalyzer == nil {
		return nil, errors.New("token analyzer cannot be nil")
	}
	if riskEngine == nil {
		return nil, errors.New("risk engine cannot be nil")
	}

	// Create a new resolver
	r := &Resolver{
		tokenAnalyzer: tokenAnalyzer,
		riskEngine:    riskEngine,
	}

	// Initialize the schema
	schema, err := r.initSchema()
	if err != nil {
		return nil, err
	}
	r.schema = schema

	return r, nil
}

// initSchema initializes the GraphQL schema
func (r *Resolver) initSchema() (*graphql.Schema, error) {
	// Create token analysis types
	tokenTypes := schema.NewTokenAnalysisTypes()

	// Define the root query
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			// Add token analysis queries
			"tokenAnalysis": &graphql.Field{
				Type:        tokenTypes.TokenAnalysisResult,
				Description: "Analyze a token for security risks",
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(tokenTypes.TokenAnalysisInput),
					},
					"options": &graphql.ArgumentConfig{
						Type: tokenTypes.TokenAnalysisOptionsInput,
					},
				},
				Resolve: r.resolveTokenAnalysis(),
			},
			"riskProfile": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "RiskProfile",
					Fields: graphql.Fields{
						"id": &graphql.Field{
							Type: graphql.String,
						},
						"name": &graphql.Field{
							Type: graphql.String,
						},
						"score": &graphql.Field{
							Type: graphql.Float,
						},
						"lastUpdate": &graphql.Field{
							Type: graphql.String,
						},
						"categories": &graphql.Field{
							Type: graphql.NewList(graphql.String),
						},
					},
				}),
				Description: "Get a risk profile",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: r.resolveRiskProfile(),
			},
			"systemStatus": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "SystemStatus",
					Fields: graphql.Fields{
						"healthy": &graphql.Field{
							Type: graphql.Boolean,
						},
						"version": &graphql.Field{
							Type: graphql.String,
						},
						"uptime": &graphql.Field{
							Type: graphql.Int,
						},
						"services": &graphql.Field{
							Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
								Name: "ServiceStatus",
								Fields: graphql.Fields{
									"name": &graphql.Field{
										Type: graphql.String,
									},
									"status": &graphql.Field{
										Type: graphql.String,
									},
									"latency": &graphql.Field{
										Type: graphql.Int,
									},
								},
							})),
						},
					},
				}),
				Description: "Get system status information",
				Resolve:     r.resolveSystemStatus(),
			},
		},
	})

	// Define the root mutation
	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootMutation",
		Fields: graphql.Fields{
			"scheduleTokenAnalysis": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "ScheduleResponse",
					Fields: graphql.Fields{
						"jobId": &graphql.Field{
							Type: graphql.String,
						},
						"estimatedCompletionTime": &graphql.Field{
							Type: graphql.String,
						},
					},
				}),
				Description: "Schedule a token analysis job",
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(tokenTypes.TokenAnalysisInput),
					},
					"options": &graphql.ArgumentConfig{
						Type: tokenTypes.TokenAnalysisOptionsInput,
					},
				},
				Resolve: r.resolveScheduleTokenAnalysis(),
			},
			"updateRiskModel": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "UpdateRiskModelResponse",
					Fields: graphql.Fields{
						"success": &graphql.Field{
							Type: graphql.Boolean,
						},
						"message": &graphql.Field{
							Type: graphql.String,
						},
					},
				}),
				Description: "Update a risk model configuration",
				Args: graphql.FieldConfigArgument{
					"modelId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"configuration": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String), // JSON string
					},
				},
				Resolve: r.resolveUpdateRiskModel(),
			},
		},
	})

	// Create the schema
	schemaConfig := graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	}

	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return nil, err
	}

	return &schema, nil
}

// Execute executes a GraphQL query
func (r *Resolver) Execute(ctx context.Context, query string, variables map[string]interface{}) *graphql.Result {
	return graphql.Do(graphql.Params{
		Schema:         *r.schema,
		RequestString:  query,
		VariableValues: variables,
		Context:        ctx,
	})
}

// resolveTokenAnalysis returns a resolver function for token analysis
func (r *Resolver) resolveTokenAnalysis() graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		// Extract input parameters
		input, ok := p.Args["input"].(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid input format")
		}

		// Extract token content
		tokenContent, ok := input["token"].(string)
		if !ok || tokenContent == "" {
			return nil, errors.New("token is required and must be a string")
		}

		// Extract context if available
		tokenContext := ""
		if ctx, ok := input["context"].(string); ok {
			tokenContext = ctx
		}

		// Extract options if available
		detailed := false
		includeRecommendations := true
		if options, ok := p.Args["options"].(map[string]interface{}); ok {
			if d, ok := options["detailed_analysis"].(bool); ok {
				detailed = d
			}
			if ir, ok := options["include_recommendations"].(bool); ok {
				includeRecommendations = ir
			}
		}

		// Run the analysis
		startTime := time.Now()
		analysisResult, err := r.tokenAnalyzer.Analyze(tokenContent, tokenContext, detailed)
		if err != nil {
			return nil, err
		}
		duration := time.Since(startTime)

		// Build response
		result := map[string]interface{}{
			"request_id":  analysisResult.RequestID,
			"timestamp":   startTime.Format(time.RFC3339),
			"duration_ms": duration.Milliseconds(),
			"scores":      analysisResult.Scores,
			"findings":    analysisResult.Findings,
			"metadata":    analysisResult.Metadata,
		}

		// Add recommendations if requested
		if includeRecommendations {
			recommendations, err := r.tokenAnalyzer.GenerateRecommendations(analysisResult)
			if err != nil {
				// Log the error but don't fail the whole request
				result["recommendations"] = []interface{}{}
			} else {
				result["recommendations"] = recommendations
			}
		}

		return result, nil
	}
}

// resolveRiskProfile returns a resolver function for risk profiles
func (r *Resolver) resolveRiskProfile() graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		// Extract profile ID
		profileID, ok := p.Args["id"].(string)
		if !ok || profileID == "" {
			return nil, errors.New("profile ID is required")
		}

		// Get risk profile from the risk engine
		profile, err := r.riskEngine.GetProfile(profileID)
		if err != nil {
			return nil, err
		}

		// Format the response
		return map[string]interface{}{
			"id":         profile.ID,
			"name":       profile.Name,
			"score":      profile.Score,
			"lastUpdate": profile.LastUpdate.Format(time.RFC3339),
			"categories": profile.Categories,
		}, nil
	}
}

// resolveSystemStatus returns a resolver function for system status
func (r *Resolver) resolveSystemStatus() graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		// In a real implementation, gather status from various services
		return map[string]interface{}{
			"healthy": true,
			"version": "1.0.0",
			"uptime":  3600, // seconds
			"services": []map[string]interface{}{
				{
					"name":    "tokenAnalyzer",
					"status":  "healthy",
					"latency": 50, // ms
				},
				{
					"name":    "riskEngine",
					"status":  "healthy",
					"latency": 30, // ms
				},
			},
		}, nil
	}
}

// resolveScheduleTokenAnalysis returns a resolver function for scheduling token analysis
func (r *Resolver) resolveScheduleTokenAnalysis() graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		// Extract input parameters (same as tokenAnalysis)
		input, ok := p.Args["input"].(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid input format")
		}

		// Extract token content
		tokenContent, ok := input["token"].(string)
		if !ok || tokenContent == "" {
			return nil, errors.New("token is required and must be a string")
		}

		// In a real implementation, this would schedule a job in a queue
		jobID := "job_" + time.Now().Format("20060102150405")
		estimatedCompletionTime := time.Now().Add(time.Minute).Format(time.RFC3339)

		return map[string]interface{}{
			"jobId":                  jobID,
			"estimatedCompletionTime": estimatedCompletionTime,
		}, nil
	}
}

// resolveUpdateRiskModel returns a resolver function for updating risk models
func (r *Resolver) resolveUpdateRiskModel() graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		// Extract model ID
		modelID, ok := p.Args["modelId"].(string)
		if !ok || modelID == "" {
			return nil, errors.New("model ID is required")
		}

		// Extract configuration
		config, ok := p.Args["configuration"].(string)
		if !ok || config == "" {
			return nil, errors.New("configuration is required")
		}

		// In a real implementation, this would update the model configuration
		success := true
		message := "Model updated successfully"

		return map[string]interface{}{
			"success": success,
			"message": message,
		}, nil
	}
}




