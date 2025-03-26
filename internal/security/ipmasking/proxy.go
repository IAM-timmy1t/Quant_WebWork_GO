package ipmasking

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ProxyConfig contains configuration for the masking proxy
type ProxyConfig struct {
	// Whether to enable the proxy
	Enabled bool

	// The masked domain to use
	MaskedDomain string

	// Default target for unmatched requests
	DefaultTarget *url.URL

	// Target mappings (path prefix -> target URL)
	TargetMappings map[string]*url.URL

	// Connection timeouts
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration

	// Whether to use secure cookies
	SecureCookies bool

	// Whether to strip incoming IP addresses from headers
	StripIPHeaders bool

	// Whether to enable DNS privacy protections
	EnableDNSPrivacy bool

	// Whether to sanitize URL query parameters containing PII
	SanitizeURLParams bool

	// List of headers to remove from requests
	HeadersToRemove []string

	// Whether to mask IPs in logs
	MaskIPsInLogs bool
}

// DefaultProxyConfig returns a default proxy configuration
func DefaultProxyConfig() *ProxyConfig {
	return &ProxyConfig{
		Enabled:           false,
		MaskedDomain:      "masked-domain.example.com",
		ConnectTimeout:    10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
		SecureCookies:     true,
		StripIPHeaders:    true,
		EnableDNSPrivacy:  true,
		SanitizeURLParams: true,
		HeadersToRemove: []string{
			"X-Forwarded-For",
			"X-Real-IP",
			"CF-Connecting-IP",
			"True-Client-IP",
		},
		MaskIPsInLogs:  true,
		TargetMappings: make(map[string]*url.URL),
	}
}

// MaskingProxy handles HTTP proxy with IP masking
type MaskingProxy struct {
	config         *ProxyConfig
	ipManager      *Manager
	logger         *zap.SugaredLogger
	defaultHandler http.Handler
	pathHandlers   map[string]http.Handler
}

// NewMaskingProxy creates a new masking proxy
func NewMaskingProxy(config *ProxyConfig, ipManager *Manager, logger *zap.SugaredLogger) (*MaskingProxy, error) {
	if config == nil {
		config = DefaultProxyConfig()
	}

	if ipManager == nil {
		ipManager = NewManager(logger)
	}

	if logger == nil {
		noop := zap.NewNop()
		logger = noop.Sugar()
	}

	proxy := &MaskingProxy{
		config:       config,
		ipManager:    ipManager,
		logger:       logger,
		pathHandlers: make(map[string]http.Handler),
	}

	// Set up the default handler if a default target is configured
	if config.DefaultTarget != nil {
		proxy.defaultHandler = proxy.createReverseProxy(config.DefaultTarget)
	}

	// Set up path-specific handlers
	for path, target := range config.TargetMappings {
		proxy.pathHandlers[path] = proxy.createReverseProxy(target)
	}

	return proxy, nil
}

// Start starts the proxy server
func (p *MaskingProxy) Start(addr string) error {
	if !p.config.Enabled {
		p.logger.Info("Proxy is disabled, not starting")
		return nil
	}

	// Start IP masking manager if not already started
	if !p.ipManager.IsEnabled() {
		if err := p.ipManager.Start(); err != nil {
			return err
		}
	}

	// Create server with appropriate timeouts
	server := &http.Server{
		Addr:         addr,
		Handler:      p,
		ReadTimeout:  p.config.ReadTimeout,
		WriteTimeout: p.config.WriteTimeout,
		IdleTimeout:  p.config.IdleTimeout,
	}

	p.logger.Infow("Starting masking proxy", "address", addr)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Errorw("Proxy server failed", "error", err)
		}
	}()

	return nil
}

// ServeHTTP implements the http.Handler interface
func (p *MaskingProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Skip processing if proxy is disabled
	if !p.config.Enabled {
		http.Error(w, "Proxy is disabled", http.StatusServiceUnavailable)
		return
	}

	// Mask client IP
	clientIP := net.ParseIP(getClientIP(r))
	if clientIP != nil {
		maskedIP := p.ipManager.GetMaskedIP(clientIP)
		r = setClientIP(r, maskedIP.String())
	}

	// Remove sensitive headers
	if p.config.StripIPHeaders {
		for _, header := range p.config.HeadersToRemove {
			r.Header.Del(header)
		}
	}

	// Sanitize URL parameters if enabled
	if p.config.SanitizeURLParams {
		sanitizeURLParams(r)
	}

	// Find the appropriate handler for the request path
	handler := p.findHandler(r.URL.Path)
	if handler == nil {
		http.Error(w, "No route for this path", http.StatusNotFound)
		return
	}

	// Serve the request
	handler.ServeHTTP(w, r)
}

