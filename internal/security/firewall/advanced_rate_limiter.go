package firewall

import (
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
	"go.uber.org/zap"
)

// RateLimitStrategy defines how rate limiting is applied
type RateLimitStrategy string

const (
	// TokenBucket uses a token bucket algorithm
	TokenBucket RateLimitStrategy = "token-bucket"
	
	// SlidingWindow uses a sliding window approach
	SlidingWindow RateLimitStrategy = "sliding-window"
	
	// FixedWindow uses a fixed window approach
	FixedWindow RateLimitStrategy = "fixed-window"
)

// RateLimitConfig contains configuration for the rate limiter
type RateLimitConfig struct {
	// Strategy defines which algorithm to use
	Strategy RateLimitStrategy
	
	// RequestsPerSecond defines the maximum rate
	RequestsPerSecond float64
	
	// BurstSize defines how many requests can be made in a burst
	BurstSize int
	
	// ShardCount for lock striping to reduce contention (power of 2 recommended)
	ShardCount int
	
	// CleanupInterval defines how often to clean up expired entries
	CleanupInterval time.Duration
	
	// EntryTTL defines how long an entry stays in the limiter
	EntryTTL time.Duration
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow checks if a request is allowed and updates the limiter
	Allow(key string) bool
	
	// AllowN checks if N requests are allowed and updates the limiter
	AllowN(key string, n int) bool
	
	// GetLimit returns the current limit for a key
	GetLimit(key string) float64
	
	// SetLimit updates the limit for a key
	SetLimit(key string, limit float64)
	
	// Cleanup removes expired entries
	Cleanup()
}

// AdvancedRateLimiter implements rate limiting with lock striping for high-scale
type AdvancedRateLimiter struct {
	config      RateLimitConfig
	shards      []*limiterShard
	hasher      func(string) uint32
	logger      *zap.SugaredLogger
	metrics     metrics.Collector
	lastCleanup time.Time
	mu          sync.Mutex // Only used for cleanup coordination
}

// limiterShard represents a shard of the rate limiter to reduce lock contention
type limiterShard struct {
	entries map[string]*limiterEntry
	mu      sync.RWMutex
}

// limiterEntry tracks rate limit state for a specific key
type limiterEntry struct {
	tokens         float64
	limit          float64
	lastRefill     time.Time
	lastSeen       time.Time
	totalAllowed   int64
	totalRejected  int64
	consecutiveHit int
}

// NewAdvancedRateLimiter creates a new rate limiter
func NewAdvancedRateLimiter(config RateLimitConfig, logger *zap.SugaredLogger, metrics metrics.Collector) *AdvancedRateLimiter {
	// Default values if not specified
	if config.ShardCount <= 0 {
		config.ShardCount = 256 // Default to 256 shards
	}
	
	if config.BurstSize <= 0 {
		config.BurstSize = int(config.RequestsPerSecond * 2)
	}
	
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	
	if config.EntryTTL <= 0 {
		config.EntryTTL = 30 * time.Minute
	}
	
	// Initialize shards
	shards := make([]*limiterShard, config.ShardCount)
	for i := 0; i < config.ShardCount; i++ {
		shards[i] = &limiterShard{
			entries: make(map[string]*limiterEntry),
		}
	}
	
	limiter := &AdvancedRateLimiter{
		config:      config,
		shards:      shards,
		hasher:      fnv32Hash, // Use FNV hash for key distribution
		logger:      logger,
		metrics:     metrics,
		lastCleanup: time.Now(),
	}
	
	// Start cleanup goroutine
	go limiter.periodicCleanup()
	
	return limiter
}

// getShard returns the shard for a key
func (rl *AdvancedRateLimiter) getShard(key string) *limiterShard {
	hash := rl.hasher(key)
	index := hash % uint32(len(rl.shards))
	return rl.shards[index]
}

// getOrCreateEntry gets an existing entry or creates a new one
func (rl *AdvancedRateLimiter) getOrCreateEntry(shard *limiterShard, key string) *limiterEntry {
	shard.mu.RLock()
	entry, exists := shard.entries[key]
	shard.mu.RUnlock()
	
	if exists {
		return entry
	}
	
	// Create new entry
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	// Double-check after acquiring write lock
	if entry, exists = shard.entries[key]; exists {
		return entry
	}
	
	// Create new entry with initial token count
	now := time.Now()
	entry = &limiterEntry{
		tokens:     float64(rl.config.BurstSize),
		limit:      rl.config.RequestsPerSecond,
		lastRefill: now,
		lastSeen:   now,
	}
	shard.entries[key] = entry
	
	return entry
}

