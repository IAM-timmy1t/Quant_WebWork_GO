package proxy

import (
	"context"
	"log"
	"net/http"
	"time"
)

// HealthChecker manages health checks for proxy targets
type HealthChecker struct {
	manager      *Manager
	checkTimeout time.Duration
	interval     time.Duration
	client       *http.Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(manager *Manager, checkTimeout, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		manager:      manager,
		checkTimeout: checkTimeout,
		interval:     interval,
		client: &http.Client{
			Timeout: checkTimeout,
			Transport: &http.Transport{
				TLSClientConfig: manager.tlsConfig,
			},
		},
	}
}

// Start begins the health checking process
func (h *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkAll()
		}
	}
}

// checkAll performs health checks on all targets
func (h *HealthChecker) checkAll() {
	h.manager.mu.RLock()
	defer h.manager.mu.RUnlock()

	for _, route := range h.manager.routes {
		route.mu.RLock()
		targets := route.Targets
		route.mu.RUnlock()

		for _, target := range targets {
			go h.checkTarget(target)
		}
	}
}

// checkTarget performs a health check on a single target
func (h *HealthChecker) checkTarget(target *ProxyTarget) {
	req, err := http.NewRequest(http.MethodGet, target.URL.String(), nil)
	if err != nil {
		log.Printf("Error creating health check request for %s: %v", target.URL, err)
		target.Health = false
		target.LastCheck = time.Now()
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		log.Printf("Health check failed for %s: %v", target.URL, err)
		target.Health = false
		target.LastCheck = time.Now()
		return
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as healthy
	target.Health = resp.StatusCode >= 200 && resp.StatusCode < 300
	target.LastCheck = time.Now()

	if !target.Health {
		log.Printf("Target %s is unhealthy (status code: %d)", target.URL, resp.StatusCode)
	}
}
