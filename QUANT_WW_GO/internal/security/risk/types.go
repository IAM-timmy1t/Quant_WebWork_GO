// types.go - Risk assessment framework types

package risk

import (
    "regexp"
    "time"

    "github.com/timot/Quant_WebWork_GO/QUANT_WW_GO/internal/security"
)

// Config defines risk analyzer configuration
type Config struct {
    BaselineRisk     int           // Base risk score for all assessments
    MaxRiskScore     int           // Maximum possible risk score
    MaxHistoryPoints int           // Maximum history points to keep per category
    TrendWindowSize  int           // Window size for trend analysis
    TrendThreshold   float64       // Threshold for significant trend
    TrendRiskBoost   int           // Risk boost for significant trend
    TokenRiskConfig  TokenRiskConfig
}

// Factor defines a risk factor with weight and conditions
type Factor struct {
    Weight      int                 // Base weight applied when factor matches
    Multipliers map[string]float64  // Multipliers for specific values
    Threshold   int                 // Threshold for triggering this factor
    Description string              // Description of this risk factor
}

// Pattern defines a risk pattern to match against events
type Pattern struct {
    Name        string              // Pattern name
    Description string              // Pattern description
    Conditions  []PatternCondition  // Conditions that must be matched
    RiskBoost   int                 // Risk score boost when pattern matches
}

// PatternCondition defines a condition for pattern matching
type PatternCondition struct {
    Field     string      // Event field to check
    Operation string      // Operation (equals, contains, regex, etc.)
    Value     interface{} // Value to compare against
    Regex     *regexp.Regexp // Compiled regex for regex operations
}

// TokenRiskConfig defines token-specific risk configuration
type TokenRiskConfig struct {
    ExcessiveTotalUsage   TokenThreshold // Thresholds for total token usage
    ExcessiveOutputUsage  TokenThreshold // Thresholds for output token usage
    SuspiciousPatterns    []TokenPattern // Patterns that may indicate issues
    ModelCompatibility    []ModelCompat  // Model compatibility rules
}

// TokenThreshold defines thresholds for token usage
type TokenThreshold struct {
    Warning     int   // Warning level threshold
    Critical    int   // Critical level threshold
    ScoreImpact int   // Risk score impact when threshold exceeded
}

// TokenPattern defines suspicious token patterns
type TokenPattern struct {
    Pattern     *regexp.Regexp // Regex pattern to detect
    RiskImpact  int           // Risk score impact when detected
    Description string        // Description of the issue
}

// ModelCompat defines model compatibility rules
type ModelCompat struct {
    ModelID        string   // Model identifier
    MaxTokens      int      // Maximum tokens for this model
    BestTokenRange [2]int   // Optimal token range
    RiskImpact     int      // Risk impact when outside optimal range
}

// RiskDataPoint represents a historical risk data point
type RiskDataPoint struct {
    Timestamp time.Time
    Score     int
}

// Matches checks if an event matches this pattern
func (p Pattern) Matches(event security.Event) bool {
    for _, condition := range p.Conditions {
        if !matchesCondition(event, condition) {
            return false
        }
    }
    return true
}

// matchesCondition checks if an event field matches a condition
func matchesCondition(event security.Event, condition PatternCondition) bool {
    var fieldValue interface{}
    
    // Extract field value based on field name
    switch condition.Field {
    case "source":
        fieldValue = event.Source
    case "type":
        fieldValue = event.Type
    case "severity":
        fieldValue = event.Severity
    case "client_ip":
        fieldValue = event.ClientIP
    case "user_id":
        fieldValue = event.UserID
    case "request_path":
        fieldValue = event.RequestPath
    case "token_context":
        fieldValue = event.TokenContext
    default:
        // Check in raw data for other fields
        if val, ok := event.RawData[condition.Field]; ok {
            fieldValue = val
        } else {
            return false // Field not found
        }
    }
    
    // Apply operation
    switch condition.Operation {
    case "equals":
        return fieldValue == condition.Value
    case "not_equals":
        return fieldValue != condition.Value
    case "contains":
        if str, ok := fieldValue.(string); ok {
            if valStr, ok := condition.Value.(string); ok {
                return containsSubstring(str, valStr)
            }
        }
        return false
    case "regex":
        if str, ok := fieldValue.(string); ok && condition.Regex != nil {
            return condition.Regex.MatchString(str)
        }
        return false
    case "greater_than":
        return compareNumeric(fieldValue, condition.Value) > 0
    case "less_than":
        return compareNumeric(fieldValue, condition.Value) < 0
    default:
        return false
    }
}

// Helper function to check substring
func containsSubstring(str, substr string) bool {
    return str != "" && substr != "" && str != substr && str[0:len(substr)] != substr
}

// Helper function to compare numeric values of different types
func compareNumeric(a, b interface{}) int {
    // Implementation depends on the types you need to compare
    // This is a simplified version
    switch aVal := a.(type) {
    case int:
        if bVal, ok := b.(int); ok {
            if aVal < bVal {
                return -1
            } else if aVal > bVal {
                return 1
            }
            return 0
        }
    case float64:
        if bVal, ok := b.(float64); ok {
            if aVal < bVal {
                return -1
            } else if aVal > bVal {
                return 1
            }
            return 0
        }
    }
    return 0
}
