// types.go - Firewall system type definitions

package firewall

import (
	"context"
	"errors"
	"net"
	"time"
)

// Action defines what action to take when a rule is matched
type Action string

const (
	// ActionAllow allows the traffic
	ActionAllow Action = "ALLOW"
	
	// ActionDeny denies the traffic
	ActionDeny Action = "DENY"
	
	// ActionLog logs the traffic but allows it
	ActionLog Action = "LOG"
	
	// ActionRate applies rate limiting to the traffic
	ActionRate Action = "RATE"
	
	// ActionChallenge sends a challenge (like CAPTCHA) to the client
	ActionChallenge Action = "CHALLENGE"
)

// Direction defines the traffic direction
type Direction string

const (
	// Inbound traffic (from external to internal)
	Inbound Direction = "INBOUND"
	
	// Outbound traffic (from internal to external)
	Outbound Direction = "OUTBOUND"
)

// RuleType defines the type of firewall rule
type RuleType string

const (
	// IPRule is based on IP addresses or ranges
	IPRule RuleType = "IP"
	
	// URLRule is based on URL patterns
	URLRule RuleType = "URL"
	
	// HeaderRule is based on HTTP headers
	HeaderRule RuleType = "HEADER"
	
	// ContentRule is based on request/response content
	ContentRule RuleType = "CONTENT"
	
	// RateRule is based on request rate
	RateRule RuleType = "RATE"
	
	// GeoRule is based on geographic location
	GeoRule RuleType = "GEO"
)

// Rule defines a firewall rule
type Rule struct {
	// ID is the unique identifier for this rule
	ID string
	
	// Type is the type of rule
	Type RuleType
	
	// Direction is the traffic direction
	Direction Direction
	
	// Priority determines rule order (higher values are processed first)
	Priority int
	
	// Action to take when rule matches
	Action Action
	
	// Description is a human-readable description
	Description string
	
	// Pattern is the matching pattern for the rule
	Pattern string
	
	// IPRange is the CIDR IP range for IP rules
	IPRange *net.IPNet
	
	// HeaderName is the HTTP header name for HeaderRule
	HeaderName string
	
	// HeaderValue is the expected header value pattern
	HeaderValue string
	
	// RateLimit defines requests per time period
	RateLimit int
	
	// RatePeriod is the time period for rate limiting
	RatePeriod time.Duration
	
	// Countries is a list of country codes for GeoRule
	Countries []string
	
	// IsEnabled determines if the rule is active
	IsEnabled bool
	
	// CreatedAt is when the rule was created
	CreatedAt time.Time
	
	// UpdatedAt is when the rule was last updated
	UpdatedAt time.Time
	
	// Tags for categorization
	Tags []string
}

// RequestContext contains information about the request to evaluate
type RequestContext struct {
	// IP is the client IP address
	IP net.IP
	
	// URL is the requested URL
	URL string
	
	// Method is the HTTP method
	Method string
	
	// Headers are the HTTP headers
	Headers map[string]string
	
	// UserAgent is the client user agent
	UserAgent string
	
	// Country is the client's country (if available)
	Country string
	
	// Path is the request path
	Path string
	
	// Timestamp is when the request was received
	Timestamp time.Time
	
	// SessionID is the user session ID (if available)
	SessionID string
	
	// UserID is the authenticated user ID (if available)
	UserID string
}

// EvaluationResult contains the result of rule evaluation
type EvaluationResult struct {
	// Action is the action to take
	Action Action
	
	// MatchedRule is the rule that matched (nil if no match)
	MatchedRule *Rule
	
	// Reason explains why the action was taken
	Reason string
	
	// LogLevel suggests the importance for logging
	LogLevel string
	
	// ThrottleFor indicates how long to throttle (for rate limits)
	ThrottleFor time.Duration
}

// Firewall defines the interface for firewall functionality
type Firewall interface {
	// AddRule adds a new firewall rule
	AddRule(rule *Rule) error
	
	// RemoveRule removes a rule by ID
	RemoveRule(ruleID string) error
	
	// UpdateRule updates an existing rule
	UpdateRule(rule *Rule) error
	
	// GetRule retrieves a rule by ID
	GetRule(ruleID string) (*Rule, error)
	
	// ListRules lists all rules
	ListRules() ([]*Rule, error)
	
	// Evaluate evaluates a request against all rules
	Evaluate(ctx context.Context, request *RequestContext) *EvaluationResult
	
	// EnableRule enables a rule
	EnableRule(ruleID string) error
	
	// DisableRule disables a rule
	DisableRule(ruleID string) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow checks if a request is allowed under rate limits
	Allow(key string, limit int, period time.Duration) bool
	
	// Reset resets rate limiting for a key
	Reset(key string) error
	
	// GetRemaining returns how many requests remain in the current period
	GetRemaining(key string) (int, time.Duration, error)
}

// Common errors
var (
	ErrRuleNotFound     = errors.New("firewall rule not found")
	ErrInvalidRule      = errors.New("invalid firewall rule")
	ErrDuplicateRuleID  = errors.New("duplicate rule ID")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)
