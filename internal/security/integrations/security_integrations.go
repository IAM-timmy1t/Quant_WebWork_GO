// security_integrations.go - Integration for security components

package integrations

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security/firewall"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security/ipmasking"
	"go.uber.org/zap"
)

// SecurityManager integrates all security components
type SecurityManager struct {
	firewall        firewall.Firewall
	ipMasking       ipmasking.IPMasker
	securityMonitor *security.Monitor
	logger          *zap.SugaredLogger
	enabled         bool
}

// Options configures the security manager
type Options struct {
	FirewallEnabled       bool
	IPMaskingEnabled      bool
	MonitoringEnabled     bool
	DefaultFirewallAction firewall.Action
	IPMaskingOptions      *ipmasking.MaskingOptions
	SecurityMonitorConfig *security.Config
}

// DefaultOptions returns default security manager options
func DefaultOptions() *Options {
	return &Options{
		FirewallEnabled:       true,
		IPMaskingEnabled:      true,
		MonitoringEnabled:     true,
		DefaultFirewallAction: firewall.ActionDeny,
		IPMaskingOptions:      ipmasking.DefaultMaskingOptions(),
		SecurityMonitorConfig: nil, // Will use default config
	}
}

// NewSecurityManager creates a new security integration manager
func NewSecurityManager(options *Options, logger *zap.SugaredLogger) (*SecurityManager, error) {
	if options == nil {
		options = DefaultOptions()
	}

	if logger == nil {
		// Create a no-op logger if none provided
		noop := zap.NewNop()
		logger = noop.Sugar()
	}

	manager := &SecurityManager{
		logger:  logger,
		enabled: true,
	}

	// Initialize firewall if enabled
	if options.FirewallEnabled {
		rateLimiter := firewall.NewMemoryRateLimiter()
		fw := firewall.NewFirewall(rateLimiter, &firewallLogger{logger})
		manager.firewall = fw
		logger.Info("Firewall component initialized")
	}

	// Initialize IP masking if enabled
	if options.IPMaskingEnabled {
		ipMasker := ipmasking.NewManagerWithOptions(options.IPMaskingOptions, logger)
		manager.ipMasking = ipMasker
		logger.Info("IP masking component initialized")
	}

	// Initialize security monitor if enabled
	if options.MonitoringEnabled {
		var monitorConfig security.Config
		if options.SecurityMonitorConfig != nil {
			monitorConfig = *options.SecurityMonitorConfig
		} else {
			monitorConfig = security.DefaultConfig()
		}

		monitor, err := security.NewMonitor(monitorConfig)
		if err != nil {
			logger.Errorw("Failed to initialize security monitor", "error", err)
			return nil, err
		}
		manager.securityMonitor = monitor
		logger.Info("Security monitoring component initialized")
	}

	return manager, nil
}

// Start starts all security components
func (m *SecurityManager) Start() error {
	if m.ipMasking != nil && !m.ipMasking.IsEnabled() {
		if err := m.ipMasking.Start(); err != nil {
			m.logger.Errorw("Failed to start IP masking", "error", err)
			return err
		}
		m.logger.Info("IP masking started")
	}

	m.enabled = true
	m.logger.Info("Security manager started")
	return nil
}

// Stop stops all security components
func (m *SecurityManager) Stop() error {
	if m.ipMasking != nil && m.ipMasking.IsEnabled() {
		if err := m.ipMasking.Stop(); err != nil {
			m.logger.Errorw("Failed to stop IP masking", "error", err)
			return err
		}
		m.logger.Info("IP masking stopped")
	}

	if m.securityMonitor != nil {
		if err := m.securityMonitor.Close(); err != nil {
			m.logger.Errorw("Failed to close security monitor", "error", err)
			return err
		}
		m.logger.Info("Security monitor stopped")
	}

	m.enabled = false
	m.logger.Info("Security manager stopped")
	return nil
}

// IsEnabled returns whether the security manager is enabled
func (m *SecurityManager) IsEnabled() bool {
	return m.enabled
}

