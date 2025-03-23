package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProxyTarget represents a backend service target
type ProxyTarget struct {
	ID        string
	URL       *url.URL
	Health    bool
	LastCheck time.Time
}

// ProxyRoute represents a routing configuration
type ProxyRoute struct {
	ID          string
	Path        string
	Targets     []*ProxyTarget
	LoadBalance bool
	current     int
	mu          sync.RWMutex
}

// Manager handles reverse proxy operations
type Manager struct {
	routes    map[string]*ProxyRoute
	mu        sync.RWMutex
	tlsConfig *tls.Config
}

// NewManager creates a new proxy manager
func NewManager(tlsConfig *tls.Config) *Manager {
	return &Manager{
		routes:    make(map[string]*ProxyRoute),
		tlsConfig: tlsConfig,
	}
}

// AddRoute creates a new proxy route
func (m *Manager) AddRoute(path string, targetURLs []string, loadBalance bool) (*ProxyRoute, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	targets := make([]*ProxyTarget, 0, len(targetURLs))
	for _, targetURL := range targetURLs {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL %s: %v", targetURL, err)
		}

		target := &ProxyTarget{
			ID:        uuid.New().String(),
			URL:       parsedURL,
			Health:    true,
			LastCheck: time.Now(),
		}
		targets = append(targets, target)
	}

	route := &ProxyRoute{
		ID:          uuid.New().String(),
		Path:        path,
		Targets:     targets,
		LoadBalance: loadBalance,
	}

	m.routes[path] = route
	return route, nil
}

// RemoveRoute removes a proxy route
func (m *Manager) RemoveRoute(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.routes, path)
}

// UpdateRoute updates an existing proxy route
func (m *Manager) UpdateRoute(path string, targetURLs []string, loadBalance bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	route, exists := m.routes[path]
	if !exists {
		return fmt.Errorf("route not found: %s", path)
	}

	targets := make([]*ProxyTarget, 0, len(targetURLs))
	for _, targetURL := range targetURLs {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			return fmt.Errorf("invalid target URL %s: %v", targetURL, err)
		}

		target := &ProxyTarget{
			ID:        uuid.New().String(),
			URL:       parsedURL,
			Health:    true,
			LastCheck: time.Now(),
		}
		targets = append(targets, target)
	}

	route.mu.Lock()
	route.Targets = targets
	route.LoadBalance = loadBalance
	route.current = 0
	route.mu.Unlock()

	return nil
}

// getNextTarget implements round-robin load balancing
func (r *ProxyRoute) getNextTarget() *ProxyTarget {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Targets) == 0 {
		return nil
	}

	// Skip unhealthy targets
	var target *ProxyTarget
	checked := 0
	for checked < len(r.Targets) {
		target = r.Targets[r.current]
		r.current = (r.current + 1) % len(r.Targets)
		if target.Health {
			break
		}
		checked++
	}

	if !target.Health {
		return nil
	}

	return target
}

// ServeHTTP implements the http.Handler interface
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	route, exists := m.routes[r.URL.Path]
	m.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	var target *ProxyTarget
	if route.LoadBalance {
		target = route.getNextTarget()
	} else {
		route.mu.RLock()
		if len(route.Targets) > 0 {
			target = route.Targets[0]
		}
		route.mu.RUnlock()
	}

	if target == nil {
		http.Error(w, "No available backends", http.StatusServiceUnavailable)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.URL.Scheme
			req.URL.Host = target.URL.Host
			req.URL.Path = target.URL.Path
			if target.URL.RawQuery == "" {
				req.URL.RawQuery = target.URL.RawQuery
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "WebWorks-Proxy")
			}
		},
		Transport: &http.Transport{
			TLSClientConfig: m.tlsConfig,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			target.Health = false
			target.LastCheck = time.Now()
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		},
	}

	proxy.ServeHTTP(w, r)
}
