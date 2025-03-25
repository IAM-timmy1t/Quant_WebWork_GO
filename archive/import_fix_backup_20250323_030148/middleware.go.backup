// middleware.go - Middleware components for API layer

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Common middleware constants
const (
	RequestIDHeader   = "X-Request-ID"
	RequestTimeHeader = "X-Request-Time"
	RateLimitRemaining = "X-RateLimit-Remaining"
	RateLimitReset     = "X-RateLimit-Reset"
	RateLimitLimit     = "X-RateLimit-Limit"
)

// ContextKey is a type for context keys
type ContextKey string

// Context keys
const (
	ContextRequestID     ContextKey = "request_id"
	ContextStartTime     ContextKey = "start_time"
	ContextUserID        ContextKey = "user_id"
	ContextClientIP      ContextKey = "client_ip"
	ContextTokenClaims   ContextKey = "token_claims"
	ContextRequestLogger ContextKey = "request_logger"
)

// StandardMiddleware contains commonly used middleware
type StandardMiddleware struct {
	logger            Logger
	metrics           MetricsCollector
	tokenManager      TokenManager
	rateLimiter       RateLimiter
	trustedProxies    []string
	requestSizeLimit  int64
	timeoutDuration   time.Duration
	compressResponses bool
}

// Logger interface for middleware logging
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// NewStandardMiddleware creates a new middleware provider
func NewStandardMiddleware(
	logger Logger,
	metrics MetricsCollector,
	tokenManager TokenManager,
	rateLimiter RateLimiter,
) *StandardMiddleware {
	return &StandardMiddleware{
		logger:            logger,
		metrics:           metrics,
		tokenManager:      tokenManager,
		rateLimiter:       rateLimiter,
		trustedProxies:    []string{},
		requestSizeLimit:  10 * 1024 * 1024, // 10MB default
		timeoutDuration:   30 * time.Second,
		compressResponses: true,
	}
}

// SetTrustedProxies sets the list of trusted proxies for IP resolution
func (m *StandardMiddleware) SetTrustedProxies(proxies []string) {
	m.trustedProxies = proxies
}

// SetRequestSizeLimit sets the maximum request size in bytes
func (m *StandardMiddleware) SetRequestSizeLimit(sizeBytes int64) {
	m.requestSizeLimit = sizeBytes
}

// SetTimeoutDuration sets the request timeout duration
func (m *StandardMiddleware) SetTimeoutDuration(timeout time.Duration) {
	m.timeoutDuration = timeout
}

// SetCompressResponses configures response compression
func (m *StandardMiddleware) SetCompressResponses(compress bool) {
	m.compressResponses = compress
}

// RequestID adds a unique request ID to each request
func (m *StandardMiddleware) RequestID() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request already has an ID
			requestID := r.Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
			}
			
			// Add request ID to response headers
			w.Header().Set(RequestIDHeader, requestID)
			
			// Add request ID to context
			ctx := context.WithValue(r.Context(), ContextRequestID, requestID)
			
			// Call next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestLogger adds logging for each request
func (m *StandardMiddleware) RequestLogger() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add start time to context
			startTime := time.Now()
			ctx := context.WithValue(r.Context(), ContextStartTime, startTime)
			
			// Get request ID from context if available
			requestID, _ := ctx.Value(ContextRequestID).(string)
			if requestID == "" {
				requestID = uuid.New().String()
				ctx = context.WithValue(ctx, ContextRequestID, requestID)
			}
			
			// Log request
			m.logger.Info("API Request", map[string]interface{}{
				"request_id": requestID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"query":      r.URL.RawQuery,
				"remoteAddr": r.RemoteAddr,
				"userAgent":  r.UserAgent(),
			})
			
			// Create response wrapper to capture status code
			rw := newResponseWriter(w)
			
			// Process request
			next.ServeHTTP(rw, r.WithContext(ctx))
			
			// Calculate duration
			duration := time.Since(startTime)
			
			// Log response
			m.logger.Info("API Response", map[string]interface{}{
				"request_id": requestID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     rw.statusCode,
				"duration":   duration.String(),
				"size":       rw.size,
			})
			
			// Report metrics if metrics collector is available
			if m.metrics != nil {
				m.metrics.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration)
				m.metrics.RecordResponseSize(r.Method, r.URL.Path, rw.size)
			}
		})
	}
}

// Timeout adds a timeout to each request
func (m *StandardMiddleware) Timeout() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), m.timeoutDuration)
			defer cancel()
			
			// Create a channel to indicate when the request is done
			done := make(chan struct{})
			
			// Process request in a goroutine
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()
			
			// Wait for the request to complete or timeout
			select {
			case <-done:
				// Request completed normally
				return
			case <-ctx.Done():
				// Request timed out
				m.logger.Warn("Request timeout", map[string]interface{}{
					"request_id": ctx.Value(ContextRequestID),
					"method":     r.Method,
					"path":       r.URL.Path,
					"timeout":    m.timeoutDuration.String(),
				})
				
				// Return timeout error
				http.Error(w, "Request timeout", http.StatusGatewayTimeout)
				return
			}
		})
	}
}

// ClientIP adds the client IP to the request context
func (m *StandardMiddleware) ClientIP() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			ip := GetIPAddress(r, m.trustedProxies)
			
			// Add IP to context
			ctx := context.WithValue(r.Context(), ContextClientIP, ip)
			
			// Call next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Authenticate validates JWT tokens
