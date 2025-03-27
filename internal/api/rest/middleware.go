// middleware.go - Middleware components for API layer

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Common middleware constants
const (
	RequestIDHeader    = "X-Request-ID"
	RequestTimeHeader  = "X-Request-Time"
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
	ContextRequestLogger ContextKey = "request_logger"
)

// MiddlewareFunc defines middleware function signature
type MiddlewareFunc func(http.Handler) http.Handler

// StandardMiddleware contains commonly used middleware
type StandardMiddleware struct {
	logger            Logger
	metrics           MetricsCollector
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
	rateLimiter RateLimiter,
) *StandardMiddleware {
	return &StandardMiddleware{
		logger:            logger,
		metrics:           metrics,
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
			// Check if request ID is already present in header
			requestID := r.Header.Get(RequestIDHeader)
			if requestID == "" {
				// Generate a unique ID for this request
				requestID = uuid.New().String()
			}

			// Add the ID to the request context
			ctx := context.WithValue(r.Context(), ContextRequestID, requestID)

			// Add to headers
			w.Header().Set(RequestIDHeader, requestID)

			// Add request start time
			startTime := time.Now()
			ctx = context.WithValue(ctx, ContextStartTime, startTime)
			w.Header().Set(RequestTimeHeader, startTime.Format(time.RFC3339))

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestLogger adds logging for each request
func (m *StandardMiddleware) RequestLogger() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract request ID from context
			requestID, _ := r.Context().Value(ContextRequestID).(string)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Create logger with request-specific fields
			fields := map[string]interface{}{
				"request_id": requestID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"client_ip":  GetIPAddress(r, m.trustedProxies),
				"user_agent": r.UserAgent(),
			}

			// Store logger in context for request-specific logging
			ctx := context.WithValue(r.Context(), ContextRequestLogger, fields)

			// Log request start
			m.logger.Info("[CSM-INFO] Request started", fields)

			// Wrap response writer to capture status code
			rw := newResponseWriter(w)

			// Record start time
			startTime := time.Now()

			// Process request with wrapped writer
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Calculate duration
			duration := time.Since(startTime)

			// Add response information
			fields["status"] = rw.statusCode
			fields["duration_ms"] = duration.Milliseconds()
			fields["size_bytes"] = rw.size

			// Log based on status code
			if rw.statusCode >= 500 {
				m.logger.Error("[CSM-ERR] Request failed", fields)
			} else if rw.statusCode >= 400 {
				m.logger.Warn("[CSM-WARN] Request error", fields)
			} else {
				m.logger.Info("[CSM-INFO] Request completed", fields)
			}
		})
	}
}

// Timeout adds a timeout to each request
func (m *StandardMiddleware) Timeout() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), m.timeoutDuration)
			defer cancel()

			// Create a done channel to notify when request is complete
			done := make(chan struct{})

			// Create a response writer wrapper
			rw := newResponseWriter(w)

			// Process the request in a goroutine
			go func() {
				next.ServeHTTP(rw, r.WithContext(ctx))
				close(done)
			}()

			// Wait for either the request to complete or the timeout to occur
			select {
			case <-done:
				// Request completed successfully
				return
			case <-ctx.Done():
				// Timeout occurred
				if ctx.Err() == context.DeadlineExceeded {
					m.logger.Warn("[CSM-WARN] Request timeout exceeded", map[string]interface{}{
						"path":        r.URL.Path,
						"method":      r.Method,
						"timeout_sec": m.timeoutDuration.Seconds(),
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusGatewayTimeout)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "[CSM-ERR-504] Request timeout exceeded",
						"message": fmt.Sprintf("The request exceeded the %v timeout", m.timeoutDuration),
					})
				}
				return
			}
		})
	}
}

// ClientIP adds the client IP to the request context
func (m *StandardMiddleware) ClientIP() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := GetIPAddress(r, m.trustedProxies)
			ctx := context.WithValue(r.Context(), ContextClientIP, clientIP)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimit applies rate limiting to requests
func (m *StandardMiddleware) RateLimit(limit int, window time.Duration) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting if no rate limiter is configured
			if m.rateLimiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Extract client IP for rate limit key
			clientIP := GetIPAddress(r, m.trustedProxies)

			// Create a rate limit key from the client IP and path
			key := fmt.Sprintf("%s:%s", clientIP, r.URL.Path)

			// Check if the request is allowed
			remaining, reset, allowed := m.rateLimiter.Allow(key, limit, window)

			// Set rate limit headers
			w.Header().Set(RateLimitLimit, fmt.Sprintf("%d", limit))
			w.Header().Set(RateLimitRemaining, fmt.Sprintf("%d", remaining))
			w.Header().Set(RateLimitReset, fmt.Sprintf("%d", reset))

			// If not allowed, return a rate limit exceeded error
			if !allowed {
				m.logger.Warn("[CSM-WARN] Rate limit exceeded", map[string]interface{}{
					"client_ip": clientIP,
					"path":      r.URL.Path,
					"limit":     limit,
					"window":    window.String(),
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "[CSM-ERR-429] Rate limit exceeded",
					"message": fmt.Sprintf("You have exceeded the rate limit of %d requests per %v", limit, window),
				})
				return
			}

			// Continue to the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeLimit limits the size of request bodies
func (m *StandardMiddleware) RequestSizeLimit() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to requests with a body
			if r.ContentLength > 0 {
				if r.ContentLength > m.requestSizeLimit {
					m.logger.Warn("[CSM-WARN] Request size limit exceeded", map[string]interface{}{
						"content_length": r.ContentLength,
						"limit":          m.requestSizeLimit,
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusRequestEntityTooLarge)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "[CSM-ERR-413] Request entity too large",
						"message": fmt.Sprintf("The request body exceeds the %d byte limit", m.requestSizeLimit),
					})
					return
				}

				// Limit the request body size
				r.Body = http.MaxBytesReader(w, r.Body, m.requestSizeLimit)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateContentType validates the Content-Type header