// GetFirewall returns the firewall component
func (m *SecurityManager) GetFirewall() firewall.Firewall {
	return m.firewall
}

// GetIPMasker returns the IP masking component
func (m *SecurityManager) GetIPMasker() ipmasking.IPMasker {
	return m.ipMasking
}

// GetSecurityMonitor returns the security monitor component
func (m *SecurityManager) GetSecurityMonitor() *security.Monitor {
	return m.securityMonitor
}

// CreateMiddlewares creates HTTP middleware chain for all security components
func (m *SecurityManager) CreateMiddlewares() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Apply middlewares in reverse order (last one applies first)

		// First apply IP masking
		if m.ipMasking != nil {
			ipMaskingMiddleware := ipmasking.NewHTTPIPMaskingMiddleware(m.ipMasking)
			next = ipMaskingMiddleware.Middleware(next)
		}

		// Then apply firewall
		if m.firewall != nil {
			next = m.createFirewallMiddleware(next)
		}

		// Finally, apply security monitoring
		if m.securityMonitor != nil {
			next = m.createMonitoringMiddleware(next)
		}

		return next
	}
}

// createFirewallMiddleware creates HTTP middleware for firewall
func (m *SecurityManager) createFirewallMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Extract IP
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		clientIP := net.ParseIP(ip)
		if clientIP == nil {
			// Could not parse IP, just pass through
			next.ServeHTTP(w, r)
			return
		}

		// Prepare request context for firewall
		requestCtx := &firewall.RequestContext{
			IP:        clientIP,
			URL:       r.URL.String(),
			Method:    r.Method,
			Headers:   make(map[string]string),
			UserAgent: r.UserAgent(),
			Path:      r.URL.Path,
			Timestamp: time.Now(),
		}

		// Copy headers
		for name, values := range r.Header {
			if len(values) > 0 {
				requestCtx.Headers[name] = values[0]
			}
		}

		// Evaluate firewall rules
		result := m.firewall.Evaluate(r.Context(), requestCtx)

		if result.Action == firewall.ActionDeny {
			// Blocked by firewall
			http.Error(w, "Forbidden", http.StatusForbidden)
			m.logger.Infow("Request blocked by firewall",
				"ip", clientIP.String(),
				"path", r.URL.Path,
				"method", r.Method,
				"reason", result.Reason,
			)
			return
		}

		// Apply rate limiting if needed
		if result.Action == firewall.ActionRate && result.ThrottleFor > 0 {
			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", "Too many requests")
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", result.ThrottleFor.Seconds()))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Continue with request
		next.ServeHTTP(w, r)
	})
}

// createMonitoringMiddleware creates HTTP middleware for security monitoring
func (m *SecurityManager) createMonitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Create security event
		event := security.Event{
			Type:      "http_request",
			Source:    "web",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_ip":  r.RemoteAddr,
				"user_agent": r.UserAgent(),
			},
		}

		// Process event asynchronously
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.securityMonitor.ProcessEvent(ctx, event); err != nil {
				m.logger.Errorw("Failed to process security event", "error", err)
			}
		}()

		// Continue with request
		next.ServeHTTP(w, r)
	})
}

// firewallLogger adapts zap.SugaredLogger to firewall.Logger
type firewallLogger struct {
	logger *zap.SugaredLogger
}

func (l *firewallLogger) Debug(msg string, fields map[string]interface{}) {
	l.logger.Debugw(msg, fieldsToArgs(fields)...)
}

func (l *firewallLogger) Info(msg string, fields map[string]interface{}) {
	l.logger.Infow(msg, fieldsToArgs(fields)...)
}

func (l *firewallLogger) Warn(msg string, fields map[string]interface{}) {
	l.logger.Warnw(msg, fieldsToArgs(fields)...)
}

func (l *firewallLogger) Error(msg string, fields map[string]interface{}) {
	l.logger.Errorw(msg, fieldsToArgs(fields)...)
}

// fieldsToArgs converts map to alternating key/value pairs for zap
func fieldsToArgs(fields map[string]interface{}) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}
