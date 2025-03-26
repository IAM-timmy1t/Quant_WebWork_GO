// middleware.go - HTTP middleware for IP masking

package ipmasking

import (
	"net"
	"net/http"
	"strings"
)

// HTTPIPMaskingMiddleware is a middleware that masks client IP addresses
type HTTPIPMaskingMiddleware struct {
	masker IPMasker
	// Optional header to set with original IP (for debugging)
	preserveOriginalHeader string
	// List of trusted proxies
	trustedProxies []string
}

// NewHTTPIPMaskingMiddleware creates a new HTTP middleware for IP masking
func NewHTTPIPMaskingMiddleware(masker IPMasker) *HTTPIPMaskingMiddleware {
	return &HTTPIPMaskingMiddleware{
		masker:                 masker,
		preserveOriginalHeader: "",
		trustedProxies:         []string{},
	}
}

// WithOriginalIPHeader configures the middleware to preserve the original IP in a header
func (m *HTTPIPMaskingMiddleware) WithOriginalIPHeader(headerName string) *HTTPIPMaskingMiddleware {
	m.preserveOriginalHeader = headerName
	return m
}

// WithTrustedProxies configures trusted proxies for IP extraction
func (m *HTTPIPMaskingMiddleware) WithTrustedProxies(proxies []string) *HTTPIPMaskingMiddleware {
	m.trustedProxies = proxies
	return m
}

// Middleware returns an http.Handler middleware function
func (m *HTTPIPMaskingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If IP masking is not enabled, just pass through
		if !m.masker.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Extract client IP
		originalIP := extractClientIP(r, m.trustedProxies)
		if originalIP == nil {
			// Could not determine client IP, just pass through
			next.ServeHTTP(w, r)
			return
		}

		// Mask the IP
		maskedIP := m.masker.GetMaskedIP(originalIP)

		// Create a new context with the masked IP
		// Note: This doesn't actually change the RemoteAddr in the request
		// In a real implementation, you might use a custom context key to store the masked IP
		// and implement more sophisticated request manipulation

		// For demonstration purposes, we'll preserve the original IP in a header if configured
		if m.preserveOriginalHeader != "" {
			r.Header.Set(m.preserveOriginalHeader, originalIP.String())
		}

		// Set X-Forwarded-For with the masked IP
		// In a real implementation, this manipulation would need to be more carefully considered
		r.Header.Set("X-Forwarded-For", maskedIP.String())

		// Call the next handler with the modified request
		next.ServeHTTP(w, r)
	})
}

// extractClientIP extracts the real client IP from a request, considering proxies
func extractClientIP(r *http.Request, trustedProxies []string) net.IP {
	// Try X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For format: client, proxy1, proxy2, ...
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			// Get the leftmost IP (client)
			clientIP := strings.TrimSpace(ips[0])
			if ip := net.ParseIP(clientIP); ip != nil {
				return ip
			}
		}
	}

	// Try X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		if ip := net.ParseIP(strings.TrimSpace(xrip)); ip != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr
	if r.RemoteAddr != "" {
		// RemoteAddr is in the format "IP:port"
		ipStr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			if ip := net.ParseIP(ipStr); ip != nil {
				return ip
			}
		} else {
			// Maybe RemoteAddr is just an IP without port
			if ip := net.ParseIP(r.RemoteAddr); ip != nil {
				return ip
			}
		}
	}

	return nil
}
