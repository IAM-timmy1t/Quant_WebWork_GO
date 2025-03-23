package discovery

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for service discovery
type Handler struct {
	registry *ServiceRegistry
}

// NewHandler creates a new discovery HTTP handler
func NewHandler(registry *ServiceRegistry) *Handler {
	return &Handler{registry: registry}
}

// RegisterRoutes registers the discovery API routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/discovery", func(r chi.Router) {
		r.Post("/services", h.registerService)
		r.Get("/services", h.listServices)
		r.Get("/services/{id}", h.getService)
		r.Put("/services/{id}", h.updateService)
		r.Delete("/services/{id}", h.deregisterService)
	})
}

// registerService handles service registration requests
func (h *Handler) registerService(w http.ResponseWriter, r *http.Request) {
	var service Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.registry.Register(&service); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(service)
}

// listServices returns all registered services
func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {
	h.registry.mu.RLock()
	services := make([]*Service, 0, len(h.registry.services))
	for _, service := range h.registry.services {
		services = append(services, service)
	}
	h.registry.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// getService returns a specific service by ID
func (h *Handler) getService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	service, err := h.registry.GetService(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

// updateService updates an existing service
func (h *Handler) updateService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	var service Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service.ID = id
	if err := h.registry.Update(&service); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

// deregisterService removes a service from the registry
func (h *Handler) deregisterService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	if err := h.registry.Deregister(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ServiceResponse represents the JSON response for service operations
type ServiceResponse struct {
	Service *Service `json:"service,omitempty"`
	Error   string   `json:"error,omitempty"`
}
