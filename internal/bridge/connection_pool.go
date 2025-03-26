package bridge

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters"
	"go.uber.org/zap"
)

// Pool errors
var (
	ErrPoolClosed     = errors.New("connection pool is closed")
	ErrAcquireTimeout = errors.New("timeout acquiring connection from pool")
	ErrPoolExhausted  = errors.New("connection pool exhausted")
)

// PoolConfig defines configuration for the connection pool
type PoolConfig struct {
	// Maximum number of connections to keep in the pool
	MaxConnections int

	// Maximum idle time before a connection is removed from the pool
	MaxIdleTime time.Duration

	// Maximum time to wait for a connection from the pool
	AcquireTimeout time.Duration

	// Whether to validate connections when taking from pool
	ValidateOnBorrow bool

	// How often to run cleanup routine
	CleanupInterval time.Duration

	// Maximum number of connections to create in a single burst
	MaxBurstSize int
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConnections:   100,
		MaxIdleTime:      5 * time.Minute,
		AcquireTimeout:   30 * time.Second,
		ValidateOnBorrow: true,
		CleanupInterval:  1 * time.Minute,
		MaxBurstSize:     10,
	}
}

// PoolStats tracks statistics about the connection pool
type PoolStats struct {
	// Total number of connections managed by the pool
	TotalConnections int64

	// Number of idle connections in the pool
	IdleConnections int64

	// Number of active connections currently in use
	ActiveConnections int64

	// Number of connection acquisitions
	Acquisitions int64

	// Number of connection returns
	Returns int64

	// Number of connection creations
	Creations int64

	// Number of connection closures
	Closures int64

	// Number of connection timeouts during acquisition
	Timeouts int64

	// Number of failed validations
	ValidationFailures int64

	// Number of times pool was exhausted
	Exhaustions int64
}

// ConnectionPool manages a pool of connections for a specific adapter
type ConnectionPool struct {
	config         PoolConfig
	adapterFactory adapters.AdapterFactory
	adapterConfig  adapters.AdapterConfig
	available      chan adapters.Adapter
	inUse          sync.Map // adapters.Adapter -> time.Time
	logger         *zap.SugaredLogger
	closed         int32 // atomic flag
	stats          PoolStats
	statsMutex     sync.Mutex // Only needed for methods that update multiple stats fields atomically
	lastCleanup    time.Time
	lifecycle      *sync.WaitGroup // For graceful shutdown
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(
	factory adapters.AdapterFactory,
	config adapters.AdapterConfig,
	poolConfig PoolConfig,
	logger *zap.SugaredLogger,
) *ConnectionPool {
	// Apply defaults for unspecified options
	if poolConfig.MaxConnections <= 0 {
		poolConfig.MaxConnections = DefaultPoolConfig().MaxConnections
	}

	if poolConfig.MaxIdleTime <= 0 {
		poolConfig.MaxIdleTime = DefaultPoolConfig().MaxIdleTime
	}

	if poolConfig.AcquireTimeout <= 0 {
		poolConfig.AcquireTimeout = DefaultPoolConfig().AcquireTimeout
	}

	if poolConfig.CleanupInterval <= 0 {
		poolConfig.CleanupInterval = DefaultPoolConfig().CleanupInterval
	}

	if poolConfig.MaxBurstSize <= 0 {
		poolConfig.MaxBurstSize = DefaultPoolConfig().MaxBurstSize
	}

	pool := &ConnectionPool{
		config:         poolConfig,
		adapterFactory: factory,
		adapterConfig:  config,
		available:      make(chan adapters.Adapter, poolConfig.MaxConnections),
		logger:         logger,
		lastCleanup:    time.Now(),
		lifecycle:      &sync.WaitGroup{},
	}

	// Start background cleanup
	pool.lifecycle.Add(1)
	go func() {
		defer pool.lifecycle.Done()
		pool.periodicCleanup()
	}()

	return pool
}

// Acquire gets a connection from the pool or creates a new one
func (p *ConnectionPool) Acquire(ctx context.Context) (adapters.Adapter, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, ErrPoolClosed
	}

	// Track acquisition attempt
	atomic.AddInt64(&p.stats.Acquisitions, 1)

	// Try to get from pool first (non-blocking check)
	select {
	case adapter := <-p.available:
		atomic.AddInt64(&p.stats.IdleConnections, -1)

		if p.config.ValidateOnBorrow {
			// Check if the connection is still valid
			if err := p.validateAdapter(adapter); err != nil {
				atomic.AddInt64(&p.stats.ValidationFailures, 1)
				p.logger.Warnw("Invalid connection in pool, creating new one", "error", err)

				// Close the invalid connection
				adapter.Close()
				atomic.AddInt64(&p.stats.Closures, 1)
				atomic.AddInt64(&p.stats.TotalConnections, -1)

				// Create a new connection
				return p.createNewConnection(ctx)
			}
		}

		// Mark as in use with current timestamp
		p.inUse.Store(adapter, time.Now())
		atomic.AddInt64(&p.stats.ActiveConnections, 1)

		return adapter, nil
	default:
		// Pool is empty, check if we can create a new connection
		totalConns := atomic.LoadInt64(&p.stats.TotalConnections)
		if totalConns < int64(p.config.MaxConnections) {
			return p.createNewConnection(ctx)
		}

		// Pool is at capacity, need to wait for a connection
		return p.waitForConnection(ctx)
	}
}

