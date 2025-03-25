// security_handlers.go - Handlers for security API endpoints

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// SecurityHandlers manages security-related API endpoints
type SecurityHandlers struct {
	securityMonitor SecurityMonitor
	scanner         SecurityScanner
	firewall        Firewall
	logger          Logger
	tokenManager    TokenManager
}

// SecurityMonitor interface for security monitoring functionality
type SecurityMonitor interface {
	ProcessEvent(event interface{}) error
	GetEventHistory(limit int) ([]interface{}, error)
	GetAlerts(filter map[string]interface{}, limit int) ([]interface{}, error)
	AcknowledgeAlert(alertID string) error
	GetRiskScore(target string) (float64, error)
}

// SecurityScanner interface for security scanning functionality
type SecurityScanner interface {
	Scan(target string, scanTypes []string, options map[string]interface{}) (interface{}, error)
	ScanAsync(target string, scanTypes []string, options map[string]interface{}, resultCh chan<- interface{}) (string, error)
	GetScanStatus(scanID string) (interface{}, error)
	GetScanResults(scanID string) (interface{}, error)
	ListScans(filter map[string]interface{}, limit int) ([]interface{}, error)
}

// Firewall interface for firewall functionality
type Firewall interface {
	AddRule(rule interface{}) error
	UpdateRule(rule interface{}) error
	GetRule(ruleID string) (interface{}, error)
	ListRules(filter map[string]interface{}) ([]interface{}, error)
	DeleteRule(ruleID string) error
	EnableRule(ruleID string) error
	DisableRule(ruleID string) error
	Evaluate(request interface{}) (interface{}, error)
}

// NewSecurityHandlers creates a new security handlers instance
func NewSecurityHandlers(monitor SecurityMonitor, scanner SecurityScanner, firewall Firewall, logger Logger) *SecurityHandlers {
	return &SecurityHandlers{
		securityMonitor: monitor,
		scanner:         scanner,
		firewall:        firewall,
		logger:          logger,
	}
}

// SetTokenManager sets the token manager for authentication
func (h *SecurityHandlers) SetTokenManager(tokenManager TokenManager) {
	h.tokenManager = tokenManager
}

// RegisterRoutes registers the security API routes
func (h *SecurityHandlers) RegisterRoutes(router *mux.Router, authMiddleware mux.MiddlewareFunc) {
	// Event endpoints
	eventsRouter := router.PathPrefix("/events").Subrouter()
	eventsRouter.Use(authMiddleware)
	eventsRouter.HandleFunc("", h.GetEvents).Methods("GET")
	eventsRouter.HandleFunc("", h.ReportEvent).Methods("POST")
	
	// Alert endpoints
	alertsRouter := router.PathPrefix("/alerts").Subrouter()
	alertsRouter.Use(authMiddleware)
	alertsRouter.HandleFunc("", h.GetAlerts).Methods("GET")
	alertsRouter.HandleFunc("/{alertID}", h.GetAlert).Methods("GET")
	alertsRouter.HandleFunc("/{alertID}/acknowledge", h.AcknowledgeAlert).Methods("POST")
	
	// Scan endpoints
	scanRouter := router.PathPrefix("/scans").Subrouter()
	scanRouter.Use(authMiddleware)
	scanRouter.HandleFunc("", h.ListScans).Methods("GET")
	scanRouter.HandleFunc("", h.StartScan).Methods("POST")
	scanRouter.HandleFunc("/{scanID}", h.GetScanStatus).Methods("GET")
	scanRouter.HandleFunc("/{scanID}/results", h.GetScanResults).Methods("GET")
	
	// Firewall endpoints
	firewallRouter := router.PathPrefix("/firewall").Subrouter()
	firewallRouter.Use(authMiddleware)
	firewallRouter.HandleFunc("/rules", h.ListRules).Methods("GET")
	firewallRouter.HandleFunc("/rules", h.AddRule).Methods("POST")
	firewallRouter.HandleFunc("/rules/{ruleID}", h.GetRule).Methods("GET")
	firewallRouter.HandleFunc("/rules/{ruleID}", h.UpdateRule).Methods("PUT")
	firewallRouter.HandleFunc("/rules/{ruleID}", h.DeleteRule).Methods("DELETE")
	firewallRouter.HandleFunc("/rules/{ruleID}/enable", h.EnableRule).Methods("POST")
	firewallRouter.HandleFunc("/rules/{ruleID}/disable", h.DisableRule).Methods("POST")
	firewallRouter.HandleFunc("/evaluate", h.EvaluateRequest).Methods("POST")
	
	// Risk assessment endpoints
	riskRouter := router.PathPrefix("/risk").Subrouter()
	riskRouter.Use(authMiddleware)
	riskRouter.HandleFunc("/score", h.GetRiskScore).Methods("GET")
}

