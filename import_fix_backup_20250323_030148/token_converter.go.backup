// token_converter.go - Cross-model token conversion and optimization

package risk

import (
	"fmt"
	"sync"
)

// TokenConverter provides functionality for managing tokens across different models
// with varying context window sizes and conversion capabilities
type TokenConverter struct {
	mu              sync.RWMutex
	conversionRates map[string]map[string]float64 // Source model to target model rates
	modelProfiles   map[string]ModelProfile
}

// NewTokenConverter creates a new token converter
func NewTokenConverter() *TokenConverter {
	converter := &TokenConverter{
		conversionRates: make(map[string]map[string]float64),
		modelProfiles:   make(map[string]ModelProfile),
	}
	
	// Register default conversion rates
	converter.registerDefaultConversionRates()
	
	return converter
}

// ModelTokenCapacity represents a model's token capacity configuration
type ModelTokenCapacity struct {
	// ModelID is the unique identifier for the model
	ModelID string
	
	// MaxInputTokens is the maximum number of input tokens allowed
	MaxInputTokens int
	
	// MaxOutputTokens is the maximum number of output tokens allowed
	MaxOutputTokens int
	
	// MaxTotalTokens is the maximum total tokens (context window size)
	MaxTotalTokens int
	
	// OptimalInputRange is the recommended input token range [min, max]
	OptimalInputRange [2]int
	
	// OptimalOutputRange is the recommended output token range [min, max]
	OptimalOutputRange [2]int
	
	// TokenizationFactor is a multiplier to convert raw text to tokens
	// (e.g., 1.3 means 1 word ≈ 1.3 tokens on average)
	TokenizationFactor float64
}

// TokenAllocationStrategy defines how tokens should be allocated in a multi-model context
type TokenAllocationStrategy struct {
	// ModelID is the target model
	ModelID string
	
	// MaxBudget is the maximum token budget to allocate
	MaxBudget int
	
	// InputOutputRatio is the recommended ratio between input and output tokens
	// (e.g., 0.75 means 75% for input, 25% for output)
	InputOutputRatio float64
	
	// ReservedSystemTokens is tokens reserved for system messages/overhead
	ReservedSystemTokens int
	
	// ChunkingThreshold is when to start chunking content
	ChunkingThreshold int
	
	// ChunkSize is the recommended chunk size when chunking
	ChunkSize int
	
	// ChunkOverlap is the recommended overlap between chunks
	ChunkOverlap int
}

// TokenConversionRequest represents a request to convert tokens between models
type TokenConversionRequest struct {
	// SourceModelID is the source model
	SourceModelID string
	
	// TargetModelID is the target model
	TargetModelID string
	
	// InputTokens is the number of input tokens in the source model
	InputTokens int
	
	// OutputTokens is the number of output tokens in the source model
	OutputTokens int
	
	// TextContent is optional raw text to analyze directly
	TextContent string
	
	// PreserveRatio indicates whether to maintain the input/output ratio
	PreserveRatio bool
	
	// AllocationStrategy is an optional custom allocation strategy
	AllocationStrategy *TokenAllocationStrategy
}

// TokenConversionResult contains the results of a token conversion
type TokenConversionResult struct {
	// SourceModelID is the source model
	SourceModelID string
	
	// TargetModelID is the target model
	TargetModelID string
	
	// OriginalInputTokens is the original input token count
	OriginalInputTokens int
	
	// OriginalOutputTokens is the original output token count
	OriginalOutputTokens int
	
	// ConvertedInputTokens is the converted input token count
	ConvertedInputTokens int
	
	// ConvertedOutputTokens is the converted output token count
	ConvertedOutputTokens int
	
	// ConversionRate is the overall conversion rate applied
	ConversionRate float64
	
	// RequiresChunking indicates if chunking is recommended
	RequiresChunking bool
	
	// RecommendedChunks is the recommended number of chunks
	RecommendedChunks int
	
	// TokenAllocation is the recommended token allocation
	TokenAllocation *TokenAllocationStrategy
	
	// Warnings contains any warnings about the conversion
	Warnings []string
}

