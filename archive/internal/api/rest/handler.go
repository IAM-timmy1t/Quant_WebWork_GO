package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/auth"
)

// Handler represents the REST API handler
type Handler struct {
	router       *chi.Mux
	authManager  *auth.JWTManager
	totpManager  *auth.TOTPManager
	securityLock *auth.SecurityLock
}

// NewHandler creates a new REST API handler
func NewHandler(
	authManager *auth.JWTManager,
	totpManager *auth.TOTPManager,
	securityLock *auth.SecurityLock,
) *Handler {
	h := &Handler{
		router:       chi.NewRouter(),
		authManager:  authManager,
		totpManager:  totpManager,
		securityLock: securityLock,
	}

	h.setupRoutes()
	return h
}

// setupRoutes configures the API routes
func (h *Handler) setupRoutes() {
	// Middleware
	h.router.Use(middleware.RequestID)
	h.router.Use(middleware.RealIP)
	h.router.Use(middleware.Logger)
	h.router.Use(middleware.Recoverer)
	h.router.Use(middleware.Timeout(60 * time.Second))

	// Public routes
	h.router.Group(func(r chi.Router) {
		r.Post("/auth/login", h.handleLogin)
		r.Post("/auth/refresh", h.handleRefreshToken)
	})

	// Protected routes (require JWT)
	h.router.Group(func(r chi.Router) {
		r.Use(auth.JWTAuthMiddleware(h.authManager))
		
		r.Get("/apps", h.handleListApps)
		r.Post("/apps", h.handleRegisterApp)
		r.Get("/apps/{appID}", h.handleGetApp)
		r.Put("/apps/{appID}", h.handleUpdateApp)
		r.Delete("/apps/{appID}", h.handleDeleteApp)
	})

	// Master admin routes (require 2FA)
	h.router.Group(func(r chi.Router) {
		r.Use(auth.JWTAuthMiddleware(h.authManager))
		r.Use(auth.Master2FAMiddleware(h.totpManager, h.securityLock))
		r.Use(auth.RoleMiddleware("master"))

		r.Post("/admin/generate-2fa", h.handleGenerate2FA)
		r.Post("/admin/reset-security", h.handleResetSecurity)
	})
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