// GetEvents handles requests to retrieve security events
func (h *SecurityHandlers) GetEvents(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default limit
	
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	events, err := h.securityMonitor.GetEventHistory(limit)
	if err != nil {
		h.logger.Error("Failed to get event history", map[string]interface{}{
			"error": err.Error(),
			"limit": limit,
		})
		http.Error(w, "Failed to retrieve events: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
		"limit":  limit,
	})
}

// ReportEvent handles requests to report a security event
func (h *SecurityHandlers) ReportEvent(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.logger.Error("Failed to decode event", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Add timestamp if not present
	if _, ok := event["timestamp"]; !ok {
		event["timestamp"] = time.Now().UTC()
	}
	
	if err := h.securityMonitor.ProcessEvent(event); err != nil {
		h.logger.Error("Failed to process event", map[string]interface{}{
			"error": err.Error(),
			"event": event,
		})
		http.Error(w, "Failed to process event: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Event processed successfully",
	})
}

// GetAlerts handles requests to retrieve security alerts
func (h *SecurityHandlers) GetAlerts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default limit
	
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	// Parse filter parameters
	filter := make(map[string]interface{})
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter["severity"] = severity
	}
	
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}
	
	if timeRange := r.URL.Query().Get("timeRange"); timeRange != "" {
		filter["timeRange"] = timeRange
	}
	
	alerts, err := h.securityMonitor.GetAlerts(filter, limit)
	if err != nil {
		h.logger.Error("Failed to get alerts", map[string]interface{}{
			"error":  err.Error(),
			"filter": filter,
			"limit":  limit,
		})
		http.Error(w, "Failed to retrieve alerts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
		"limit":  limit,
	})
}