// registerDefaultConversionRates sets up known model conversion rates
func (tc *TokenConverter) registerDefaultConversionRates() {
	// Define some common models and their conversion rates
	models := []string{
		"gpt-3.5-turbo",
		"gpt-4",
		"claude-3-sonnet",
		"claude-3-opus",
		"llama-2-70b",
		"gemini-pro",
	}
	
	// Initialize conversion maps
	for _, srcModel := range models {
		tc.conversionRates[srcModel] = make(map[string]float64)
		for _, tgtModel := range models {
			if srcModel == tgtModel {
				tc.conversionRates[srcModel][tgtModel] = 1.0 // Same model = no conversion
			} else {
				// Default to 1.0 but will be overridden below
				tc.conversionRates[srcModel][tgtModel] = 1.0
			}
		}
	}
	
	// Define specific conversion rates
	// These are approximations based on different tokenization algorithms
	
	// GPT-3.5 to others
	tc.conversionRates["gpt-3.5-turbo"]["gpt-4"] = 0.95      // GPT-4 tokenizer is slightly more efficient
	tc.conversionRates["gpt-3.5-turbo"]["claude-3-sonnet"] = 1.15 // Claude tokenizer is different
	tc.conversionRates["gpt-3.5-turbo"]["claude-3-opus"] = 1.15   // Claude tokenizer is different
	tc.conversionRates["gpt-3.5-turbo"]["llama-2-70b"] = 1.05  // Slightly different tokenization
	tc.conversionRates["gpt-3.5-turbo"]["gemini-pro"] = 1.12  // Different tokenization
	
	// GPT-4 to others
	tc.conversionRates["gpt-4"]["gpt-3.5-turbo"] = 1.05     // Reverse of above
	tc.conversionRates["gpt-4"]["claude-3-sonnet"] = 1.2     // Claude tokenizer is different
	tc.conversionRates["gpt-4"]["claude-3-opus"] = 1.2       // Claude tokenizer is different
	tc.conversionRates["gpt-4"]["llama-2-70b"] = 1.1        // Different tokenization
	tc.conversionRates["gpt-4"]["gemini-pro"] = 1.18        // Different tokenization
	
	// Claude to others
	tc.conversionRates["claude-3-sonnet"]["gpt-3.5-turbo"] = 0.87 // Reverse of above
	tc.conversionRates["claude-3-sonnet"]["gpt-4"] = 0.83         // Reverse of above
	tc.conversionRates["claude-3-sonnet"]["claude-3-opus"] = 1.0  // Same tokenizer
	tc.conversionRates["claude-3-sonnet"]["llama-2-70b"] = 0.9   // Different tokenization
	tc.conversionRates["claude-3-sonnet"]["gemini-pro"] = 0.95   // Different tokenization
	
	// Repeat for claude-3-opus with same values as sonnet (same tokenizer)
	tc.conversionRates["claude-3-opus"]["gpt-3.5-turbo"] = 0.87
	tc.conversionRates["claude-3-opus"]["gpt-4"] = 0.83
	tc.conversionRates["claude-3-opus"]["claude-3-sonnet"] = 1.0
	tc.conversionRates["claude-3-opus"]["llama-2-70b"] = 0.9
	tc.conversionRates["claude-3-opus"]["gemini-pro"] = 0.95
	
	// LLaMA 2 to others
	tc.conversionRates["llama-2-70b"]["gpt-3.5-turbo"] = 0.95
	tc.conversionRates["llama-2-70b"]["gpt-4"] = 0.91
	tc.conversionRates["llama-2-70b"]["claude-3-sonnet"] = 1.11
	tc.conversionRates["llama-2-70b"]["claude-3-opus"] = 1.11
	tc.conversionRates["llama-2-70b"]["gemini-pro"] = 1.06
	
	// Gemini to others
	tc.conversionRates["gemini-pro"]["gpt-3.5-turbo"] = 0.89
	tc.conversionRates["gemini-pro"]["gpt-4"] = 0.85
	tc.conversionRates["gemini-pro"]["claude-3-sonnet"] = 1.05
	tc.conversionRates["gemini-pro"]["claude-3-opus"] = 1.05
	tc.conversionRates["gemini-pro"]["llama-2-70b"] = 0.94
}

// RegisterModelCapacity registers a model's token capacity information
func (tc *TokenConverter) RegisterModelCapacity(capacity ModelTokenCapacity) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Create a model profile from capacity information
	profile := ModelProfile{
		ID:                capacity.ModelID,
		MaxContextTokens:  capacity.MaxTotalTokens,
		OptimalTokenRange: [2]int{capacity.OptimalInputRange[0] + capacity.OptimalOutputRange[0], 
			capacity.OptimalInputRange[1] + capacity.OptimalOutputRange[1]},
		TokenCostWeighting: 1.0, // Default
		HasCompression:    false, // Default
	}
	
	tc.modelProfiles[capacity.ModelID] = profile
}

