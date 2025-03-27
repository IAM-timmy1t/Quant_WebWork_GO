// rate_limiter_impl.go - Implementation of rate limiting functionality

package firewall

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiterConfig defines configuration for rate limiters
type RateLimiterConfig struct {
	// DefaultLimit is the default number of requests allowed in the interval
	DefaultLimit int

	// DefaultInterval is the default time interval for rate limiting
	DefaultInterval time.Duration

	// CleanupInterval is how often to clean up stale entries
	CleanupInterval time.Duration

	// MaxEntries is the maximum number of entries to track (0 = unlimited)
	MaxEntries int

	// BurstFactor allows temporary bursts above the limit
	BurstFactor float64
}

// DefaultRateLimiterConfig returns the default configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		DefaultLimit:    100,
		DefaultInterval: time.Minute,
		CleanupInterval: 10 * time.Minute,
		MaxEntries:      10000,
		BurstFactor:     1.5,
	}
}

// rateLimiterEntry tracks usage for a specific key
type rateLimiterEntry struct {
	count       int
	lastRequest time.Time
	limit       int
	interval    time.Duration
}

// RateLimiter defines the rate limiting interface
type RateLimiter interface {
	Allow(key string, limit int, period time.Duration) bool
	Reset(key string) error
	GetRemaining(key string) (int, time.Duration, error)
	SetLimit(key string, limit int, period time.Duration)
	RemoveKey(key string)
	Close()
	GetStats() map[string]interface{}
}

// MemoryRateLimiter implements the RateLimiter interface using in-memory storage
type MemoryRateLimiter struct {
	entries     map[string]*rateLimiterEntry
	config      RateLimiterConfig
	mutex       sync.RWMutex
	stopCleanup chan struct{}
	logger      *zap.SugaredLogger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) RateLimiter {
	// Use default config for any zero values
	defaultConfig := DefaultRateLimiterConfig()
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = defaultConfig.DefaultLimit
	}
	if config.DefaultInterval <= 0 {
		config.DefaultInterval = defaultConfig.DefaultInterval
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = defaultConfig.CleanupInterval
	}
	if config.BurstFactor <= 0 {
		config.BurstFactor = defaultConfig.BurstFactor
	}

	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()

	rl := &MemoryRateLimiter{
		entries:     make(map[string]*rateLimiterEntry),
		config:      config,
		stopCleanup: make(chan struct{}),
		logger:      sugar,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request is allowed under rate limits
func (rl *MemoryRateLimiter) Allow(key string, limit int, period time.Duration) bool {
	// Use defaults if parameters are invalid
	if limit <= 0 {
		limit = rl.config.DefaultLimit
	}
	if period <= 0 {
		period = rl.config.DefaultInterval
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Get or create entry
	entry, exists := rl.entries[key]
	if !exists {
		// Check if we're at max entries
		if rl.config.MaxEntries > 0 && len(rl.entries) >= rl.config.MaxEntries {
			// Evict oldest entry
			var oldestKey string
			var oldestTime time.Time
			first := true

			for k, e := range rl.entries {
				if first || e.lastRequest.Before(oldestTime) {
					oldestKey = k
					oldestTime = e.lastRequest
					first = false
				}
			}

			if !first {
				delete(rl.entries, oldestKey)
			}
		}

		// Create new entry
		entry = &rateLimiterEntry{
			count:       1,
			lastRequest: now,
			limit:       limit,
			interval:    period,
		}
		rl.entries[key] = entry
		return true
	}

	// If interval has elapsed, reset counter
	if now.Sub(entry.lastRequest) > entry.interval {
		entry.count = 1
		entry.lastRequest = now
		entry.limit = limit     // Update limit
		entry.interval = period // Update interval
		return true
	}

	// Check if under limit
	if entry.count < entry.limit {
		entry.count++
		entry.lastRequest = now
		return true
	}

	// Check if burst is allowed
	burstLimit := int(float64(entry.limit) * rl.config.BurstFactor)
	if entry.count < burstLimit {
		entry.count++
		entry.lastRequest = now
		rl.logger.Debugw("Rate limit burst allowed", "key", key, "count", entry.count, "limit", entry.limit)
		return true
	}

	// Rate limit exceeded
	rl.logger.Infow("Rate limit exceeded", "key", key, "count", entry.count, "limit", entry.limit)
	return false
}

// Reset resets the counter for a key
func (rl *MemoryRateLimiter) Reset(key string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if entry, exists := rl.entries[key]; exists {
		entry.count = 0
		entry.lastRequest = time.Now()
	}

	return nil
}

// GetRemaining returns the number of requests remaining and wait time
func (rl *MemoryRateLimiter) GetRemaining(key string) (int, time.Duration, error) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	entry, exists := rl.entries[key]
	if !exists {
		return rl.config.DefaultLimit, 0, nil
	}

	// If interval has elapsed, full limit would be available
	now := time.Now()
	elapsed := now.Sub(entry.lastRequest)
	if elapsed > entry.interval {
		return entry.limit, 0, nil
	}

	// Calculate remaining
	remaining := entry.limit - entry.count
	if remaining < 0 {
		remaining = 0
	}

	// Calculate time until reset
	timeUntilReset := entry.interval - elapsed

	return remaining, timeUntilReset, nil
}

// SetLimit sets a custom limit for a key
func (rl *MemoryRateLimiter) SetLimit(key string, limit int, period time.Duration) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	entry, exists := rl.entries[key]
	if !exists {
		entry = &rateLimiterEntry{
			count:       0,
			lastRequest: time.Now(),
			limit:       limit,
			interval:    period,
		}
		rl.entries[key] = entry
	} else {
		entry.limit = limit
		entry.interval = period
	}
}

// RemoveKey removes a key from the rate limiter
func (rl *MemoryRateLimiter) RemoveKey(key string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.entries, key)
}

// cleanupLoop periodically cleans up stale entries
func (rl *MemoryRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCleanup:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup removes stale entries
func (rl *MemoryRateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	keysToDelete := make([]string, 0)

	for key, entry := range rl.entries {
		// Remove entries that haven't been used in 3x their interval
		if now.Sub(entry.lastRequest) > entry.interval*3 {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(rl.entries, key)
	}

	if len(keysToDelete) > 0 {
		rl.logger.Debugw("Cleaned up rate limiter entries", "count", len(keysToDelete))
	}
}

// Close stops the background cleanup process
func (rl *MemoryRateLimiter) Close() {
	close(rl.stopCleanup)
}

// GetStats returns statistics about the rate limiter
func (rl *MemoryRateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	return map[string]interface{}{
		"total_entries": len(rl.entries),
		"config": map[string]interface{}{
			"default_limit":    rl.config.DefaultLimit,
			"default_interval": rl.config.DefaultInterval.String(),
			"cleanup_interval": rl.config.CleanupInterval.String(),
			"max_entries":      rl.config.MaxEntries,
			"burst_factor":     rl.config.BurstFactor,
		},
	}
}
