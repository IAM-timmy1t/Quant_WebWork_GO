// firewall.go - Firewall implementation

package firewall

import (
	"context"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// FirewallImpl implements the Firewall interface
type FirewallImpl struct {
	rules       map[string]*Rule
	rulesByType map[RuleType][]*Rule
	rateLimiter RateLimiter
	mutex       sync.RWMutex
	ipCache     map[string]net.IP
	ipCacheMu   sync.RWMutex
	logger      Logger
}

// Logger interface for firewall logging
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// defaultLogger is a basic implementation if none is provided
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *defaultLogger) Info(msg string, fields map[string]interface{})  {}
func (l *defaultLogger) Warn(msg string, fields map[string]interface{})  {}
func (l *defaultLogger) Error(msg string, fields map[string]interface{}) {}

// NewFirewall creates a new firewall instance
func NewFirewall(rateLimiter RateLimiter, logger Logger) *FirewallImpl {
	if rateLimiter == nil {
		rateLimiter = NewMemoryRateLimiter()
	}
	
	if logger == nil {
		logger = &defaultLogger{}
	}
	
	return &FirewallImpl{
		rules:       make(map[string]*Rule),
		rulesByType: make(map[RuleType][]*Rule),
		rateLimiter: rateLimiter,
		ipCache:     make(map[string]net.IP),
		logger:      logger,
	}
}

// AddRule adds a new firewall rule
func (f *FirewallImpl) AddRule(rule *Rule) error {
	if rule == nil || rule.ID == "" || rule.Type == "" || rule.Action == "" {
		return ErrInvalidRule
	}
	
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	// Check for duplicate ID
	if _, exists := f.rules[rule.ID]; exists {
		return ErrDuplicateRuleID
	}
	
	// Set timestamps
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	
	// Validate and prepare the rule
	if err := f.prepareRule(rule); err != nil {
		return err
	}
	
	// Store the rule
	f.rules[rule.ID] = rule
	
	// Index by type
	f.rulesByType[rule.Type] = append(f.rulesByType[rule.Type], rule)
	
	// Sort by priority for each type
	f.sortRulesByPriority(rule.Type)
	
	return nil
}

// prepareRule validates and prepares a rule for usage
func (f *FirewallImpl) prepareRule(rule *Rule) error {
	// Enable by default
	if !rule.IsEnabled {
		rule.IsEnabled = true
	}
	
	// Handle specific rule types
	switch rule.Type {
	case IPRule:
		if rule.IPRange == nil && rule.Pattern != "" {
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
		// Compile regex pattern if needed
		if rule.Pattern == "" {
			return ErrInvalidRule
		}
		
		// Validate pattern as regex (we don't store the compiled pattern
		// to avoid sync.Mutex issues with regexp.Regexp)
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return ErrInvalidRule
		}
	
	case RateRule:
		if rule.RateLimit <= 0 || rule.RatePeriod <= 0 {
			return ErrInvalidRule
		}
	
	case GeoRule:
		if len(rule.Countries) == 0 {
			return ErrInvalidRule
		}
	}
	
	return nil
}

// sortRulesByPriority sorts rules of a given type by priority
func (f *FirewallImpl) sortRulesByPriority(ruleType RuleType) {
	rules := f.rulesByType[ruleType]
	
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
}

// RemoveRule removes a rule by ID
func (f *FirewallImpl) RemoveRule(ruleID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	rule, exists := f.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}
	
	// Remove from main map
	delete(f.rules, ruleID)
	
	// Remove from type index
	rules := f.rulesByType[rule.Type]
	for i, r := range rules {
		if r.ID == ruleID {
			f.rulesByType[rule.Type] = append(rules[:i], rules[i+1:]...)
			break
		}
	}
	
	return nil
}

// UpdateRule updates an existing rule
func (f *FirewallImpl) UpdateRule(rule *Rule) error {
	if rule == nil || rule.ID == "" {
		return ErrInvalidRule
	}
	
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	// Check if rule exists
	existingRule, exists := f.rules[rule.ID]
	if !exists {
		return ErrRuleNotFound
	}
	
	// Preserve creation time
	rule.CreatedAt = existingRule.CreatedAt
	rule.UpdatedAt = time.Now()
	
	// Validate and prepare the rule
	if err := f.prepareRule(rule); err != nil {
		return err
	}
	
	// If rule type changed, we need to update indexes
	if existingRule.Type != rule.Type {
		// Remove from old type
		oldRules := f.rulesByType[existingRule.Type]
		for i, r := range oldRules {
			if r.ID == rule.ID {
				f.rulesByType[existingRule.Type] = append(oldRules[:i], oldRules[i+1:]...)
				break
			}
		}
		
		// Add to new type
		f.rulesByType[rule.Type] = append(f.rulesByType[rule.Type], rule)
		
		// Sort both
		f.sortRulesByPriority(existingRule.Type)
		f.sortRulesByPriority(rule.Type)
	} else if existingRule.Priority != rule.Priority {
		// Just resort the current type
		f.rules[rule.ID] = rule
		f.sortRulesByPriority(rule.Type)
	} else {
		// Just update the rule
		f.rules[rule.ID] = rule
		
		// Update in the type slice
		for i, r := range f.rulesByType[rule.Type] {
			if r.ID == rule.ID {
				f.rulesByType[rule.Type][i] = rule
				break
			}
		}
	}
	
	return nil
}