// SetConversionRate sets a specific conversion rate between two models
func (tc *TokenConverter) SetConversionRate(sourceModel, targetModel string, rate float64) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Initialize maps if they don't exist
	if _, exists := tc.conversionRates[sourceModel]; !exists {
		tc.conversionRates[sourceModel] = make(map[string]float64)
	}
	
	tc.conversionRates[sourceModel][targetModel] = rate
}

// GetConversionRate gets the conversion rate between two models
func (tc *TokenConverter) GetConversionRate(sourceModel, targetModel string) float64 {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	// Default rate if models not found
	if rateMap, exists := tc.conversionRates[sourceModel]; exists {
		if rate, exists := rateMap[targetModel]; exists {
			return rate
		}
	}
	
	// Default to 1.0 (no conversion) if not specifically defined
	return 1.0
}

// ConvertTokens converts token counts between models
func (tc *TokenConverter) ConvertTokens(request TokenConversionRequest) (*TokenConversionResult, error) {
	// Initialize result
	result := &TokenConversionResult{
		SourceModelID:       request.SourceModelID,
		TargetModelID:       request.TargetModelID,
		OriginalInputTokens: request.InputTokens,
		OriginalOutputTokens: request.OutputTokens,
		Warnings:           make([]string, 0),
	}
	
	// Get conversion rate
	conversionRate := tc.GetConversionRate(request.SourceModelID, request.TargetModelID)
	result.ConversionRate = conversionRate
	
	// Convert tokens
	result.ConvertedInputTokens = int(float64(request.InputTokens) * conversionRate)
	result.ConvertedOutputTokens = int(float64(request.OutputTokens) * conversionRate)
	
	// Get model capacities
	tc.mu.RLock()
	targetProfile, hasTargetProfile := tc.modelProfiles[request.TargetModelID]
	tc.mu.RUnlock()
	
	if !hasTargetProfile {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Target model %s not recognized, using default capacity estimates", 
				request.TargetModelID))
		
		// Use default profile
		targetProfile = ModelProfile{
			ID:               request.TargetModelID,
			MaxContextTokens: 8000, // Conservative default
			OptimalTokenRange: [2]int{2000, 6000},
		}
	}
	
	// Check if chunking is needed
	totalTokens := result.ConvertedInputTokens + result.ConvertedOutputTokens
	
	if totalTokens > targetProfile.MaxContextTokens {
		result.RequiresChunking = true
		result.RecommendedChunks = (totalTokens / targetProfile.MaxContextTokens) + 1
		
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Total converted tokens (%d) exceeds target model capacity (%d), "+
				"recommend splitting into %d chunks", 
				totalTokens, targetProfile.MaxContextTokens, result.RecommendedChunks))
	} else if totalTokens > targetProfile.OptimalTokenRange[1] {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Total converted tokens (%d) exceeds optimal range (%d), "+
				"may impact performance", 
				totalTokens, targetProfile.OptimalTokenRange[1]))
	}
	
	// Generate token allocation strategy if it doesn't exist
	if request.AllocationStrategy == nil {
		// Default allocation strategy
		inputRatio := 0.8 // 80% input, 20% output by default
		
		if request.PreserveRatio && (request.InputTokens + request.OutputTokens > 0) {
			// Preserve original ratio
			inputRatio = float64(request.InputTokens) / float64(request.InputTokens + request.OutputTokens)
		}
		
		// Determine chunking settings
		var chunkingThreshold, chunkSize, chunkOverlap int
		
		// Use 80% of max context as chunking threshold
		chunkingThreshold = int(float64(targetProfile.MaxContextTokens) * 0.8)
		
		// Use optimal max as chunk size
		chunkSize = targetProfile.OptimalTokenRange[1]
		
		// Use 10% overlap
		chunkOverlap = int(float64(chunkSize) * 0.1)
		
		// Create allocation strategy
		result.TokenAllocation = &TokenAllocationStrategy{
			ModelID:             request.TargetModelID,
			MaxBudget:           targetProfile.MaxContextTokens,
			InputOutputRatio:    inputRatio,
			ReservedSystemTokens: 100, // Default reservation
			ChunkingThreshold:   chunkingThreshold,
			ChunkSize:           chunkSize,
			ChunkOverlap:        chunkOverlap,
		}
	} else {
		// Use provided strategy but ensure it's for the right model
		strategy := *request.AllocationStrategy
		strategy.ModelID = request.TargetModelID
		result.TokenAllocation = &strategy
	}
	
	return result, nil
}

