// router.go - REST API router and endpoint definitions

package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// APIRouter handles REST API routing and endpoint registration
type APIRouter struct {
	router        *mux.Router
	middlewares   []MiddlewareFunc
	routes        []*Route
	errorHandler  ErrorHandler
	metrics       MetricsCollector
	tokenManager  TokenManager
	defaultCORS   *CORSOptions
	basePath      string
	documentation APIDocumentation
	config        *RouterConfig
}

// Route defines a REST API route
type Route struct {
	Path        string           // URL path pattern
	Method      string           // HTTP method
	Handler     http.HandlerFunc // Handler function
	Middlewares []MiddlewareFunc // Route-specific middlewares
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
	Limit    int           // Maximum requests
	Window   time.Duration // Time window for rate limit
	PerIP    bool          // Whether limit is per IP
	KeyFunc  RateLimitKeyFunc // Function to extract key for rate limiting
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
	Title          string                  // API title
	Description    string                  // API description
	Version        string                  // API version
	Contact        map[string]string       // Contact information
	TermsOfService string                  // Terms of service URL
	License        map[string]string       // License information
	Servers        []map[string]string     // Servers information
	ExternalDocs   map[string]string       // External documentation
	Tags           []map[string]string     // API tags
	SecurityDefs   map[string]interface{}  // Security definitions
	Components     map[string]interface{}  // Reusable components
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

// TokenManager handles API token validation
type TokenManager interface {
	ValidateToken(token string) (map[string]interface{}, error)
	GenerateToken(claims map[string]interface{}, expiresIn time.Duration) (string, error)
	RevokeToken(token string) error
}

// MiddlewareFunc defines middleware function signature
type MiddlewareFunc func(http.Handler) http.Handler

// NewRouter creates a new API router
func NewRouter(config *RouterConfig) *APIRouter {
	if config == nil {
		config = defaultRouterConfig()
	}
	
	r := &APIRouter{
		router:      mux.NewRouter(),
		middlewares: []MiddlewareFunc{},
		routes:      []*Route{},
		config:      config,
		basePath:    config.BasePath,
	}
	
	// Set default error handler
	r.errorHandler = defaultErrorHandler
	
	// Configure default CORS if enabled
	if config.EnableCORS {
		if config.CORSOptions != nil {
			r.defaultCORS = config.CORSOptions
		} else {
			r.defaultCORS = defaultCORSOptions()
		}
	}
	
	// Apply base path if specified
	if config.BasePath != "" {
		r.router = r.router.PathPrefix(config.BasePath).Subrouter()
	}
	
	return r
}

// defaultRouterConfig returns default router configuration
func defaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		EnableCORS:        true,
		EnableCompression: true,
		EnableLogging:     true,
		LogRequests:       true,
		Timeout:           30 * time.Second,
		MaxRequestSize:    10 * 1024 * 1024, // 10 MB
		EnableMetrics:     true,
		EnableDocs:        true,
		ValidateRequests:  true,
		ValidateResponses: false,
		APIKeyHeader:      "X-API-Key",
		JWTHeader:         "Authorization",
	}
}

// defaultCORSOptions returns default CORS options
func defaultCORSOptions() *CORSOptions {
	return &CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// defaultErrorHandler provides a default error handling implementation
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error, status int) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"status":  status,
			"message": err.Error(),
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"path":      r.URL.Path,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If JSON encoding fails, fall back to plain text
		http.Error(w, fmt.Sprintf("Error: %s", err.Error()), status)
	}
}

// SetErrorHandler sets a custom error handler
func (r *APIRouter) SetErrorHandler(handler ErrorHandler) {
	r.errorHandler = handler
}

// SetMetricsCollector sets a metrics collector
func (r *APIRouter) SetMetricsCollector(metrics MetricsCollector) {
	r.metrics = metrics
}

// SetTokenManager sets a token manager
func (r *APIRouter) SetTokenManager(tokenManager TokenManager) {
	r.tokenManager = tokenManager
}

// UseMiddleware adds a global middleware to the router
func (r *APIRouter) UseMiddleware(middleware MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middleware)
}

// RegisterRoute registers a new API route
func (r *APIRouter) RegisterRoute(route *Route) error {
	if route.Path == "" || route.Method == "" || route.Handler == nil {
		return fmt.Errorf("invalid route: path, method, and handler are required")
	}
	
	// Add to routes list for documentation
	r.routes = append(r.routes, route)
	
	// Create handler chain with middlewares
	var handler http.Handler = route.Handler
	
	// Apply route-specific middlewares in reverse order
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}
	
	// Apply global middlewares in reverse order
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}
	
	// Register the route
	r.router.
		Methods(route.Method, "OPTIONS").
		Path(route.Path).
		Handler(handler)
	
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
	if r.documentation.Title == "" {
		r.documentation.Title = "API Documentation"
	}
	
	if r.documentation.Version == "" {
		r.documentation.Version = "1.0.0"
	}
	
	return r.documentation
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
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
	
	// Default error handler
	defaultErrorHandler(w, r, err, statusCode)
}

// ExtractBearerToken extracts a Bearer token from the Authorization header
func ExtractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}
	
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("authorization header format must be Bearer {token}")
	}
	
	return parts[1], nil
}

// GetIPAddress extracts the client IP address from a request
func GetIPAddress(r *http.Request, trustedProxies []string) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs (client, proxy1, proxy2...)
		ips := strings.Split(xff, ",")
		
		// If we have trusted proxies, we need to check them
		if len(trustedProxies) > 0 {
			// Start from the rightmost IP and move left
			for i := len(ips) - 1; i >= 0; i-- {
				ip := strings.TrimSpace(ips[i])
				
				// Check if this IP is a trusted proxy
				isTrustedProxy := false
				for _, trusted := range trustedProxies {
					if ip == trusted {
						isTrustedProxy = true
						break
					}
				}
				
				// If not a trusted proxy, this is the client IP
				if !isTrustedProxy {
					return ip
				}
			}
		}
		
		// If no trusted proxies or all IPs are trusted, return leftmost (client)
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	
	return ip
}