// GetAlert handles requests to retrieve a specific security alert
func (h *SecurityHandlers) GetAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["alertID"]
	
	if alertID == "" {
		http.Error(w, "Alert ID is required", http.StatusBadRequest)
		return
	}
	
	// This is a mock implementation, in a real system we would fetch the alert
	// from the security monitor
	alerts, err := h.securityMonitor.GetAlerts(map[string]interface{}{
		"id": alertID,
	}, 1)
	
	if err != nil {
		h.logger.Error("Failed to get alert", map[string]interface{}{
			"error":   err.Error(),
			"alertID": alertID,
		})
		http.Error(w, "Failed to retrieve alert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	if len(alerts) == 0 {
		http.Error(w, "Alert not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts[0])
}

// AcknowledgeAlert handles requests to acknowledge a security alert
func (h *SecurityHandlers) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["alertID"]
	
	if alertID == "" {
		http.Error(w, "Alert ID is required", http.StatusBadRequest)
		return
	}
	
	if err := h.securityMonitor.AcknowledgeAlert(alertID); err != nil {
		h.logger.Error("Failed to acknowledge alert", map[string]interface{}{
			"error":   err.Error(),
			"alertID": alertID,
		})
		http.Error(w, "Failed to acknowledge alert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Alert acknowledged successfully",
	})
}

// StartScan handles requests to start a security scan
func (h *SecurityHandlers) StartScan(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Target    string                 `json:"target"`
		ScanTypes []string               `json:"scanTypes"`
		Options   map[string]interface{} `json:"options"`
		Async     bool                   `json:"async"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("Failed to decode scan request", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	if request.Target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}
	
	if request.Async {
		// Run scan asynchronously
		resultCh := make(chan interface{}, 1)
		scanID, err := h.scanner.ScanAsync(request.Target, request.ScanTypes, request.Options, resultCh)
		
		if err != nil {
			h.logger.Error("Failed to start async scan", map[string]interface{}{
				"error":  err.Error(),
				"target": request.Target,
			})
			http.Error(w, "Failed to start scan: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Scan started successfully",
			"scanID":  scanID,
		})
	} else {
		// Run scan synchronously
		result, err := h.scanner.Scan(request.Target, request.ScanTypes, request.Options)
		
		if err != nil {
			h.logger.Error("Failed to run scan", map[string]interface{}{
				"error":  err.Error(),
				"target": request.Target,
			})
			http.Error(w, "Failed to run scan: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// GetScanStatus handles requests to get the status of a security scan
func (h *SecurityHandlers) GetScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scanID"]
	
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}
	
	status, err := h.scanner.GetScanStatus(scanID)
	if err != nil {
		h.logger.Error("Failed to get scan status", map[string]interface{}{
			"error":  err.Error(),
			"scanID": scanID,
		})
		http.Error(w, "Failed to get scan status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetScanResults handles requests to get the results of a security scan
func (h *SecurityHandlers) GetScanResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scanID"]
	
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}
	
	results, err := h.scanner.GetScanResults(scanID)
	if err != nil {
		h.logger.Error("Failed to get scan results", map[string]interface{}{
			"error":  err.Error(),
			"scanID": scanID,
		})
		http.Error(w, "Failed to get scan results: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// ListScans handles requests to list security scans
func (h *SecurityHandlers) ListScans(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default limit
	
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	// Parse filter parameters
	filter := make(map[string]interface{})
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}
	
	if target := r.URL.Query().Get("target"); target != "" {
		filter["target"] = target
	}
	
	scans, err := h.scanner.ListScans(filter, limit)
	if err != nil {
		h.logger.Error("Failed to list scans", map[string]interface{}{
			"error":  err.Error(),
			"filter": filter,
			"limit":  limit,
		})
		http.Error(w, "Failed to list scans: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scans": scans,
		"count": len(scans),
		"limit": limit,
	})
}

// ListRules handles requests to list firewall rules
func (h *SecurityHandlers) ListRules(w http.ResponseWriter, r *http.Request) {
	// Parse filter parameters
	filter := make(map[string]interface{})
	if ruleType := r.URL.Query().Get("type"); ruleType != "" {
		filter["type"] = ruleType
	}
	
	if enabled := r.URL.Query().Get("enabled"); enabled != "" {
		filter["enabled"] = enabled == "true"
	}
	
	rules, err := h.firewall.ListRules(filter)
	if err != nil {
		h.logger.Error("Failed to list rules", map[string]interface{}{
			"error":  err.Error(),
			"filter": filter,
		})
		http.Error(w, "Failed to list rules: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// GetRule handles requests to get a specific firewall rule
func (h *SecurityHandlers) GetRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["ruleID"]
	
	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}
	
	rule, err := h.firewall.GetRule(ruleID)
	if err != nil {
		h.logger.Error("Failed to get rule", map[string]interface{}{
			"error":  err.Error(),
			"ruleID": ruleID,
		})
		http.Error(w, "Failed to get rule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// AddRule handles requests to add a firewall rule
func (h *SecurityHandlers) AddRule(w http.ResponseWriter, r *http.Request) {
	var rule interface{}
	
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		h.logger.Error("Failed to decode rule", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := h.firewall.AddRule(rule); err != nil {
		h.logger.Error("Failed to add rule", map[string]interface{}{
			"error": err.Error(),
			"rule":  rule,
		})
		http.Error(w, "Failed to add rule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rule added successfully",
	})
}

// UpdateRule handles requests to update a firewall rule
func (h *SecurityHandlers) UpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["ruleID"]
	
	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}
	
	var rule interface{}
	
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		h.logger.Error("Failed to decode rule", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := h.firewall.UpdateRule(rule); err != nil {
		h.logger.Error("Failed to update rule", map[string]interface{}{
			"error":  err.Error(),
			"ruleID": ruleID,
			"rule":   rule,
		})
		http.Error(w, "Failed to update rule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rule updated successfully",
	})
}

// DeleteRule handles requests to delete a firewall rule
func (h *SecurityHandlers) DeleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["ruleID"]
	
	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}
	
	if err := h.firewall.DeleteRule(ruleID); err != nil {
		h.logger.Error("Failed to delete rule", map[string]interface{}{
			"error":  err.Error(),
			"ruleID": ruleID,
		})
		http.Error(w, "Failed to delete rule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rule deleted successfully",
	})
}

// EnableRule handles requests to enable a firewall rule
func (h *SecurityHandlers) EnableRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["ruleID"]
	
	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}
	
	if err := h.firewall.EnableRule(ruleID); err != nil {
		h.logger.Error("Failed to enable rule", map[string]interface{}{
			"error":  err.Error(),
			"ruleID": ruleID,
		})
		http.Error(w, "Failed to enable rule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rule enabled successfully",
	})
}

// DisableRule handles requests to disable a firewall rule
func (h *SecurityHandlers) DisableRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["ruleID"]
	
	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}
	
	if err := h.firewall.DisableRule(ruleID); err != nil {
		h.logger.Error("Failed to disable rule", map[string]interface{}{
			"error":  err.Error(),
			"ruleID": ruleID,
		})
		http.Error(w, "Failed to disable rule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rule disabled successfully",
	})
}

// EvaluateRequest handles requests to evaluate a request against firewall rules
func (h *SecurityHandlers) EvaluateRequest(w http.ResponseWriter, r *http.Request) {
	var request interface{}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("Failed to decode request", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := h.firewall.Evaluate(request)
	if err != nil {
		h.logger.Error("Failed to evaluate request", map[string]interface{}{
			"error":   err.Error(),
			"request": request,
		})
		http.Error(w, "Failed to evaluate request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetRiskScore handles requests to get a risk score for a target
func (h *SecurityHandlers) GetRiskScore(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	
	if target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}
	
	score, err := h.securityMonitor.GetRiskScore(target)
	if err != nil {
		h.logger.Error("Failed to get risk score", map[string]interface{}{
			"error":  err.Error(),
			"target": target,
		})
		http.Error(w, "Failed to get risk score: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"target": target,
		"score":  score,
	})
}

// Logger interface for security handlers
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// TokenManager interface for authentication
type TokenManager interface {
	ValidateToken(token string) (map[string]interface{}, error)
	GenerateToken(claims map[string]interface{}, expiresIn time.Duration) (string, error)
	RevokeToken(token string) error
}