// CalculateTextTokens estimates the number of tokens in a text string for a specific model
func (tc *TokenConverter) CalculateTextTokens(text string, modelID string) int {
	// This is a very rough estimation - in a real implementation, 
	// you would use model-specific tokenizers
	
	// For GPT models, a rough approximation is 4 characters per token
	// This varies significantly by language and content type
	
	tc.mu.RLock()
	profile, hasProfile := tc.modelProfiles[modelID]
	tc.mu.RUnlock()
	
	var tokenizationFactor float64 = 0.25 // Default: 4 chars per token
	
	// Adjust based on model if we have a profile
	if hasProfile {
		switch profile.ID {
		case "gpt-3.5-turbo", "gpt-4":
			tokenizationFactor = 0.25 // 4 chars per token
		case "claude-3-sonnet", "claude-3-opus":
			tokenizationFactor = 0.22 // ~4.5 chars per token
		case "llama-2-70b":
			tokenizationFactor = 0.23 // ~4.3 chars per token
		case "gemini-pro":
			tokenizationFactor = 0.24 // ~4.2 chars per token
		}
	}
	
	// Calculate rough token count
	return int(float64(len(text)) * tokenizationFactor)
}

// OptimizeTokenAllocation generates an optimized token allocation plan
// for a given token budget across multiple models
func (tc *TokenConverter) OptimizeTokenAllocation(
	tokenBudget int,
	modelIDs []string,
	priorityWeights map[string]float64,
) (map[string]TokenAllocationStrategy, error) {
	
	if len(modelIDs) == 0 {
		return nil, fmt.Errorf("no models specified for token allocation")
	}
	
	// Initialize result
	result := make(map[string]TokenAllocationStrategy)
	
	// If only one model, allocate everything to it
	if len(modelIDs) == 1 {
		modelID := modelIDs[0]
		
		tc.mu.RLock()
		profile, hasProfile := tc.modelProfiles[modelID]
		tc.mu.RUnlock()
		
		var maxContextTokens int
		if hasProfile {
			maxContextTokens = profile.MaxContextTokens
		} else {
			maxContextTokens = 8000 // Default
		}
		
		// Cap by model's maximum context
		allocatedBudget := tokenBudget
		if allocatedBudget > maxContextTokens {
			allocatedBudget = maxContextTokens
		}
		
		result[modelID] = TokenAllocationStrategy{
			ModelID:             modelID,
			MaxBudget:           allocatedBudget,
			InputOutputRatio:    0.8, // Default
			ReservedSystemTokens: 100,
			ChunkingThreshold:   int(float64(allocatedBudget) * 0.8),
			ChunkSize:           int(float64(allocatedBudget) * 0.7),
			ChunkOverlap:        int(float64(allocatedBudget) * 0.07),
		}
		
		return result, nil
	}
	
	// For multiple models, allocate based on weights
	totalWeight := 0.0
	effectiveWeights := make(map[string]float64)
	
	// Use provided weights or default to equal weights
	for _, modelID := range modelIDs {
		weight, exists := priorityWeights[modelID]
		if !exists {
			weight = 1.0 // Default weight
		}
		effectiveWeights[modelID] = weight
		totalWeight += weight
	}
	
	// Normalize weights
	for modelID, weight := range effectiveWeights {
		effectiveWeights[modelID] = weight / totalWeight
	}
	
	// Allocate tokens based on weights
	remainingBudget := tokenBudget
	
	// First pass: allocate initial budgets
	for _, modelID := range modelIDs {
		weight := effectiveWeights[modelID]
		
		tc.mu.RLock()
		profile, hasProfile := tc.modelProfiles[modelID]
		tc.mu.RUnlock()
		
		var maxContextTokens int
		if hasProfile {
			maxContextTokens = profile.MaxContextTokens
		} else {
			maxContextTokens = 8000 // Default
		}
		
		// Calculate initial allocation
		allocation := int(float64(tokenBudget) * weight)
		
		// Cap by model's maximum context
		if allocation > maxContextTokens {
			allocation = maxContextTokens
		}
		
		// Update remaining budget
		remainingBudget -= allocation
		
		// Create allocation strategy
		result[modelID] = TokenAllocationStrategy{
			ModelID:             modelID,
			MaxBudget:           allocation,
			InputOutputRatio:    0.8, // Default
			ReservedSystemTokens: 100,
			ChunkingThreshold:   int(float64(allocation) * 0.8),
			ChunkSize:           int(float64(allocation) * 0.7),
			ChunkOverlap:        int(float64(allocation) * 0.07),
		}
	}
	
	// Second pass: distribute any remaining budget proportionally
	if remainingBudget > 0 {
		// Recalculate total weight for models that can accept more tokens
		totalRemainingWeight := 0.0
		for _, modelID := range modelIDs {
			tc.mu.RLock()
			profile, hasProfile := tc.modelProfiles[modelID]
			tc.mu.RUnlock()
			
			var maxContextTokens int
			if hasProfile {
				maxContextTokens = profile.MaxContextTokens
			} else {
				maxContextTokens = 8000 // Default
			}
			
			currentAllocation := result[modelID].MaxBudget
			
			if currentAllocation < maxContextTokens {
				totalRemainingWeight += effectiveWeights[modelID]
			}
		}
		
		if totalRemainingWeight > 0 {
			// Distribute remaining budget
			for _, modelID := range modelIDs {
				tc.mu.RLock()
				profile, hasProfile := tc.modelProfiles[modelID]
				tc.mu.RUnlock()
				
				var maxContextTokens int
				if hasProfile {
					maxContextTokens = profile.MaxContextTokens
				} else {
					maxContextTokens = 8000 // Default
				}
				
				currentAllocation := result[modelID].MaxBudget
				
				if currentAllocation < maxContextTokens {
					// Calculate additional allocation
					weight := effectiveWeights[modelID] / totalRemainingWeight
					additionalAllocation := int(float64(remainingBudget) * weight)
					
					// Cap by model's maximum context
					if currentAllocation+additionalAllocation > maxContextTokens {
						additionalAllocation = maxContextTokens - currentAllocation
					}
					
					// Update allocation
					strategy := result[modelID]
					strategy.MaxBudget += additionalAllocation
					strategy.ChunkingThreshold = int(float64(strategy.MaxBudget) * 0.8)
					strategy.ChunkSize = int(float64(strategy.MaxBudget) * 0.7)
					strategy.ChunkOverlap = int(float64(strategy.MaxBudget) * 0.07)
					
					result[modelID] = strategy
					
					remainingBudget -= additionalAllocation
					
					// If no more remaining budget, exit
					if remainingBudget <= 0 {
						break
					}
				}
			}
		}
	}
	
	return result, nil
}

