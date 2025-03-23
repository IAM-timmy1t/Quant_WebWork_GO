package rest

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/timot/Quant_WebWork_GO/internal/dashboard"
	"github.com/timot/Quant_WebWork_GO/internal/monitoring"
)

// DashboardHandler handles dashboard-related HTTP requests
type DashboardHandler struct {
	service *dashboard.Service
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(service *dashboard.Service) *DashboardHandler {
	return &DashboardHandler{
		service: service,
	}
}

// RegisterRoutes registers dashboard routes
func (h *DashboardHandler) RegisterRoutes(r *mux.Router) {
	// Dashboard pages
	r.HandleFunc("/dashboard", h.handleDashboard).Methods("GET")
	r.HandleFunc("/dashboard/metrics", h.handleMetricsPage).Methods("GET")
	r.HandleFunc("/dashboard/security", h.handleSecurityPage).Methods("GET")

	// API endpoints
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Metrics endpoints
	api.HandleFunc("/metrics/current", h.handleCurrentMetrics).Methods("GET")
	api.HandleFunc("/metrics/range", h.handleMetricsRange).Methods("GET")
	api.HandleFunc("/metrics/aggregated", h.handleAggregatedMetrics).Methods("GET")
	api.HandleFunc("/metrics/export", h.handleMetricsExport).Methods("GET")

	// Security endpoints
	api.HandleFunc("/security/events", h.handleSecurityEvents).Methods("GET")
	api.HandleFunc("/security/status", h.handleSecurityStatus).Methods("GET")
	api.HandleFunc("/security/risks", h.handleRiskScores).Methods("GET")

	// System endpoints
	api.HandleFunc("/system/status", h.handleSystemStatus).Methods("GET")
	api.HandleFunc("/system/health", h.handleHealthCheck).Methods("GET")
}

// handleDashboard serves the main dashboard page
func (h *DashboardHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dashboard/index.html")
}

// handleMetricsPage serves the metrics dashboard page
func (h *DashboardHandler) handleMetricsPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dashboard/metrics.html")
}

// handleSecurityPage serves the security dashboard page
func (h *DashboardHandler) handleSecurityPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dashboard/security.html")
}

// handleCurrentMetrics returns current system metrics
func (h *DashboardHandler) handleCurrentMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetLatestMetrics(r.Context())
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, metrics)
}

// handleMetricsRange returns metrics for a time range
func (h *DashboardHandler) handleMetricsRange(w http.ResponseWriter, r *http.Request) {
	start, end, err := h.parseTimeRange(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	metrics, err := h.service.GetMetricsRange(r.Context(), start, end)
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, metrics)
}

// handleAggregatedMetrics returns aggregated metrics
func (h *DashboardHandler) handleAggregatedMetrics(w http.ResponseWriter, r *http.Request) {
	start, end, err := h.parseTimeRange(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "hourly"
	}

	metrics, err := h.service.GetAggregatedMetrics(r.Context(), start, end, dashboard.MetricsAggregation(period))
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, metrics)
}

// handleMetricsExport exports metrics in various formats
func (h *DashboardHandler) handleMetricsExport(w http.ResponseWriter, r *http.Request) {
	start, end, err := h.parseTimeRange(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	metrics, err := h.service.GetMetricsRange(r.Context(), start, end)
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	switch dashboard.MetricsExportFormat(format) {
	case dashboard.FormatCSV:
		h.writeMetricsCSV(w, metrics)
	case dashboard.FormatJSON:
		h.writeJSON(w, metrics)
	default:
		h.writeError(w, fmt.Errorf("unsupported format: %s", format), http.StatusBadRequest)
	}
}

// writeMetricsCSV writes metrics data in CSV format
func (h *DashboardHandler) writeMetricsCSV(w http.ResponseWriter, metrics interface{}) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=metrics.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Timestamp", "Type", "Source", "Value", "Unit"})

	// Write data
	switch m := metrics.(type) {
	case []monitoring.ResourceMetrics:
		for _, metric := range m {
			writer.Write([]string{
				metric.Timestamp.Format(time.RFC3339),
				metric.Type,
				metric.Source,
				fmt.Sprintf("%f", metric.Value),
				metric.Unit,
			})
		}
	case monitoring.ResourceMetrics:
		writer.Write([]string{
			m.Timestamp.Format(time.RFC3339),
			m.Type,
			m.Source,
			fmt.Sprintf("%f", m.Value),
			m.Unit,
		})
	}
}

// handleSecurityEvents returns security events
func (h *DashboardHandler) handleSecurityEvents(w http.ResponseWriter, r *http.Request) {
	start, end, err := h.parseTimeRange(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	events, err := h.service.GetSecurityEvents(r.Context(), start, end)
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "csv" {
		h.writeSecurityEventsCSV(w, events)
		return
	}

	h.writeJSON(w, events)
}

// writeSecurityEventsCSV writes security events in CSV format
func (h *DashboardHandler) writeSecurityEventsCSV(w http.ResponseWriter, events []monitoring.SecurityEvent) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=security_events.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Timestamp", "Type", "Source", "Severity", "Description"})

	// Write data
	for _, event := range events {
		writer.Write([]string{
			event.Timestamp.Format(time.RFC3339),
			event.Type,
			event.Source,
			event.Severity,
			event.Description,
		})
	}
}

// handleSecurityStatus returns current security status
func (h *DashboardHandler) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetSecurityStatus(r.Context())
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, status)
}

// handleRiskScores returns current risk scores
func (h *DashboardHandler) handleRiskScores(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source != "" {
		score, err := h.service.GetRiskScore(r.Context(), source)
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
		h.writeJSON(w, map[string]float64{source: score})
		return
	}

	// Return all risk scores if no source specified
	h.writeJSON(w, h.service.GetSecurityStatus())
}

// handleSystemStatus returns overall system status
func (h *DashboardHandler) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetLatestMetrics(r.Context())
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	security, err := h.service.GetSecurityStatus(r.Context())
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"metrics":  metrics,
		"security": security,
		"time":     time.Now(),
	}

	h.writeJSON(w, status)
}

// handleHealthCheck performs a health check
func (h *DashboardHandler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
	}

	h.writeJSON(w, health)
}

// Helper functions

func (h *DashboardHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *DashboardHandler) writeError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (h *DashboardHandler) parseTimeRange(r *http.Request) (time.Time, time.Time, error) {
	query := r.URL.Query()
	
	// Parse start time
	startStr := query.Get("start")
	if startStr == "" {
		// Default to 24 hours ago
		return time.Now().Add(-24 * time.Hour), time.Now(), nil
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start time format: %v", err)
	}

	// Parse end time
	endStr := query.Get("end")
	var end time.Time
	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end time format: %v", err)
		}
	}

	// Validate time range
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end time cannot be before start time")
	}

	return start, end, nil
}