// createReverseProxy creates a reverse proxy handler for the given target
func (p *MaskingProxy) createReverseProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize director to modify requests
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Call the default director
		defaultDirector(req)

		// Set host header
		req.Host = target.Host

		// Add masking headers
		req.Header.Set("X-Forwarded-Host", p.config.MaskedDomain)
		req.Header.Set("X-Masked-By", "QUANT-WebWork-GO")

		// Modify origin header if present
		if origin := req.Header.Get("Origin"); origin != "" {
			req.Header.Set("Origin", strings.Replace(origin, req.Host, p.config.MaskedDomain, 1))
		}
	}

	// Customize ModifyResponse to process the response
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Replace any occurrences of the real domain with the masked one
		if resp.Header.Get("Location") != "" {
			location := resp.Header.Get("Location")
			maskedLocation := strings.Replace(location, target.Host, p.config.MaskedDomain, 1)
			resp.Header.Set("Location", maskedLocation)
		}

		// Same for Content-Location
		if resp.Header.Get("Content-Location") != "" {
			contentLocation := resp.Header.Get("Content-Location")
			maskedContentLocation := strings.Replace(contentLocation, target.Host, p.config.MaskedDomain, 1)
			resp.Header.Set("Content-Location", maskedContentLocation)
		}

		// Modify cookies to use the masked domain
		if cookies := resp.Cookies(); len(cookies) > 0 {
			for _, cookie := range cookies {
				cookie.Domain = p.config.MaskedDomain
				if p.config.SecureCookies {
					cookie.Secure = true
					cookie.SameSite = http.SameSiteNoneMode
				}
			}
		}

		// Add security headers
		resp.Header.Set("X-Content-Type-Options", "nosniff")
		resp.Header.Set("X-Frame-Options", "SAMEORIGIN")
		resp.Header.Set("X-XSS-Protection", "1; mode=block")
		resp.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		return nil
	}

	// Customize error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Warnw("Proxy error", "error", err, "path", r.URL.Path)
		http.Error(w, "Proxy Error", http.StatusBadGateway)
	}

	// Add connection timeout to proxy transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   p.config.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	proxy.Transport = transport

	return proxy
}

// findHandler finds the appropriate handler for the given path
func (p *MaskingProxy) findHandler(path string) http.Handler {
	// Check for path-specific handlers
	for prefix, handler := range p.pathHandlers {
		if strings.HasPrefix(path, prefix) {
			return handler
		}
	}

	// Use default handler as fallback
	return p.defaultHandler
}

// AddRouteMapping adds a new route mapping
func (p *MaskingProxy) AddRouteMapping(pathPrefix string, targetURL *url.URL) {
	p.pathHandlers[pathPrefix] = p.createReverseProxy(targetURL)
}

// RemoveRouteMapping removes a route mapping
func (p *MaskingProxy) RemoveRouteMapping(pathPrefix string) {
	delete(p.pathHandlers, pathPrefix)
}

// SetDefaultTarget sets the default target
func (p *MaskingProxy) SetDefaultTarget(targetURL *url.URL) {
	if targetURL != nil {
		p.defaultHandler = p.createReverseProxy(targetURL)
	} else {
		p.defaultHandler = nil
	}
}

// Helper functions

// getClientIP extracts the client IP from request
func getClientIP(r *http.Request) string {
	// Try various headers
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
		if ip := r.Header.Get(header); ip != "" {
			// X-Forwarded-For can contain multiple IPs, use the first one
			if header == "X-Forwarded-For" {
				ips := strings.Split(ip, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}
			return ip
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// setClientIP modifies the request to use the given IP
func setClientIP(r *http.Request, ip string) *http.Request {
	// Create a clone of the request
	clone := r.Clone(context.Background())

	// Update RemoteAddr
	clone.RemoteAddr = ip

	// Return the modified request
	return clone
}

// sanitizeURLParams removes potentially sensitive info from URL
func sanitizeURLParams(r *http.Request) {
	// List of potentially sensitive parameter names (case-insensitive)
	sensitiveParams := []string{
		"token", "auth", "key", "password", "secret", "email", "phone", "ssn",
		"creditcard", "credit_card", "cc", "dob", "birthdate", "birth_date",
	}

	// Get existing query values
	q := r.URL.Query()

	// Check each parameter
	for _, param := range sensitiveParams {
		if q.Has(param) {
			q.Set(param, "[REDACTED]")
		}
	}

	// Update the URL
	r.URL.RawQuery = q.Encode()
}