// ValidateTokenUsage validates token usage against model constraints
func (tc *TokenConverter) ValidateTokenUsage(
	modelID string,
	inputTokens, outputTokens int,
) (bool, []string) {
	warnings := make([]string, 0)
	
	tc.mu.RLock()
	profile, hasProfile := tc.modelProfiles[modelID]
	tc.mu.RUnlock()
	
	if !hasProfile {
		warnings = append(warnings, 
			fmt.Sprintf("Model %s not recognized, using default capacity estimates", modelID))
		
		// Use default profile
		profile = ModelProfile{
			ID:               modelID,
			MaxContextTokens: 8000, // Conservative default
			OptimalTokenRange: [2]int{2000, 6000},
		}
	}
	
	totalTokens := inputTokens + outputTokens
	
	// Check total token count
	if totalTokens > profile.MaxContextTokens {
		warnings = append(warnings,
			fmt.Sprintf("Total tokens (%d) exceeds model capacity (%d)", 
				totalTokens, profile.MaxContextTokens))
		return false, warnings
	}
	
	// Check if within optimal range
	if totalTokens < profile.OptimalTokenRange[0] {
		warnings = append(warnings,
			fmt.Sprintf("Total tokens (%d) is below optimal minimum (%d), "+
				"may be inefficient", 
				totalTokens, profile.OptimalTokenRange[0]))
	} else if totalTokens > profile.OptimalTokenRange[1] {
		warnings = append(warnings,
			fmt.Sprintf("Total tokens (%d) exceeds optimal maximum (%d), "+
				"may impact performance", 
				totalTokens, profile.OptimalTokenRange[1]))
	}
	
	return len(warnings) == 0, warnings
}
