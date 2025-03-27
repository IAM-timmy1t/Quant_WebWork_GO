// firewall_impl.go - Implementation of Firewall system

package firewall

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Firewall implements the Firewall interface
type Firewall struct {
	rules       map[string]*Rule
	rulesByType map[RuleType][]*Rule
	rateLimiter RateLimiter
	mutex       sync.RWMutex
	ipCache     map[string]net.IP
	ipCacheMu   sync.RWMutex
	logger      *zap.SugaredLogger
}

// NewFirewall creates a new firewall instance
func NewFirewall(limiter RateLimiter, logger *zap.SugaredLogger) *Firewall {
	if limiter == nil {
		limiter = NewRateLimiter(RateLimiterConfig{
			DefaultLimit:    100,
			DefaultInterval: time.Minute,
			CleanupInterval: 10 * time.Minute,
		})
	}

	if logger == nil {
		zapLogger, _ := zap.NewProduction()
		logger = zapLogger.Sugar()
	}

	return &Firewall{
		rules:       make(map[string]*Rule),
		rulesByType: make(map[RuleType][]*Rule),
		rateLimiter: limiter,
		ipCache:     make(map[string]net.IP),
		logger:      logger,
	}
}

// AddRule adds a new firewall rule
func (f *Firewall) AddRule(rule *Rule) error {
	if rule == nil {
		return ErrInvalidRule
	}

	// Validate required fields
	if rule.Type == "" || rule.Action == "" {
		return ErrInvalidRule
	}

	// Generate ID if not provided
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Check for duplicate ID
	if _, exists := f.rules[rule.ID]; exists {
		return ErrDuplicateRuleID
	}

	// Set timestamps
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	// Validate and prepare the rule based on type
	if err := f.prepareRule(rule); err != nil {
		return err
	}

	// Store the rule
	f.rules[rule.ID] = rule

	// Index by type
	f.rulesByType[rule.Type] = append(f.rulesByType[rule.Type], rule)

	// Sort by priority for each type
	f.sortRulesByPriority(rule.Type)

	f.logger.Infow("Rule added", "rule_id", rule.ID, "type", rule.Type, "action", rule.Action)
	return nil
}

// prepareRule validates and prepares a rule based on its type
func (f *Firewall) prepareRule(rule *Rule) error {
	// Enable by default if not specified
	if !rule.IsEnabled {
		rule.IsEnabled = true
	}

	// Set default direction if not specified
	if rule.Direction == "" {
		rule.Direction = Inbound
	}

	// Set default priority if not specified
	if rule.Priority == 0 {
		rule.Priority = 100 // Default middle priority
	}

	// Validate based on rule type
	switch rule.Type {
	case IPRule:
		if rule.IPRange == nil && rule.Pattern != "" {
			// Try to parse as CIDR
			_, ipNet, err := net.ParseCIDR(rule.Pattern)
			if err != nil {
				// Try as single IP
				ip := net.ParseIP(rule.Pattern)
				if ip == nil {
					return ErrInvalidRule
				}

				// Convert to /32 CIDR for IPv4 or /128 for IPv6
				var mask net.IPMask
				if ip.To4() != nil {
					mask = net.CIDRMask(32, 32)
				} else {
					mask = net.CIDRMask(128, 128)
				}

				ipNet = &net.IPNet{
					IP:   ip,
					Mask: mask,
				}
			}
			rule.IPRange = ipNet
		}

		if rule.IPRange == nil {
			return ErrInvalidRule
		}

	case URLRule, HeaderRule, ContentRule:
		// Require pattern for these rule types
		if rule.Pattern == "" {
			return ErrInvalidRule
		}

	case RateRule:
		// Require rate limit parameters
		if rule.RateLimit <= 0 || rule.RatePeriod <= 0 {
			return ErrInvalidRule
		}

	case GeoRule:
		// Require country codes
		if len(rule.Countries) == 0 {
			return ErrInvalidRule
		}
	}

	return nil
}

