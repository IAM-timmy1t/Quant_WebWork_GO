package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/timot/Quant_WebWork_GO/internal/auth"
	"github.com/timot/Quant_WebWork_GO/internal/discovery"
	"github.com/timot/Quant_WebWork_GO/internal/integration"
	"github.com/timot/Quant_WebWork_GO/internal/proxy"
)

// Handler manages dashboard API endpoints
type Handler struct {
	serviceRegistry *discovery.ServiceRegistry
	proxyManager    *proxy.Manager
	integrator      *integration.ProxyDiscoveryIntegrator
	metrics         *MetricsCollector
	wsHub           *WebSocketHub
	auth            *auth.Authenticator
}

// NewHandler creates a new dashboard handler
func NewHandler(
	serviceRegistry *discovery.ServiceRegistry,
	proxyManager *proxy.Manager,
	integrator *integration.ProxyDiscoveryIntegrator,
	auth *auth.Authenticator,
) *Handler {
	h := &Handler{
		serviceRegistry: serviceRegistry,
		proxyManager:    proxyManager,
		integrator:      integrator,
		metrics:         NewMetricsCollector(),
		wsHub:           NewWebSocketHub(),
		auth:            auth,
	}

	// Start WebSocket hub
	go h.wsHub.Run()

	// Start metrics collection
	go h.metrics.Start()

	return h
}

// RegisterRoutes registers dashboard API routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Require admin authentication for all dashboard routes
	r.Group(func(r chi.Router) {
		r.Use(h.auth.AdminAuthMiddleware)

		r.Route("/api/v1/dashboard", func(r chi.Router) {
			// Overview
			r.Get("/overview", h.handleGetOverview)

			// Services
			r.Get("/services", h.handleGetServices)
			r.Post("/services", h.handleAddService)
			r.Put("/services/{id}", h.handleUpdateService)
			r.Delete("/services/{id}", h.handleDeleteService)

			// Proxy Routes
			r.Get("/routes", h.handleGetRoutes)
			r.Post("/routes", h.handleAddRoute)
			r.Put("/routes/{id}", h.handleUpdateRoute)
			r.Delete("/routes/{id}", h.handleDeleteRoute)

			// Metrics
			r.Get("/metrics", h.handleGetMetrics)
			r.Get("/metrics/service/{id}", h.handleGetServiceMetrics)

			// Logs
			r.Get("/logs", h.handleGetLogs)
			r.Delete("/logs", h.handleClearLogs)

			// Configuration
			r.Get("/config", h.handleGetConfig)
			r.Put("/config/proxy", h.handleUpdateProxyConfig)
			r.Put("/config/metrics", h.handleUpdateMetricsConfig)

			// WebSocket
			r.HandleFunc("/ws", h.handleWebSocket)
		})
	})
}

// Overview handler
func (h *Handler) handleGetOverview(w http.ResponseWriter, r *http.Request) {
	overview := h.getSystemOverview()
	respondJSON(w, overview)
}

// Services handlers
func (h *Handler) handleGetServices(w http.ResponseWriter, r *http.Request) {
	services := h.serviceRegistry.GetAllServices()
	respondJSON(w, services)
}

func (h *Handler) handleAddService(w http.ResponseWriter, r *http.Request) {
	var service discovery.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service data")
		return
	}

	if err := h.serviceRegistry.Register(&service); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.wsHub.BroadcastEvent("service_update", service)
	respondJSON(w, service)
}

func (h *Handler) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var service discovery.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service data")
		return
	}

	service.ID = id
	if err := h.serviceRegistry.Update(&service); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.wsHub.BroadcastEvent("service_update", service)
	respondJSON(w, service)
}

func (h *Handler) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.serviceRegistry.Deregister(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.wsHub.BroadcastEvent("service_delete", map[string]string{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

// Proxy route handlers
func (h *Handler) handleGetRoutes(w http.ResponseWriter, r *http.Request) {
	routes := h.proxyManager.GetRoutes()
	respondJSON(w, routes)
}

func (h *Handler) handleAddRoute(w http.ResponseWriter, r *http.Request) {
	var route proxy.ProxyRoute
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid route data")
		return
	}

	if err := h.proxyManager.AddRoute(&route); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.wsHub.BroadcastEvent("route_update", route)
	respondJSON(w, route)
}

func (h *Handler) handleUpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var route proxy.ProxyRoute
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid route data")
		return
	}

	route.ID = id
	if err := h.proxyManager.UpdateRoute(&route); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.wsHub.BroadcastEvent("route_update", route)
	respondJSON(w, route)
}

func (h *Handler) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.proxyManager.DeleteRoute(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.wsHub.BroadcastEvent("route_delete", map[string]string{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

// Metrics handlers
func (h *Handler) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.metrics.GetMetrics()
	respondJSON(w, metrics)
}

func (h *Handler) handleGetServiceMetrics(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "id")
	metrics := h.metrics.GetServiceMetrics(serviceID)
	if metrics == nil {
		respondError(w, http.StatusNotFound, "Service metrics not found")
		return
	}
	respondJSON(w, metrics)
}

// Log handlers
func (h *Handler) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	limit := 100 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	logs := h.metrics.GetLogs(level, limit)
	respondJSON(w, logs)
}

func (h *Handler) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	h.metrics.ClearLogs()
	w.WriteHeader(http.StatusNoContent)
}

// Configuration handlers
func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.getSystemConfig()
	respondJSON(w, config)
}

func (h *Handler) handleUpdateProxyConfig(w http.ResponseWriter, r *http.Request) {
	var config proxy.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid proxy configuration")
		return
	}

	if err := h.proxyManager.UpdateConfig(config); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.wsHub.BroadcastEvent("config_update", map[string]interface{}{
		"type": "proxy",
		"config": config,
	})
	respondJSON(w, config)
}

func (h *Handler) handleUpdateMetricsConfig(w http.ResponseWriter, r *http.Request) {
	var config MetricsConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid metrics configuration")
		return
	}

	h.metrics.UpdateConfig(config)
	h.wsHub.BroadcastEvent("config_update", map[string]interface{}{
		"type": "metrics",
		"config": config,
	})
	respondJSON(w, config)
}

// WebSocket handler
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Could not upgrade connection")
		return
	}

	// Create new client and start handling messages
	client := h.wsHub.NewClient(conn)
	go client.readPump()
	go client.writePump()
}

// Helper functions
func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// getSystemOverview returns the current system overview
func (h *Handler) getSystemOverview() Overview {
	metrics := h.metrics.GetMetrics()
	services := h.serviceRegistry.GetAllServices()
	routes := h.proxyManager.GetRoutes()

	var healthyServices int
	for _, service := range services {
		if service.Status == "healthy" {
			healthyServices++
		}
	}

	healthStatus := 100.0
	if len(services) > 0 {
		healthStatus = float64(healthyServices) / float64(len(services)) * 100
	}

	return Overview{
		TotalServices:   len(services),
		HealthyServices: healthyServices,
		ActiveRoutes:    len(routes),
		HealthStatus:    healthStatus,
		Metrics:         metrics,
	}
}

// getSystemConfig returns the current system configuration
func (h *Handler) getSystemConfig() SystemConfig {
	return SystemConfig{
		Proxy:   h.proxyManager.GetConfig(),
		Metrics: h.metrics.GetConfig(),
	}
}