func (m *StandardMiddleware) Authenticate(optional bool) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			token, err := ExtractBearerToken(r)
			
			if err != nil {
				if optional {
					// If authentication is optional, continue without token
					next.ServeHTTP(w, r)
					return
				}
				
				// Return authentication error
				w.Header().Set("WWW-Authenticate", "Bearer")
				Error(w, r, fmt.Errorf("authentication required: %w", err), http.StatusUnauthorized, m.errorHandler(r))
				return
			}
			
			// Validate token
			claims, err := m.tokenManager.ValidateToken(token)
			if err != nil {
				if optional {
					// If authentication is optional, continue without token
					next.ServeHTTP(w, r)
					return
				}
				
				// Return authentication error
				w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
				Error(w, r, fmt.Errorf("invalid token: %w", err), http.StatusUnauthorized, m.errorHandler(r))
				return
			}
			
			// Add claims to context
			ctx := context.WithValue(r.Context(), ContextTokenClaims, claims)
			
			// Add user ID to context if available
			if userID, ok := claims["sub"].(string); ok {
				ctx = context.WithValue(ctx, ContextUserID, userID)
			}
			
			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimit applies rate limiting to requests
func (m *StandardMiddleware) RateLimit(limit int, window time.Duration) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting if no rate limiter is available
			if m.rateLimiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			
			// Get client IP for rate limiting
			clientIP, ok := r.Context().Value(ContextClientIP).(string)
			if !ok {
				clientIP = GetIPAddress(r, m.trustedProxies)
			}
			
			// Create rate limit key
			key := fmt.Sprintf("%s:%s:%s", clientIP, r.Method, r.URL.Path)
			
			// Check rate limit
			remaining, reset, allowed := m.rateLimiter.Allow(key, limit, window)
			
			// Set rate limit headers
			w.Header().Set(RateLimitLimit, fmt.Sprintf("%d", limit))
			w.Header().Set(RateLimitRemaining, fmt.Sprintf("%d", remaining))
			w.Header().Set(RateLimitReset, fmt.Sprintf("%d", reset))
			
			if !allowed {
				// Return rate limit error
				Error(w, r, fmt.Errorf("rate limit exceeded"), http.StatusTooManyRequests, m.errorHandler(r))
				
				// Report metrics if metrics collector is available
				if m.metrics != nil {
					m.metrics.RecordRateLimit(key, false)
				}
				
				return
			}
			
			// Report metrics if metrics collector is available
			if m.metrics != nil {
				m.metrics.RecordRateLimit(key, true)
			}
			
			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeLimit limits the size of request bodies
func (m *StandardMiddleware) RequestSizeLimit() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if no body
			if r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			
			// Replace body with limited reader
			r.Body = http.MaxBytesReader(w, r.Body, m.requestSizeLimit)
			
			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateContentType validates the Content-Type header
func (m *StandardMiddleware) ValidateContentType(contentType string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for methods that don't typically have a body
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Check Content-Type header
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, contentType) {
				Error(w, r, fmt.Errorf("unsupported content type: %s", ct), http.StatusUnsupportedMediaType, m.errorHandler(r))
				return
			}
			
			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateRequest validates request bodies against a schema
func (m *StandardMiddleware) ValidateRequest(schema interface{}) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for methods that don't typically have a body
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Parse request body
			if err := json.NewDecoder(r.Body).Decode(schema); err != nil {
				Error(w, r, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest, m.errorHandler(r))
				return
			}
			
			// Reset body for downstream handlers
			r.Body.Close()
			
			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// CORS handles Cross-Origin Resource Sharing
func (m *StandardMiddleware) CORS(options *CORSOptions) MiddlewareFunc {
	if options == nil {
		options = defaultCORSOptions()
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			
			// Allow origins
			origin := r.Header.Get("Origin")
			if origin != "" {
				// Check if origin is allowed
				allowed := false
				for _, allowedOrigin := range options.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						allowed = true
						break
					}
				}
				
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
			
			// Allow credentials
			if options.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			
			// Expose headers
			if len(options.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(options.ExposedHeaders, ", "))
			}
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				// Allow methods
				if len(options.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(options.AllowedMethods, ", "))
				}
				
				// Allow headers
				if len(options.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(options.AllowedHeaders, ", "))
				}
				
				// Max age
				if options.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", options.MaxAge))
				}
				
				// Return OK for preflight requests
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			// Call next handler for non-preflight requests
			next.ServeHTTP(w, r)
		})
	}
}

// Recovery handles panics in request handling
func (m *StandardMiddleware) Recovery() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log error
					m.logger.Error("Request panic", map[string]interface{}{
						"error":      fmt.Sprintf("%v", err),
						"request_id": r.Context().Value(ContextRequestID),
						"method":     r.Method,
						"path":       r.URL.Path,
					})
					
					// Return server error
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}()
			
			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(key string, limit int, window time.Duration) (remaining int, reset int64, allowed bool)
}

// responseWriter is a wrapper for http.ResponseWriter that captures status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

// newResponseWriter creates a new response writer wrapper
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// Flush implements http.Flusher if the underlying response writer supports it
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// errorHandler returns an error handler for a request
func (m *StandardMiddleware) errorHandler(r *http.Request) ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request, err error, status int) {
		// Get request ID from context
		requestID, _ := r.Context().Value(ContextRequestID).(string)
		
		// Create error response
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"status":  status,
				"message": err.Error(),
				"request_id": requestID,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"path":      r.URL.Path,
		}
		
		// Log error
		m.logger.Error("API Error", map[string]interface{}{
			"request_id": requestID,
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     status,
			"error":      err.Error(),
		})
		
		// Return error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
	}
}