// GetRule retrieves a rule by ID
func (f *FirewallImpl) GetRule(ruleID string) (*Rule, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	
	rule, exists := f.rules[ruleID]
	if !exists {
		return nil, ErrRuleNotFound
	}
	
	// Return a copy to prevent modification
	ruleCopy := *rule
	return &ruleCopy, nil
}

// ListRules lists all rules
func (f *FirewallImpl) ListRules() ([]*Rule, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	
	rules := make([]*Rule, 0, len(f.rules))
	for _, rule := range f.rules {
		ruleCopy := *rule
		rules = append(rules, &ruleCopy)
	}
	
	// Sort by priority
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	
	return rules, nil
}

// EnableRule enables a rule
func (f *FirewallImpl) EnableRule(ruleID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	rule, exists := f.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}
	
	rule.IsEnabled = true
	rule.UpdatedAt = time.Now()
	
	return nil
}

// DisableRule disables a rule
func (f *FirewallImpl) DisableRule(ruleID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	rule, exists := f.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}
	
	rule.IsEnabled = false
	rule.UpdatedAt = time.Now()
	
	return nil
}

// Evaluate evaluates a request against all rules
func (f *FirewallImpl) Evaluate(ctx context.Context, request *RequestContext) *EvaluationResult {
	if request == nil {
		return &EvaluationResult{
			Action:   ActionAllow,
			Reason:   "No request context provided",
			LogLevel: "ERROR",
		}
	}
	
	// Default result allows traffic
	result := &EvaluationResult{
		Action:   ActionAllow,
		Reason:   "No rules matched",
		LogLevel: "DEBUG",
	}
	
	// Check IP rules
	if ipResult := f.evaluateIPRules(request); ipResult != nil {
		return ipResult
	}
	
	// Check URL rules
	if urlResult := f.evaluateURLRules(request); urlResult != nil {
		return urlResult
	}
	
	// Check header rules
	if headerResult := f.evaluateHeaderRules(request); headerResult != nil {
		return headerResult
	}
	
	// Check rate limit rules
	if rateResult := f.evaluateRateRules(request); rateResult != nil {
		return rateResult
	}
	
	// Check geo rules
	if geoResult := f.evaluateGeoRules(request); geoResult != nil {
		return geoResult
	}
	
	// Check content rules
	if contentResult := f.evaluateContentRules(request); contentResult != nil {
		return contentResult
	}
	
	return result
}

// evaluateIPRules checks IP-based rules
func (f *FirewallImpl) evaluateIPRules(request *RequestContext) *EvaluationResult {
	f.mutex.RLock()
	rules := f.rulesByType[IPRule]
	f.mutex.RUnlock()
	
	if len(rules) == 0 || request.IP == nil {
		return nil
	}
	
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		
		if rule.IPRange.Contains(request.IP) {
			return &EvaluationResult{
				Action:      rule.Action,
				MatchedRule: rule,
				Reason:      "IP match: " + request.IP.String(),
				LogLevel:    "INFO",
			}
		}
	}
	
	return nil
}

// evaluateURLRules checks URL-based rules
func (f *FirewallImpl) evaluateURLRules(request *RequestContext) *EvaluationResult {
	f.mutex.RLock()
	rules := f.rulesByType[URLRule]
	f.mutex.RUnlock()
	
	if len(rules) == 0 || request.URL == "" {
		return nil
	}
	
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		
		// Compile the pattern
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue
		}
		
		if pattern.MatchString(request.URL) {
			return &EvaluationResult{
				Action:      rule.Action,
				MatchedRule: rule,
				Reason:      "URL match: " + request.URL,
				LogLevel:    "INFO",
			}
		}
	}
	
	return nil
}

