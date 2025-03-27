// handlers.go - REST API handlers implementation

package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
)

// APIResponse standardizes API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

// APIError provides structured error information
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMeta provides metadata about the API response
type APIMeta struct {
	Timestamp   int64  `json:"timestamp"`
	RequestID   string `json:"request_id,omitempty"`
	APIVersion  string `json:"api_version,omitempty"`
	TimeElapsed string `json:"time_elapsed,omitempty"`
}

// HealthStatus represents the health check response
type HealthStatus struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Uptime    int64             `json:"uptime"` // seconds
	Timestamp int64             `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion   string   `json:"go_version"`
	NumCPU      int      `json:"num_cpu"`
	NumGoroutine int      `json:"num_goroutine"`
	GOOS        string   `json:"os"`
	GOARCH      string   `json:"arch"`
	Hostname    string   `json:"hostname"`
	Memory      MemInfo  `json:"memory"`
}

// MemInfo provides memory usage information
type MemInfo struct {
	Alloc      uint64 `json:"alloc"`       // bytes allocated and still in use
	TotalAlloc uint64 `json:"total_alloc"` // bytes allocated (even if freed)
	Sys        uint64 `json:"sys"`         // bytes obtained from system
	NumGC      uint32 `json:"num_gc"`      // number of garbage collections
}

// BridgeService represents a bridge service configuration
type BridgeService struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	URL         string            `json:"url"`
	Protocol    string            `json:"protocol"`
	Timeout     int               `json:"timeout"` // milliseconds
	Retries     int               `json:"retries"`
	Status      string            `json:"status"`
	LastSeen    int64             `json:"last_seen"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

// HealthCheckHandler handles health check requests
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Create health status response
	status := HealthStatus{
		Status:    "healthy",
		Version:   "1.0.0", // TODO: Get from config
		Uptime:    int64(time.Since(startTime).Seconds()),
		Timestamp: time.Now().Unix(),
		Services:  make(map[string]string),
	}
	
	// Add service statuses
	status.Services["api"] = "healthy"
	status.Services["bridge"] = "healthy"
	
	// Send response
	sendJSONResponse(w, http.StatusOK, status, startTime)
}

// SystemInfoHandler returns system information
func SystemInfoHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	info := SystemInfo{
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		Memory: MemInfo{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
		},
	}
	
	// Get hostname if possible
	info.Hostname = "localhost"
	
	sendJSONResponse(w, http.StatusOK, info, startTime)
}

// BridgeStatusHandler returns the status of the bridge system
func BridgeStatusHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	status := map[string]interface{}{
		"status":            "active",
		"active_services":   5, // Mock values
		"total_connections": 12,
		"uptime":            "12h 34m",
		"messages_sent":     1234,
		"messages_received": 2345,
	}
	
	sendJSONResponse(w, http.StatusOK, status, startTime)
}

// ListBridgeServicesHandler returns a list of registered bridge services
func ListBridgeServicesHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Mock service list for now
	services := []BridgeService{
		{
			ID:       "service-1",
			Name:     "Example Service 1",
			Type:     "rest",
			URL:      "https://api.example.com/v1",
			Protocol: "http",
			Timeout:  5000,
			Retries:  3,
			Status:   "active",
			LastSeen: time.Now().Unix(),
			Metadata: map[string]string{
				"region": "us-west",
				"tier":   "production",
			},
		},
		{
			ID:       "service-2",
			Name:     "Example Service 2",
			Type:     "grpc",
			URL:      "grpc.example.com:50051",
			Protocol: "grpc",
			Timeout:  3000,
			Retries:  2,
			Status:   "active",
			LastSeen: time.Now().Unix(),
			Metadata: map[string]string{
				"region": "us-east",
				"tier":   "staging",
			},
		},
	}
	
	sendJSONResponse(w, http.StatusOK, services, startTime)
}

// GetBridgeServiceHandler returns details for a specific bridge service
func GetBridgeServiceHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	vars := mux.Vars(r)
	serviceID := vars["id"]
	
	// TODO: Fetch actual service by ID from bridge manager
	service := BridgeService{
		ID:       serviceID,
		Name:     "Example Service",
		Type:     "rest",
		URL:      "https://api.example.com/v1",
		Protocol: "http",
		Timeout:  5000,
		Retries:  3,
		Status:   "active",
		LastSeen: time.Now().Unix(),
		Metadata: map[string]string{
			"region": "us-west",
			"tier":   "production",
		},
	}
	
	sendJSONResponse(w, http.StatusOK, service, startTime)
}

// RegisterBridgeServiceHandler registers a new bridge service
func RegisterBridgeServiceHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	var service BridgeService
	err := json.NewDecoder(r.Body).Decode(&service)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body", err.Error(), startTime)
		return
	}
	
	// Validate required fields
	if service.Name == "" || service.URL == "" || service.Protocol == "" {
		sendErrorResponse(w, http.StatusBadRequest, "validation_error", "Missing required fields", "Name, URL, and Protocol are required", startTime)
		return
	}
	
	// TODO: Register service with bridge manager
	service.ID = fmt.Sprintf("service-%d", time.Now().Unix())
	service.Status = "active"
	service.LastSeen = time.Now().Unix()
	
	sendJSONResponse(w, http.StatusCreated, service, startTime)
}

// UpdateBridgeServiceHandler updates an existing bridge service
func UpdateBridgeServiceHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	vars := mux.Vars(r)
	serviceID := vars["id"]
	
	var service BridgeService
	err := json.NewDecoder(r.Body).Decode(&service)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body", err.Error(), startTime)
		return
	}
	
	// Set the ID from path parameter
	service.ID = serviceID
	service.LastSeen = time.Now().Unix()
	
	// TODO: Update service in bridge manager
	
	sendJSONResponse(w, http.StatusOK, service, startTime)
}

// DeregisterBridgeServiceHandler removes a bridge service
func DeregisterBridgeServiceHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	vars := mux.Vars(r)
	serviceID := vars["id"]
	
	// TODO: Remove service from bridge manager
	
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"id":      serviceID,
		"message": "Service successfully deregistered",
	}, startTime)
}

// sendJSONResponse sends a standardized JSON response
func sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}, startTime time.Time) {
	response := APIResponse{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
		Meta: &APIMeta{
			Timestamp:   time.Now().Unix(),
			APIVersion:  "v1",
			TimeElapsed: fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse sends a standardized error response
func sendErrorResponse(w http.ResponseWriter, statusCode int, code, message, details string, startTime time.Time) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &APIMeta{
			Timestamp:   time.Now().Unix(),
			APIVersion:  "v1",
			TimeElapsed: fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(metricsCollector *metrics.Collector, method, path string, status int, duration time.Duration) {
	if metricsCollector == nil {
		return
	}
	
	tags := map[string]string{
		"method": method,
		"path":   path,
		"status": fmt.Sprintf("%d", status),
	}
	
	metricsCollector.Collect("http", "request_duration", float64(duration.Milliseconds()), tags)
	metricsCollector.Collect("http", "requests_total", 1.0, tags)
	
	// Categorize by status code family (2xx, 4xx, 5xx)
	statusFamily := fmt.Sprintf("%dxx", status/100)
	metricsCollector.Collect("http", "requests_"+statusFamily, 1.0, tags)
}
