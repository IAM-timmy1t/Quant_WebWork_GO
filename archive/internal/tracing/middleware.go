package tracing

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware adds distributed tracing to HTTP requests
func (t *Tracer) TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !t.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Start a new span for this request
		ctx, span := t.StartSpan(r.Context(), fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path))
		defer span.End()

		// Add basic request information
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.UserAgent()),
		)

		// Add headers as attributes (excluding sensitive ones)
		for name, values := range r.Header {
			if !isSensitiveHeader(name) {
				span.SetAttributes(attribute.String(
					fmt.Sprintf("http.header.%s", strings.ToLower(name)),
					strings.Join(values, ","),
				))
			}
		}

		// Create wrapped response writer to capture status code
		wrapper := &responseWrapper{
			ResponseWriter: w,
			statusCode:    http.StatusOK,
		}

		// Add request timing
		startTime := time.Now()

		// Call next handler with tracing context
		next.ServeHTTP(wrapper, r.WithContext(ctx))

		// Add response information
		duration := time.Since(startTime)
		span.SetAttributes(
			attribute.Int("http.status_code", wrapper.statusCode),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		// Record error if status code indicates one
		if wrapper.statusCode >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
			if wrapper.statusCode >= 500 {
				span.SetStatus(trace.StatusCodeError, fmt.Sprintf("HTTP %d", wrapper.statusCode))
			}
		}
	})
}

// responseWrapper wraps http.ResponseWriter to capture the status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// isSensitiveHeader checks if a header name is sensitive
func isSensitiveHeader(name string) bool {
	name = strings.ToLower(name)
	sensitiveHeaders := map[string]bool{
		"authorization":   true,
		"x-api-key":      true,
		"cookie":         true,
		"set-cookie":     true,
		"x-csrf-token":   true,
		"x-auth-token":   true,
		"x-session-id":   true,
		"x-access-token": true,
	}
	return sensitiveHeaders[name]
}