// createNewConnection creates a new adapter connection
func (p *ConnectionPool) createNewConnection(ctx context.Context) (adapters.Adapter, error) {
	// Create a new adapter
	adapter, err := p.adapterFactory(p.adapterConfig)
	if err != nil {
		return nil, err
	}

	// Connect the adapter
	if err := adapter.Connect(ctx); err != nil {
		return nil, err
	}

	// Update stats
	atomic.AddInt64(&p.stats.Creations, 1)
	atomic.AddInt64(&p.stats.TotalConnections, 1)
	atomic.AddInt64(&p.stats.ActiveConnections, 1)

	// Mark as in use
	p.inUse.Store(adapter, time.Now())

	return adapter, nil
}

// waitForConnection waits for a connection to become available or creates new ones if possible
func (p *ConnectionPool) waitForConnection(ctx context.Context) (adapters.Adapter, error) {
	// Set up a timeout for waiting
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.AcquireTimeout)
	defer cancel()

	// Wait for a connection to become available or timeout
	select {
	case adapter := <-p.available:
		atomic.AddInt64(&p.stats.IdleConnections, -1)

		if p.config.ValidateOnBorrow {
			if err := p.validateAdapter(adapter); err != nil {
				atomic.AddInt64(&p.stats.ValidationFailures, 1)
				adapter.Close()
				atomic.AddInt64(&p.stats.Closures, 1)
				atomic.AddInt64(&p.stats.TotalConnections, -1)

				// Try again with a reduced timeout
				remainingTime := p.config.AcquireTimeout - time.Since(time.Now().Add(-p.config.AcquireTimeout))
				if remainingTime <= 0 {
					atomic.AddInt64(&p.stats.Timeouts, 1)
					return nil, ErrAcquireTimeout
				}

				newCtx, newCancel := context.WithTimeout(ctx, remainingTime)
				defer newCancel()
				return p.Acquire(newCtx)
			}
		}

		p.inUse.Store(adapter, time.Now())
		atomic.AddInt64(&p.stats.ActiveConnections, 1)
		return adapter, nil

	case <-timeoutCtx.Done():
		atomic.AddInt64(&p.stats.Timeouts, 1)
		return nil, ErrAcquireTimeout
	}
}

// Return returns a connection to the pool
func (p *ConnectionPool) Return(adapter adapters.Adapter) {
	if adapter == nil {
		return
	}

	if atomic.LoadInt32(&p.closed) == 1 {
		// Pool is closed, just close the adapter
		adapter.Close()
		atomic.AddInt64(&p.stats.Closures, 1)
		atomic.AddInt64(&p.stats.TotalConnections, -1)
		return
	}

	// Remove from in-use map
	p.inUse.Delete(adapter)
	atomic.AddInt64(&p.stats.ActiveConnections, -1)
	atomic.AddInt64(&p.stats.Returns, 1)

	// Try to add back to available pool, if full just close it
	select {
	case p.available <- adapter:
		atomic.AddInt64(&p.stats.IdleConnections, 1)
	default:
		// Pool is full, close the connection
		adapter.Close()
		atomic.AddInt64(&p.stats.Closures, 1)
		atomic.AddInt64(&p.stats.TotalConnections, -1)
	}
}

