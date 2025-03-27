// router.go - REST API router and endpoint definitions

package rest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// APIRouter handles REST API routing and endpoint registration
type APIRouter struct {
	router        *mux.Router
	routes        []*Route
	errorHandler  ErrorHandler
	metrics       MetricsCollector
	defaultCORS   *CORSOptions
	basePath      string
	documentation APIDocumentation
	config        *RouterConfig
	logger        *zap.SugaredLogger
}

// Route defines a REST API route
type Route struct {
	Path        string           // URL path pattern
	Method      string           // HTTP method
	Handler     http.HandlerFunc // Handler function
	Name        string           // Route name for documentation
	Description string           // Route description for documentation
	Tags        []string         // Tags for documentation and categorization
	Deprecated  bool             // Whether the route is deprecated
	Version     string           // API version for this route
	Schema      RouteSchema      // Request/response schema for validation
	Auth        AuthConfig       // Authentication configuration
	RateLimit   *RateLimit       // Rate limiting configuration
}

// RouteSchema defines schema validation for requests and responses
type RouteSchema struct {
	RequestModel  interface{} // Model for request validation
	ResponseModel interface{} // Model for response validation
}

// AuthConfig defines authentication requirements for a route
type AuthConfig struct {
	Required    bool     // Whether authentication is required
	Roles       []string // Required roles
	Permissions []string // Required permissions
	Scopes      []string // Required OAuth scopes
}

// RateLimit defines rate limiting for a route
type RateLimit struct {
	Limit   int              // Maximum requests
	Window  time.Duration    // Time window for rate limit
	PerIP   bool             // Whether limit is per IP
	KeyFunc RateLimitKeyFunc // Function to extract key for rate limiting
}

// RateLimitKeyFunc extracts a key for rate limiting from a request
type RateLimitKeyFunc func(*http.Request) string

// CORSOptions defines Cross-Origin Resource Sharing options
type CORSOptions struct {
	AllowedOrigins   []string // Allowed origins (e.g., "http://example.com")
	AllowedMethods   []string // Allowed HTTP methods
	AllowedHeaders   []string // Allowed headers
	ExposedHeaders   []string // Headers exposed to the client
	AllowCredentials bool     // Whether credentials are allowed
	MaxAge           int      // How long preflight results can be cached (seconds)
}

// RouterConfig defines configuration for the API router
type RouterConfig struct {
	BasePath          string
	EnableCORS        bool
	CORSOptions       *CORSOptions
	EnableCompression bool
	EnableLogging     bool
	LogRequests       bool
	Timeout           time.Duration
	MaxRequestSize    int64
	TrustedProxies    []string
	APIKeyHeader      string
	JWTHeader         string
	EnableMetrics     bool
	EnableDocs        bool
	ValidateRequests  bool
	ValidateResponses bool
}

// APIDocumentation contains API documentation metadata
type APIDocumentation struct {
	Title          string
	Description    string
	Version        string
	Contact        *Contact
	License        *License
	TermsOfService string
	ExternalDocs   *ExternalDocs
	Security       []SecurityScheme
}

// Contact information for the API
type Contact struct {
	Name  string
	URL   string
	Email string
}

// License information for the API
type License struct {
	Name string
	URL  string
}

// ExternalDocs links to external documentation
type ExternalDocs struct {
	Description string
	URL         string
}

// SecurityScheme defines API security requirements
type SecurityScheme struct {
	Type        string
	Name        string
	In          string
	Description string
}

// ErrorHandler processes API errors
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error, status int)

// MetricsCollector collects API metrics
type MetricsCollector interface {
	RecordRequest(method, path string, status int, duration time.Duration)
	RecordRequestSize(method, path string, size int64)
	RecordResponseSize(method, path string, size int64)
	RecordRateLimit(key string, allowed bool)
}