func (m *StandardMiddleware) ValidateContentType(contentType string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check for requests with bodies
			if r.ContentLength > 0 {
				// Extract content type and compare
				requestContentType := r.Header.Get("Content-Type")
				if !strings.HasPrefix(requestContentType, contentType) {
					m.logger.Warn("[CSM-WARN] Invalid content type", map[string]interface{}{
						"received": requestContentType,
						"expected": contentType,
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnsupportedMediaType)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "[CSM-ERR-415] Unsupported media type",
						"message": fmt.Sprintf("Expected Content-Type: %s", contentType),
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateRequest validates request bodies against a schema
func (m *StandardMiddleware) ValidateRequest(schema interface{}) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate for requests with bodies
			if r.ContentLength > 0 && schema != nil {
				// Decode request body
				var requestData map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
					m.logger.Warn("[CSM-WARN] Request validation failed", map[string]interface{}{
						"error": err.Error(),
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "[CSM-ERR-400] Invalid request format",
						"message": "Could not parse request body as valid JSON",
					})
					return
				}

				// Reset the body for the next handler
				// TODO: Implement proper request validation
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS handles Cross-Origin Resource Sharing
func (m *StandardMiddleware) CORS(options *CORSOptions) MiddlewareFunc {
	if options == nil {
		options = &CORSOptions{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
			ExposedHeaders:   []string{"X-Total-Count", "X-Request-ID"},
			AllowCredentials: true,
			MaxAge:           86400,
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set allowed origins
			origin := r.Header.Get("Origin")
			if origin != "" {
				if len(options.AllowedOrigins) == 1 && options.AllowedOrigins[0] == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					for _, allowedOrigin := range options.AllowedOrigins {
						if allowedOrigin == origin {
							w.Header().Set("Access-Control-Allow-Origin", origin)
							break
						}
					}
				}
			}

			// Set other CORS headers
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(options.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(options.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(options.ExposedHeaders, ", "))

			// Set credentials and max age
			if options.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", options.MaxAge))

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Continue to the next handler
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
					// Log the panic
					m.logger.Error("[CSM-ERR] Request handler panic", map[string]interface{}{
						"error":  fmt.Sprintf("%v", err),
						"path":   r.URL.Path,
						"method": r.Method,
					})

					// Return a 500 error
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "[CSM-ERR-500] Internal server error",
						"message": "The server encountered an error while processing your request",
					})
				}
			}()

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
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += int64(size)
	return size, err
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
		// Extract request logger from context
		logFields, _ := r.Context().Value(ContextRequestLogger).(map[string]interface{})
		if logFields == nil {
			logFields = map[string]interface{}{}
		}

		// Add error information
		logFields["error"] = err.Error()
		logFields["status"] = status

		// Log based on status code
		if status >= 500 {
			m.logger.Error("[CSM-ERR] Request error", logFields)
		} else if status >= 400 {
			m.logger.Warn("[CSM-WARN] Request error", logFields)
		} else {
			m.logger.Info("[CSM-INFO] Request notice", logFields)
		}

		// Prepare error response
		errorResponse := map[string]interface{}{
			"error": map[string]interface{}{
				"status":    status,
				"code":      fmt.Sprintf("[CSM-ERR-%d]", status),
				"message":   err.Error(),
				"timestamp": time.Now().Format(time.RFC3339),
				"path":      r.URL.Path,
				"method":    r.Method,
			},
		}

		// Send error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(errorResponse)
	}
}

// LoggingMiddleware creates a middleware that logs requests
func LoggingMiddleware(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			ww := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(ww, r)

			// Calculate duration
			duration := time.Since(start)

			// Log request details
			logger.Infow("HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.statusCode,
				"duration", duration,
				"size", ww.size,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// MetricsMiddleware creates a middleware that records request metrics
func MetricsMiddleware(metricsCollector *metrics.Collector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			ww := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(ww, r)

			// Calculate duration
			duration := time.Since(start)

			// Record metrics
			metricsCollector.RecordHTTPRequest(r.Method, r.URL.Path, ww.statusCode, duration.Seconds())
		})
	}
}

// RateLimitMiddleware creates a middleware that limits request rates
func RateLimitMiddleware(limit int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := make(map[string]*time.Ticker)

	// Clean up old limiters periodically
	go func() {
		for {
			time.Sleep(time.Minute * 5)
			mu.Lock()
			for ip, limiter := range limiters {
				limiter.Stop()
				delete(limiters, ip)
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP for rate limiting
			ip := r.RemoteAddr
			if strings.Contains(ip, ":") {
				ip = strings.Split(ip, ":")[0]
			}

			// Check if IP is rate limited
			mu.Lock()
			if _, exists := limiters[ip]; !exists {
				// Create a new rate limiter for this IP
				limiters[ip] = time.NewTicker(time.Second / time.Duration(limit))
				// Start a goroutine to clean up this ticker after 1 hour
				go func(key string, ticker *time.Ticker) {
					time.Sleep(time.Hour)
					mu.Lock()
					delete(limiters, key)
					ticker.Stop()
					mu.Unlock()
				}(ip, limiters[ip])
			}

			// Get the ticker for this IP
			ticker := limiters[ip]
			mu.Unlock()

			// Try to consume a token from the rate limiter
			select {
			case <-ticker.C:
				// Token available, process request
				next.ServeHTTP(w, r)
			default:
				// No token available, rate limit exceeded
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Rate limit exceeded",
					"message": "Too many requests, please try again later",
				})
			}
		})
	}
}
