package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/timot/Quant_WebWork_GO/pkg/models"
)

type registerAppRequest struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Config      map[string]string `json:"config"`
}

type updateAppRequest struct {
	Description string            `json:"description"`
	Config      map[string]string `json:"config"`
	Status      string            `json:"status"`
}

// handleListApps returns a list of registered applications
func (h *Handler) handleListApps(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement pagination
	apps, err := h.appRegistry.ListApps(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list apps")
		return
	}

	respondJSON(w, http.StatusOK, apps)
}

// handleRegisterApp registers a new application
func (h *Handler) handleRegisterApp(w http.ResponseWriter, r *http.Request) {
	var req registerAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	app := &models.App{
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.appRegistry.RegisterApp(r.Context(), app); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register app")
		return
	}

	// Trigger reverse proxy configuration update
	if err := h.proxyManager.UpdateConfig(app); err != nil {
		// Log error but don't fail the request
		// TODO: Implement proper error logging
		respondError(w, http.StatusInternalServerError, "Failed to update proxy configuration")
		return
	}

	respondJSON(w, http.StatusCreated, app)
}

// handleGetApp returns details of a specific application
func (h *Handler) handleGetApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	
	app, err := h.appRegistry.GetApp(r.Context(), appID)
	if err != nil {
		if err == models.ErrAppNotFound {
			respondError(w, http.StatusNotFound, "App not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}

	respondJSON(w, http.StatusOK, app)
}

// handleUpdateApp updates an existing application
func (h *Handler) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")

	var req updateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	app, err := h.appRegistry.GetApp(r.Context(), appID)
	if err != nil {
		if err == models.ErrAppNotFound {
			respondError(w, http.StatusNotFound, "App not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}

	// Update fields
	app.Description = req.Description
	app.Config = req.Config
	app.Status = req.Status
	app.UpdatedAt = time.Now()

	if err := h.appRegistry.UpdateApp(r.Context(), app); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update app")
		return
	}

	// Update proxy configuration if needed
	if err := h.proxyManager.UpdateConfig(app); err != nil {
		// Log error but don't fail the request
		// TODO: Implement proper error logging
	}

	respondJSON(w, http.StatusOK, app)
}

// handleDeleteApp removes an application
func (h *Handler) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")

	app, err := h.appRegistry.GetApp(r.Context(), appID)
	if err != nil {
		if err == models.ErrAppNotFound {
			respondError(w, http.StatusNotFound, "App not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get app")
		return
	}

	if err := h.appRegistry.DeleteApp(r.Context(), appID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete app")
		return
	}

	// Remove proxy configuration
	if err := h.proxyManager.RemoveConfig(app); err != nil {
		// Log error but don't fail the request
		// TODO: Implement proper error logging
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