// Close closes the connection pool and all connections
func (p *ConnectionPool) Close() {
	// Only allow Close to run once
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return
	}

	p.logger.Info("Shutting down connection pool")

	// Close all connections in the available pool
	// We use a separate goroutine to drain to avoid blocking if channel is full
	go func() {
		for {
			select {
			case adapter := <-p.available:
				atomic.AddInt64(&p.stats.IdleConnections, -1)
				if adapter != nil {
					adapter.Close()
					atomic.AddInt64(&p.stats.Closures, 1)
					atomic.AddInt64(&p.stats.TotalConnections, -1)
				}
			default:
				// Channel drained
				close(p.available)
				return
			}
		}
	}()

	// Close all in-use connections
	p.inUse.Range(func(key, value interface{}) bool {
		adapter := key.(adapters.Adapter)
		adapter.Close()
		atomic.AddInt64(&p.stats.Closures, 1)
		atomic.AddInt64(&p.stats.TotalConnections, -1)
		p.inUse.Delete(key)
		atomic.AddInt64(&p.stats.ActiveConnections, -1)
		return true
	})

	// Wait for cleanup goroutine to finish
	p.lifecycle.Wait()

	p.logger.Info("Connection pool shutdown complete")
}

// validateAdapter checks if an adapter is still valid
func (p *ConnectionPool) validateAdapter(adapter adapters.Adapter) error {
	// In a real implementation, we would ping or otherwise check
	// that the connection is still valid. This could involve:
	// 1. Checking last activity time
	// 2. Sending a ping/heartbeat message
	// 3. Checking connection state

	// Basic implementation - just check it's not nil
	if adapter == nil {
		return errors.New("nil adapter")
	}

	return nil
}

// periodicCleanup periodically cleans up idle connections
func (p *ConnectionPool) periodicCleanup() {
	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if atomic.LoadInt32(&p.closed) == 1 {
				return
			}

			p.cleanupIdleConnections()
		}
	}
}

// cleanupIdleConnections removes idle connections from the pool
func (p *ConnectionPool) cleanupIdleConnections() {
	// Skip if already cleaned up recently
	if time.Since(p.lastCleanup) < p.config.CleanupInterval/2 {
		return
	}

	p.lastCleanup = time.Now()

	// Calculate how many connections to remove based on utilization patterns
	idleCount := atomic.LoadInt64(&p.stats.IdleConnections)
	totalCount := atomic.LoadInt64(&p.stats.TotalConnections)
	activeCount := atomic.LoadInt64(&p.stats.ActiveConnections)

	// Only remove connections if we have more than 25% of connections idle
	// and maintain at least 10% buffer of idle connections for bursts
	targetIdle := totalCount / 10   // 10% buffer
	if idleCount > targetIdle*2.5 { // more than 25% idle
		toRemove := idleCount - targetIdle
		if toRemove > int64(p.config.MaxBurstSize) {
			toRemove = int64(p.config.MaxBurstSize) // Don't remove too many at once
		}

		p.logger.Debugw("Cleaning up idle connections",
			"total", totalCount,
			"active", activeCount,
			"idle", idleCount,
			"toRemove", toRemove)

		// Remove connections
		for i := int64(0); i < toRemove; i++ {
			select {
			case adapter := <-p.available:
				atomic.AddInt64(&p.stats.IdleConnections, -1)
				adapter.Close()
				atomic.AddInt64(&p.stats.Closures, 1)
				atomic.AddInt64(&p.stats.TotalConnections, -1)
			default:
				// No more idle connections to remove
				return
			}
		}
	}
}

// GetStats returns current pool statistics
func (p *ConnectionPool) GetStats() PoolStats {
	return PoolStats{
		TotalConnections:   atomic.LoadInt64(&p.stats.TotalConnections),
		IdleConnections:    atomic.LoadInt64(&p.stats.IdleConnections),
		ActiveConnections:  atomic.LoadInt64(&p.stats.ActiveConnections),
		Acquisitions:       atomic.LoadInt64(&p.stats.Acquisitions),
		Returns:            atomic.LoadInt64(&p.stats.Returns),
		Creations:          atomic.LoadInt64(&p.stats.Creations),
		Closures:           atomic.LoadInt64(&p.stats.Closures),
		Timeouts:           atomic.LoadInt64(&p.stats.Timeouts),
		ValidationFailures: atomic.LoadInt64(&p.stats.ValidationFailures),
		Exhaustions:        atomic.LoadInt64(&p.stats.Exhaustions),
	}
}

// IsClosed returns whether the pool is closed
func (p *ConnectionPool) IsClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}