// sortRulesByPriority sorts rules of a given type by priority
func (f *Firewall) sortRulesByPriority(ruleType RuleType) {
	rules := f.rulesByType[ruleType]
	if len(rules) <= 1 {
		return
	}

	// Sort by priority (higher values processed first)
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// Evaluate evaluates a request against all rules
func (f *Firewall) Evaluate(ctx context.Context, request *RequestContext) *EvaluationResult {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// Default action is to allow if no rules match
	result := &EvaluationResult{
		Action:    ActionAllow,
		Reason:    "No matching rules found",
		LogLevel:  "debug",
	}

	// First check IP rules as they're typically the most efficient
	if ipResult := f.evaluateRulesByType(IPRule, request); ipResult != nil {
		return ipResult
	}

	// Then check geo rules
	if geoResult := f.evaluateRulesByType(GeoRule, request); geoResult != nil {
		return geoResult
	}

	// Then URL rules
	if urlResult := f.evaluateRulesByType(URLRule, request); urlResult != nil {
		return urlResult
	}

	// Then header rules
	if headerResult := f.evaluateRulesByType(HeaderRule, request); headerResult != nil {
		return headerResult
	}

	// Then content rules
	if contentResult := f.evaluateRulesByType(ContentRule, request); contentResult != nil {
		return contentResult
	}

	// Finally rate rules
	if rateResult := f.evaluateRulesByType(RateRule, request); rateResult != nil {
		return rateResult
	}

	return result
}

// evaluateRulesByType checks rules of a specific type
func (f *Firewall) evaluateRulesByType(ruleType RuleType, request *RequestContext) *EvaluationResult {
	rules, exists := f.rulesByType[ruleType]
	if !exists || len(rules) == 0 {
		return nil
	}

	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}

		matched := false

		// Check rule based on type
		switch rule.Type {
		case IPRule:
			if rule.IPRange != nil && rule.IPRange.Contains(request.IP) {
				matched = true
			}

		case URLRule:
			// Simple contains check for now - in a real implementation you'd use regex
			if rule.Pattern != "" && (request.URL == rule.Pattern || 
				(len(request.URL) >= len(rule.Pattern) && request.URL[:len(rule.Pattern)] == rule.Pattern)) {
				matched = true
			}

		case HeaderRule:
			if rule.HeaderName != "" {
				if value, exists := request.Headers[rule.HeaderName]; exists {
					if rule.HeaderValue == "" || rule.HeaderValue == value {
						matched = true
					}
				}
			}

		case ContentRule:
			// Implementation depends on how content is examined
			// Simple check for demonstration purposes
			if request.UserAgent != "" && rule.Pattern != "" && 
				(request.UserAgent == rule.Pattern || 
					(len(request.UserAgent) >= len(rule.Pattern) && 
						request.UserAgent[:len(rule.Pattern)] == rule.Pattern)) {
				matched = true
			}

		case RateRule:
			// Check rate limit
			key := request.IP.String()
			if request.UserID != "" {
				key = "user:" + request.UserID
			} else if request.SessionID != "" {
				key = "session:" + request.SessionID
			}

			if !f.rateLimiter.Allow(key, rule.RateLimit, rule.RatePeriod) {
				matched = true
			}

		case GeoRule:
			if request.Country != "" {
				for _, country := range rule.Countries {
					if country == request.Country {
						matched = true
						break
					}
				}
			}
		}

		if matched {
			return &EvaluationResult{
				Action:      rule.Action,
				MatchedRule: rule,
				Reason:      rule.Description,
				LogLevel:    "info",
				ThrottleFor: rule.Type == RateRule && rule.Action == ActionRate ? rule.RatePeriod : 0,
			}
		}
	}

	return nil
}

// GetRule retrieves a rule by ID
func (f *Firewall) GetRule(ruleID string) (*Rule, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	rule, exists := f.rules[ruleID]
	if !exists {
		return nil, ErrRuleNotFound
	}

	return rule, nil
}

// ListRules lists all rules
func (f *Firewall) ListRules() ([]*Rule, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	rules := make([]*Rule, 0, len(f.rules))
	for _, rule := range f.rules {
		rules = append(rules, rule)
	}

	return rules, nil
}

// RemoveRule removes a rule by ID
func (f *Firewall) RemoveRule(ruleID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	rule, exists := f.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}

	// Remove from main rules map
	delete(f.rules, ruleID)

	// Remove from type-based index
	if rules, exists := f.rulesByType[rule.Type]; exists {
		for i, r := range rules {
			if r.ID == ruleID {
				// Remove from slice
				f.rulesByType[rule.Type] = append(rules[:i], rules[i+1:]...)
				break
			}
		}
	}

	f.logger.Infow("Rule removed", "rule_id", ruleID)
	return nil
}

// EnableRule enables a rule
func (f *Firewall) EnableRule(ruleID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	rule, exists := f.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}

	rule.IsEnabled = true
	rule.UpdatedAt = time.Now()

	f.logger.Infow("Rule enabled", "rule_id", ruleID)
	return nil
}

// DisableRule disables a rule
func (f *Firewall) DisableRule(ruleID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	rule, exists := f.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}

	rule.IsEnabled = false
	rule.UpdatedAt = time.Now()

	f.logger.Infow("Rule disabled", "rule_id", ruleID)
	return nil
}

// UpdateRule updates an existing rule
func (f *Firewall) UpdateRule(rule *Rule) error {
	if rule == nil || rule.ID == "" {
		return ErrInvalidRule
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	existingRule, exists := f.rules[rule.ID]
	if !exists {
		return ErrRuleNotFound
	}

	// Preserve creation time
	rule.CreatedAt = existingRule.CreatedAt
	rule.UpdatedAt = time.Now()

	// Validate and prepare the updated rule
	if err := f.prepareRule(rule); err != nil {
		return err
	}

	// If type changed, update the indexes
	if existingRule.Type != rule.Type {
		// Remove from old type index
		oldTypeRules := f.rulesByType[existingRule.Type]
		for i, r := range oldTypeRules {
			if r.ID == rule.ID {
				f.rulesByType[existingRule.Type] = append(oldTypeRules[:i], oldTypeRules[i+1:]...)
				break
			}
		}

		// Add to new type index
		f.rulesByType[rule.Type] = append(f.rulesByType[rule.Type], rule)
		f.sortRulesByPriority(rule.Type)
	} else if existingRule.Priority != rule.Priority {
		// Priority changed, resort
		f.rulesByType[rule.Type] = append(f.rulesByType[rule.Type], rule)
		f.sortRulesByPriority(rule.Type)
	}

	// Update the rule
	f.rules[rule.ID] = rule

	f.logger.Infow("Rule updated", "rule_id", rule.ID)
	return nil
}
