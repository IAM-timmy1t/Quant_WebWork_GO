package auth

import (
	"net/http"
	"strings"
	"sync"
)

// SecurityLock manages authentication attempts and violations
type SecurityLock struct {
	violations     map[string]int
	maxViolations int
	mu            sync.RWMutex
}

// NewSecurityLock creates a new security lock instance
func NewSecurityLock(maxViolations int) *SecurityLock {
	return &SecurityLock{
		violations:     make(map[string]int),
		maxViolations: maxViolations,
	}
}

// AddViolation records an authentication violation
func (s *SecurityLock) AddViolation(remoteAddr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.violations[remoteAddr]++
	return s.violations[remoteAddr] >= s.maxViolations
}

// Master2FAMiddleware validates 2FA tokens for master admin access
func Master2FAMiddleware(totpManager *TOTPManager, securityLock *SecurityLock) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			totpCode := r.Header.Get("X-2FA-Code")
			if totpCode == "" {
				http.Error(w, "2FA code required", http.StatusUnauthorized)
				return
			}

			// Get master secret from secure storage (implement this)
			masterSecret := "MASTER_SECRET" // TODO: Implement secure storage

			if !totpManager.ValidateCode(masterSecret, totpCode) {
				if securityLock.AddViolation(r.RemoteAddr) {
					http.Error(w, "Too many failed attempts", http.StatusTooManyRequests)
					return
				}
				http.Error(w, "Invalid 2FA code", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// JWTAuthMiddleware validates JWT tokens for regular API access
func JWTAuthMiddleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Extract bearer token
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}

			claims, err := jwtManager.ValidateToken(tokenParts[1])
			if err != nil {
				if err == ErrExpiredToken {
					http.Error(w, "Token has expired", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add claims to request context
			ctx := ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoleMiddleware checks if the user has the required role
func RoleMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if claims.Role != requiredRole {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