// evaluateHeaderRules checks header-based rules
func (f *FirewallImpl) evaluateHeaderRules(request *RequestContext) *EvaluationResult {
	f.mutex.RLock()
	rules := f.rulesByType[HeaderRule]
	f.mutex.RUnlock()
	
	if len(rules) == 0 || len(request.Headers) == 0 {
		return nil
	}
	
	for _, rule := range rules {
		if !rule.IsEnabled || rule.HeaderName == "" {
			continue
		}
		
		// Check if header exists
		headerValue, exists := request.Headers[rule.HeaderName]
		if !exists {
			continue
		}
		
		// If pattern is defined, check header value
		if rule.Pattern != "" {
			pattern, err := regexp.Compile(rule.Pattern)
			if err != nil {
				continue
			}
			
			if pattern.MatchString(headerValue) {
				return &EvaluationResult{
					Action:      rule.Action,
					MatchedRule: rule,
					Reason:      "Header match: " + rule.HeaderName,
					LogLevel:    "INFO",
				}
			}
		} else if rule.HeaderValue != "" {
			// Direct string comparison
			if rule.HeaderValue == headerValue {
				return &EvaluationResult{
					Action:      rule.Action,
					MatchedRule: rule,
					Reason:      "Header match: " + rule.HeaderName,
					LogLevel:    "INFO",
				}
			}
		} else {
			// Just checking for header presence
			return &EvaluationResult{
				Action:      rule.Action,
				MatchedRule: rule,
				Reason:      "Header presence: " + rule.HeaderName,
				LogLevel:    "INFO",
			}
		}
	}
	
	return nil
}

// evaluateRateRules checks rate limit rules
func (f *FirewallImpl) evaluateRateRules(request *RequestContext) *EvaluationResult {
	f.mutex.RLock()
	rules := f.rulesByType[RateRule]
	f.mutex.RUnlock()
	
	if len(rules) == 0 {
		return nil
	}
	
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		
		// Determine the key for rate limiting
		var key string
		if rule.Pattern == "ip" {
			key = "ip:" + request.IP.String()
		} else if rule.Pattern == "session" && request.SessionID != "" {
			key = "session:" + request.SessionID
		} else if rule.Pattern == "user" && request.UserID != "" {
			key = "user:" + request.UserID
		} else if strings.HasPrefix(rule.Pattern, "header:") && len(request.Headers) > 0 {
			headerName := strings.TrimPrefix(rule.Pattern, "header:")
			if headerValue, exists := request.Headers[headerName]; exists {
				key = "header:" + headerName + ":" + headerValue
			}
		} else if rule.Pattern == "path" && request.Path != "" {
			key = "path:" + request.Path
		} else {
			// Default to IP
			key = "ip:" + request.IP.String()
		}
		
		// Check rate limit
		if key != "" {
			if !f.rateLimiter.Allow(key, rule.RateLimit, rule.RatePeriod) {
				remaining, retryAfter, _ := f.rateLimiter.GetRemaining(key)
				
				return &EvaluationResult{
					Action:      rule.Action,
					MatchedRule: rule,
					Reason:      "Rate limit exceeded for " + key,
					LogLevel:    "WARN",
					ThrottleFor: retryAfter,
				}
			}
		}
	}
	
	return nil
}

// evaluateGeoRules checks geo-based rules
func (f *FirewallImpl) evaluateGeoRules(request *RequestContext) *EvaluationResult {
	f.mutex.RLock()
	rules := f.rulesByType[GeoRule]
	f.mutex.RUnlock()
	
	if len(rules) == 0 || request.Country == "" {
		return nil
	}
	
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		
		for _, country := range rule.Countries {
			if strings.EqualFold(country, request.Country) {
				return &EvaluationResult{
					Action:      rule.Action,
					MatchedRule: rule,
					Reason:      "Geo match: " + request.Country,
					LogLevel:    "INFO",
				}
			}
		}
	}
	
	return nil
}

// evaluateContentRules checks content-based rules
func (f *FirewallImpl) evaluateContentRules(request *RequestContext) *EvaluationResult {
	// Content rules are more complex and might need access to the request body
	// For now, we just provide a basic implementation based on path and user agent
	
	f.mutex.RLock()
	rules := f.rulesByType[ContentRule]
	f.mutex.RUnlock()
	
	if len(rules) == 0 {
		return nil
	}
	
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue
		}
		
		// Check against user agent
		if request.UserAgent != "" && pattern.MatchString(request.UserAgent) {
			return &EvaluationResult{
				Action:      rule.Action,
				MatchedRule: rule,
				Reason:      "Content match in User-Agent",
				LogLevel:    "INFO",
			}
		}
		
		// Check against path
		if request.Path != "" && pattern.MatchString(request.Path) {
			return &EvaluationResult{
				Action:      rule.Action,
				MatchedRule: rule,
				Reason:      "Content match in Path",
				LogLevel:    "INFO",
			}
		}
	}
	
	return nil
}