// Allow checks if a request is allowed based on rate limits
func (rl *AdvancedRateLimiter) Allow(key string) bool {
	return rl.AllowN(key, 1)
}

// AllowN checks if n requests are allowed based on rate limits
func (rl *AdvancedRateLimiter) AllowN(key string, n int) bool {
	shard := rl.getShard(key)
	entry := rl.getOrCreateEntry(shard, key)
	
	now := time.Now()
	
	// Update entry with thread safety
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	// Update last seen time
	entry.lastSeen = now
	
	// Refill tokens based on time elapsed
	elapsed := now.Sub(entry.lastRefill).Seconds()
	entry.lastRefill = now
	
	// Calculate how many new tokens to add based on rate
	newTokens := elapsed * entry.limit
	
	// Add new tokens, but don't exceed burst size
	entry.tokens = min(float64(rl.config.BurstSize), entry.tokens+newTokens)
	
	// Check if we have enough tokens
	if entry.tokens >= float64(n) {
		entry.tokens -= float64(n)
		entry.totalAllowed++
		entry.consecutiveHit = 0
		
		// Record metrics
		if rl.metrics != nil {
			rl.metrics.Collect("security", "rate_limit_allowed", 1, map[string]string{
				"key": key,
			})
		}
		
		return true
	}
	
	// Not enough tokens, request rejected
	entry.totalRejected++
	entry.consecutiveHit++
	
	// Record metrics
	if rl.metrics != nil {
		rl.metrics.Collect("security", "rate_limit_rejected", 1, map[string]string{
			"key": key,
		})
	}
	
	// Log on high consecutive hits
	if entry.consecutiveHit > 10 && entry.consecutiveHit%10 == 0 {
		rl.logger.Warnw("Rate limit exceeded multiple times",
			"key", key,
			"consecutiveHits", entry.consecutiveHit,
			"limit", entry.limit,
			"totalRejected", entry.totalRejected,
		)
	}
	
	return false
}

// GetLimit returns the current limit for a key
func (rl *AdvancedRateLimiter) GetLimit(key string) float64 {
	shard := rl.getShard(key)
	
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	
	entry, exists := shard.entries[key]
	if !exists {
		return rl.config.RequestsPerSecond
	}
	
	return entry.limit
}

// SetLimit updates the limit for a key
func (rl *AdvancedRateLimiter) SetLimit(key string, limit float64) {
	shard := rl.getShard(key)
	entry := rl.getOrCreateEntry(shard, key)
	
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	// Update the limit
	entry.limit = limit
	
	// Log the change
	rl.logger.Infow("Rate limit updated",
		"key", key,
		"newLimit", limit,
	)
}

// periodicCleanup runs cleanup at regular intervals
func (rl *AdvancedRateLimiter) periodicCleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.Cleanup()
	}
}

// Cleanup removes expired entries
func (rl *AdvancedRateLimiter) Cleanup() {
	// Use a mutex to ensure only one cleanup runs at a time
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Skip if we recently cleaned up
	if time.Since(rl.lastCleanup) < time.Minute {
		return
	}
	
	rl.lastCleanup = time.Now()
	expiryTime := time.Now().Add(-rl.config.EntryTTL)
	
	// Track how many entries we clean up
	removed := 0
	
	// Clean up each shard
	for _, shard := range rl.shards {
		// Collect keys to remove
		keysToRemove := []string{}
		
		shard.mu.RLock()
		for key, entry := range shard.entries {
			if entry.lastSeen.Before(expiryTime) {
				keysToRemove = append(keysToRemove, key)
			}
		}
		shard.mu.RUnlock()
		
		// Remove the expired entries
		if len(keysToRemove) > 0 {
			shard.mu.Lock()
			for _, key := range keysToRemove {
				delete(shard.entries, key)
				removed++
			}
			shard.mu.Unlock()
		}
	}
	
	if removed > 0 {
		rl.logger.Infow("Cleaned up rate limiter entries",
			"removed", removed,
		)
	}
	
	// Record metrics
	if rl.metrics != nil {
		rl.metrics.Collect("security", "rate_limit_cleanup", float64(removed), nil)
	}
}

// fnv32Hash is a simple 32-bit FNV-1a hash function
func fnv32Hash(key string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	
	hash := uint32(offset32)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= prime32
	}
	
	return hash
}

// min returns the minimum of two values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
