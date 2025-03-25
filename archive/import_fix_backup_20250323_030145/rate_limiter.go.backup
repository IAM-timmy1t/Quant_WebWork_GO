// rate_limiter.go - Rate limiting implementation

package firewall

import (
	"sync"
	"time"
)

// MemoryRateLimiter implements an in-memory token bucket rate limiter
type MemoryRateLimiter struct {
	buckets map[string]*tokenBucket
	mutex   sync.RWMutex
	// Add storage cleanup mechanism to prevent memory leaks
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	tokens        int
	capacity      int
	refillRate    float64 // tokens per nanosecond
	lastRefill    time.Time
	periodSeconds int64
}

// NewMemoryRateLimiter creates a new in-memory rate limiter
func NewMemoryRateLimiter() *MemoryRateLimiter {
	rl := &MemoryRateLimiter{
		buckets:         make(map[string]*tokenBucket),
		cleanupInterval: 10 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}
	
	// Start cleanup routine
	go rl.periodicCleanup()
	
	return rl
}

// Allow checks if a request is allowed under rate limits
func (rl *MemoryRateLimiter) Allow(key string, limit int, period time.Duration) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	bucket, exists := rl.buckets[key]
	if !exists {
		// Create a new bucket
		bucket = &tokenBucket{
			tokens:        limit - 1, // Start with all tokens minus the current request
			capacity:      limit,
			refillRate:    float64(limit) / float64(period.Nanoseconds()),
			lastRefill:    time.Now(),
			periodSeconds: int64(period.Seconds()),
		}
		rl.buckets[key] = bucket
		return true
	}
	
	// Refill the bucket based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	bucket.lastRefill = now
	
	// Calculate tokens to add
	tokensToAdd := float64(elapsed.Nanoseconds()) * bucket.refillRate
	
	// Add tokens up to capacity
	bucket.tokens += int(tokensToAdd)
	if bucket.tokens > bucket.capacity {
		bucket.tokens = bucket.capacity
	}
	
	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	
	return false
}

// Reset resets rate limiting for a key
func (rl *MemoryRateLimiter) Reset(key string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	if bucket, exists := rl.buckets[key]; exists {
		// Reset to full capacity
		bucket.tokens = bucket.capacity
		bucket.lastRefill = time.Now()
	}
	
	return nil
}

// GetRemaining returns how many requests remain in the current period
func (rl *MemoryRateLimiter) GetRemaining(key string) (int, time.Duration, error) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	bucket, exists := rl.buckets[key]
	if !exists {
		return 0, 0, ErrRateLimit
	}
	
	// Calculate current tokens
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	
	// Calculate tokens to add
	tokensToAdd := float64(elapsed.Nanoseconds()) * bucket.refillRate
	
	// Current tokens (not modifying the actual bucket)
	currentTokens := bucket.tokens + int(tokensToAdd)
	if currentTokens > bucket.capacity {
		currentTokens = bucket.capacity
	}
	
	// Calculate time until next token
	var waitTime time.Duration
	if currentTokens <= 0 {
		// Calculate time until at least one token is available
		tokensNeeded := 1 - currentTokens
		waitTime = time.Duration(float64(tokensNeeded) / bucket.refillRate)
	}
	
	return currentTokens, waitTime, nil
}

// periodicCleanup removes expired buckets to prevent memory leaks
func (rl *MemoryRateLimiter) periodicCleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
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

// cleanup removes expired buckets
func (rl *MemoryRateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	for key, bucket := range rl.buckets {
		// Remove buckets that haven't been used for more than 3x their period
		if now.Sub(bucket.lastRefill) > time.Duration(bucket.periodSeconds*3)*time.Second {
			delete(rl.buckets, key)
		}
	}
}

// Close stops the background cleanup process
func (rl *MemoryRateLimiter) Close() {
	close(rl.stopCleanup)
}

// Custom errors
var (
	ErrRateLimit = Error("rate limit error")
)

// Error type for the rate limiter
type Error string

func (e Error) Error() string {
	return string(e)
}