// NewRouter creates a new API router
func NewRouter(cfg *config.Config, logger *zap.SugaredLogger, metricsCollector *metrics.Collector) *mux.Router {
	// Create main router
	router := mux.NewRouter().StrictSlash(true)

	// Create API subrouter with version prefix
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Register standard middlewares
	router.Use(LoggingMiddleware(logger))

	if metricsCollector != nil {
		router.Use(MetricsMiddleware(metricsCollector))
	}

	// Add rate limiting if enabled
	if cfg.Security.RateLimiting.Enabled {
		router.Use(RateLimitMiddleware(cfg.Security.RateLimiting.DefaultLimit))
	}

	// Register health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "UP"})
	})

	// Register API endpoints
	apiRouter.HandleFunc("/system/status", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"status":    "operational",
			"version":   "1.0.0",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(status)
	}).Methods("GET")

	// Bridge endpoints
	bridgeRouter := apiRouter.PathPrefix("/bridge").Subrouter()
	bridgeRouter.HandleFunc("/status", getBridgeStatus).Methods("GET")
	bridgeRouter.HandleFunc("/create", createBridgeConnection).Methods("POST")
	bridgeRouter.HandleFunc("/list", listBridgeConnections).Methods("GET")
	bridgeRouter.HandleFunc("/{id}", getBridgeConnection).Methods("GET")
	bridgeRouter.HandleFunc("/{id}", deleteBridgeConnection).Methods("DELETE")

	// Security endpoints
	securityRouter := apiRouter.PathPrefix("/security").Subrouter()
	securityRouter.HandleFunc("/config", getSecurityConfig).Methods("GET")
	securityRouter.HandleFunc("/firewall/rules", getFirewallRules).Methods("GET")
	securityRouter.HandleFunc("/firewall/rules", createFirewallRule).Methods("POST")
	securityRouter.HandleFunc("/firewall/rules/{id}", deleteFirewallRule).Methods("DELETE")

	// Monitoring endpoints
	monitoringRouter := apiRouter.PathPrefix("/monitoring").Subrouter()
	monitoringRouter.HandleFunc("/metrics", getSystemMetrics).Methods("GET")
	monitoringRouter.HandleFunc("/alerts", getSystemAlerts).Methods("GET")

	return router
}

// Helper functions for endpoint handlers

func getBridgeStatus(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "operational"})
}

func createBridgeConnection(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	RespondWithJSON(w, http.StatusCreated, map[string]string{"id": "bridge-123", "status": "created"})
}

func listBridgeConnections(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	connections := []map[string]string{
		{"id": "bridge-123", "status": "active"},
		{"id": "bridge-456", "status": "inactive"},
	}
	RespondWithJSON(w, http.StatusOK, connections)
}

func getBridgeConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bridgeID := vars["id"]

	// Implementation pending
	RespondWithJSON(w, http.StatusOK, map[string]string{
		"id":     bridgeID,
		"status": "active",
	})
}

func deleteBridgeConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bridgeID := vars["id"]

	// Implementation pending
	RespondWithJSON(w, http.StatusOK, map[string]string{
		"id":     bridgeID,
		"status": "deleted",
	})
}

func getSecurityConfig(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"firewall_enabled":      true,
		"rate_limiting_enabled": true,
		"ip_masking_enabled":    false,
	})
}

func getFirewallRules(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	rules := []map[string]interface{}{
		{"id": "rule-1", "type": "ip", "action": "allow"},
		{"id": "rule-2", "type": "ip", "action": "block"},
	}
	RespondWithJSON(w, http.StatusOK, rules)
}

func createFirewallRule(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	RespondWithJSON(w, http.StatusCreated, map[string]string{
		"id":     "rule-3",
		"status": "created",
	})
}

func deleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	// Implementation pending
	RespondWithJSON(w, http.StatusOK, map[string]string{
		"id":     ruleID,
		"status": "deleted",
	})
}

func getSystemMetrics(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	metrics := map[string]interface{}{
		"cpu_usage":           25.5,
		"memory_usage":        512,
		"active_connections":  42,
		"requests_per_second": 150.5,
	}
	RespondWithJSON(w, http.StatusOK, metrics)
}

func getSystemAlerts(w http.ResponseWriter, r *http.Request) {
	// Implementation pending
	alerts := []map[string]interface{}{
		{"id": "alert-1", "severity": "warning", "message": "High CPU usage"},
		{"id": "alert-2", "severity": "critical", "message": "Excessive failed login attempts"},
	}
	RespondWithJSON(w, http.StatusOK, alerts)
}

// SetErrorHandler sets a custom error handler
func (r *APIRouter) SetErrorHandler(handler ErrorHandler) {
	r.errorHandler = handler
}

// SetMetricsCollector sets a metrics collector
func (r *APIRouter) SetMetricsCollector(metrics MetricsCollector) {
	r.metrics = metrics
}

// RegisterRoute registers a new API route
func (r *APIRouter) RegisterRoute(route *Route) error {
	// Validate route
	if route.Path == "" {
		return fmt.Errorf("route path cannot be empty")
	}
	if route.Method == "" {
		return fmt.Errorf("route method cannot be empty")
	}
	if route.Handler == nil {
		return fmt.Errorf("route handler cannot be nil")
	}

	// Create a new route on the router
	r.router.HandleFunc(route.Path, route.Handler).Methods(route.Method)

	// Store the route for documentation
	r.routes = append(r.routes, route)

	return nil
}

// RegisterRoutes registers multiple routes at once
func (r *APIRouter) RegisterRoutes(routes []*Route) error {
	for _, route := range routes {
		if err := r.RegisterRoute(route); err != nil {
			return err
		}
	}
	return nil
}

// Handler returns the final http.Handler for the API
func (r *APIRouter) Handler() http.Handler {
	return r.router
}

// GenerateDocumentation generates API documentation
func (r *APIRouter) GenerateDocumentation() APIDocumentation {
	docs := r.documentation

	// Default values if not set
	if docs.Title == "" {
		docs.Title = "API Documentation"
	}
	if docs.Version == "" {
		docs.Version = "1.0.0"
	}

	return docs
}

// SetDocumentation sets API documentation metadata
func (r *APIRouter) SetDocumentation(docs APIDocumentation) {
	r.documentation = docs
}

// GetRoutes returns all registered routes
func (r *APIRouter) GetRoutes() []*Route {
	return r.routes
}

// NotFoundHandler sets a custom handler for 404 Not Found errors
func (r *APIRouter) NotFoundHandler(handler http.Handler) {
	r.router.NotFoundHandler = handler
}

// MethodNotAllowedHandler sets a custom handler for 405 Method Not Allowed errors
func (r *APIRouter) MethodNotAllowedHandler(handler http.Handler) {
	r.router.MethodNotAllowedHandler = handler
}

// WithBasePath sets the base path for all routes
func (r *APIRouter) WithBasePath(path string) *APIRouter {
	r.basePath = path
	r.router = r.router.PathPrefix(path).Subrouter()
	return r
}

// ServeHTTP implements the http.Handler interface
func (r *APIRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.router.ServeHTTP(w, req)
}

// JSON returns a JSON response with the provided status code
func JSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Marshal and write the response
	if data != nil {
		return json.NewEncoder(w).Encode(data)
	}

	return nil
}

// Error returns an error response with the provided status code
func Error(w http.ResponseWriter, r *http.Request, err error, statusCode int, errorHandler ErrorHandler) {
	if errorHandler != nil {
		errorHandler(w, r, err, statusCode)
		return
	}

	// Use default JSON error response if no error handler provided
	JSON(w, statusCode, map[string]string{
		"error": err.Error(),
	})
}

// ExtractBearerToken extracts a Bearer token from the Authorization header
func ExtractBearerToken(r *http.Request) (string, error) {
	// Get the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("no Authorization header found")
	}

	// Check if it's a Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("Authorization header is not a Bearer token")
	}

	// Extract the token
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("Bearer token is empty")
	}

	return token, nil
}

// GetIPAddress extracts the client IP address from a request
func GetIPAddress(r *http.Request, trustedProxies []string) string {
	// Check for X-Forwarded-For header
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// The X-Forwarded-For header can contain multiple IPs in a comma-separated list:
		// client, proxy1, proxy2, ...
		ips := strings.Split(forwardedFor, ",")

		// Get the leftmost IP that isn't a trusted proxy
		clientIP := strings.TrimSpace(ips[0])

		// Verify it's not a trusted proxy
		if !isProxyIP(clientIP, trustedProxies) {
			return clientIP
		}

		// Otherwise, try to find the first non-proxy IP
		for i := 1; i < len(ips); i++ {
			ip := strings.TrimSpace(ips[i])
			if !isProxyIP(ip, trustedProxies) {
				return ip
			}
		}
	}

	// Check for X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If there's no port, just use the whole RemoteAddr
		return r.RemoteAddr
	}

	return ip
}

// isProxyIP checks if an IP is in the list of trusted proxies
func isProxyIP(ip string, trustedProxies []string) bool {
	for _, proxyIP := range trustedProxies {
		if ip == proxyIP {
			return true
		}
	}
	return false
}
