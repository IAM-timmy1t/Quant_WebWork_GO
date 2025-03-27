// error_handler.go - Standardized error handling for REST APIs

package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Status     int               `json:"-"`                    // HTTP status code (not included in response)
	Code       string            `json:"code"`                 // Application-specific error code
	Message    string            `json:"message"`              // Human-readable error message
	Details    interface{}       `json:"details,omitempty"`    // Additional error details
	RequestID  string            `json:"request_id,omitempty"` // Request ID for tracking
	Timestamp  string            `json:"timestamp"`            // When the error occurred
	Path       string            `json:"path,omitempty"`       // Request path
	Validation []ValidationError `json:"validation,omitempty"` // Validation errors
	TraceID    string            `json:"trace_id,omitempty"`   // Trace ID for debugging (only in dev/staging)
}

// ValidationError represents a specific validation error
type ValidationError struct {
	Field   string `json:"field"`           // Field that failed validation
	Message string `json:"message"`         // Error message
	Code    string `json:"code,omitempty"`  // Error code
	Value   string `json:"value,omitempty"` // Value that caused the error
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// APIError types
const (
	ErrorBadRequest         = "BAD_REQUEST"
	ErrorUnauthorized       = "UNAUTHORIZED"
	ErrorForbidden          = "FORBIDDEN"
	ErrorNotFound           = "NOT_FOUND"
	ErrorMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	ErrorConflict           = "CONFLICT"
	ErrorTooManyRequests    = "TOO_MANY_REQUESTS"
	ErrorInternalServer     = "INTERNAL_SERVER_ERROR"
	ErrorServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrorValidation         = "VALIDATION_ERROR"
	ErrorResourceExists     = "RESOURCE_EXISTS"
	ErrorResourceNotFound   = "RESOURCE_NOT_FOUND"
	ErrorInvalidToken       = "INVALID_TOKEN"
	ErrorExpiredToken       = "EXPIRED_TOKEN"
	ErrorInsufficientScope  = "INSUFFICIENT_SCOPE"
	ErrorDatabaseError      = "DATABASE_ERROR"
	ErrorExternalService    = "EXTERNAL_SERVICE_ERROR"
	ErrorRateLimit          = "RATE_LIMIT_EXCEEDED"
	ErrorMalformedData      = "MALFORMED_DATA"
	ErrorInvalidParameters  = "INVALID_PARAMETERS"
)

// StandardErrorHandler provides a comprehensive error handling solution for API endpoints
type StandardErrorHandler struct {
	logger            Logger
	metricsCollector  metrics.Collector
	environment       string // "development", "staging", "production"
	includeStackTrace bool
	sensitiveFields   []string          // Fields to redact from error details
	errorMessages     map[string]string // Custom error messages
	errorMapping      map[error]string  // Map standard errors to error codes
}

// NewStandardErrorHandler creates a new error handler
func NewStandardErrorHandler(logger Logger, metricsCollector metrics.Collector) *StandardErrorHandler {
	handler := &StandardErrorHandler{
		logger:            logger,
		metricsCollector:  metricsCollector,
		environment:       "development",
		includeStackTrace: true,
		sensitiveFields:   []string{"password", "token", "secret", "key", "auth"},
		errorMessages:     make(map[string]string),
		errorMapping:      make(map[error]string),
	}

	// Set default error messages
	handler.SetDefaultErrorMessages()

	return handler
}

// SetEnvironment configures the environment setting
func (h *StandardErrorHandler) SetEnvironment(env string) {
	h.environment = strings.ToLower(env)
	// Only include stack traces in development or staging
	h.includeStackTrace = h.environment == "development" || h.environment == "staging"
}

// SetSensitiveFields sets fields that should be redacted from error details
func (h *StandardErrorHandler) SetSensitiveFields(fields []string) {
	h.sensitiveFields = fields
}

// SetErrorMessage sets a custom error message for a specific error code
func (h *StandardErrorHandler) SetErrorMessage(code, message string) {
	h.errorMessages[code] = message
}

// MapError maps a standard error to an API error code
func (h *StandardErrorHandler) MapError(err error, code string) {
	h.errorMapping[err] = code
}

// SetDefaultErrorMessages configures default error messages
func (h *StandardErrorHandler) SetDefaultErrorMessages() {
	messages := map[string]string{
		ErrorBadRequest:         "The request contains invalid parameters or payload",
		ErrorUnauthorized:       "Authentication is required to access this resource",
		ErrorForbidden:          "You don't have permission to access this resource",
		ErrorNotFound:           "The requested resource was not found",
		ErrorMethodNotAllowed:   "The HTTP method is not supported for this resource",
		ErrorConflict:           "The request conflicts with the current state of the resource",
		ErrorTooManyRequests:    "Rate limit exceeded, please try again later",
		ErrorInternalServer:     "An unexpected error occurred while processing your request",
		ErrorServiceUnavailable: "The service is temporarily unavailable, please try again later",
		ErrorValidation:         "The request contains validation errors",
		ErrorResourceExists:     "The resource already exists",
		ErrorResourceNotFound:   "The requested resource was not found",
		ErrorInvalidToken:       "The provided authentication token is invalid",
		ErrorExpiredToken:       "The authentication token has expired",
		ErrorInsufficientScope:  "The token does not have the required permissions",
		ErrorDatabaseError:      "A database error occurred while processing your request",
		ErrorExternalService:    "An error occurred with an external service",
		ErrorRateLimit:          "Rate limit exceeded, please try again later",
		ErrorMalformedData:      "The request contains malformed data",
		ErrorInvalidParameters:  "One or more request parameters are invalid",
	}

	for code, message := range messages {
		h.errorMessages[code] = message
	}
}

// defaultErrorHandler is the default handler for errors
var defaultErrorHandler = func(w http.ResponseWriter, r *http.Request, err error, status int) {
	// Create a simple error response
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"status":    status,
			"message":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
			"path":      r.URL.Path,
			"method":    r.Method,
		},
	}

	// Send error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse)
}

// HandleError processes and returns a standardized API error
func (h *StandardErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error, status int) {
	// Extract request ID and start time from context if available
	ctx := r.Context()
	requestID, _ := ctx.Value(ContextRequestID).(string)
	startTime, _ := ctx.Value(ContextStartTime).(time.Time)

	// Generate trace ID for internal tracking
	traceID := fmt.Sprintf("%s-%d", requestID, time.Now().UnixNano())

	// Map known errors to error codes
	errorCode := h.getErrorCode(err, status)

	// Create error response
	errorResponse := &ErrorResponse{
		Status:    status,
		Code:      errorCode,
		Message:   h.getErrorMessage(errorCode, err),
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      r.URL.Path,
	}

	// Add trace ID in non-production environments
	if h.environment != "production" {
		errorResponse.TraceID = traceID
	}

	// Check for validation errors
	if validationErr, ok := err.(ValidationErrors); ok {
		errorResponse.Validation = validationErr.Errors
	}

	// Add request duration for metrics
	var duration time.Duration
	if !startTime.IsZero() {
		duration = time.Since(startTime)
	}

	// Log the error with appropriate level based on status code
	logFields := map[string]interface{}{
		"request_id": requestID,
		"trace_id":   traceID,
		"status":     status,
		"error_code": errorCode,
		"path":       r.URL.Path,
		"method":     r.Method,
		"duration":   duration.String(),
		"client_ip":  GetIPAddress(r, nil),
	}

	if h.includeStackTrace {
		logFields["stack_trace"] = string(debug.Stack())
	}

	// Log with appropriate level
	if status >= 500 {
		h.logger.Error(err.Error(), logFields)
	} else if status >= 400 {
		h.logger.Warn(err.Error(), logFields)
	} else {
		h.logger.Info(err.Error(), logFields)
	}

	// Track error metrics if metrics collector is available
	if errorCode != "" {
		if h.metricsCollector != (metrics.Collector{}) {
			h.metricsCollector.IncErrorCount(errorCode, r.Method, r.URL.Path, status)
		}
	}

	// Send error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse)
}

// getErrorCode maps an error to its appropriate code
func (h *StandardErrorHandler) getErrorCode(err error, status int) string {
	// Check for mapped errors
	for mappedErr, code := range h.errorMapping {
		if errors.Is(err, mappedErr) {
			return code
		}
	}

	// Map HTTP status codes to error codes
	switch status {
	case http.StatusBadRequest:
		return ErrorBadRequest
	case http.StatusUnauthorized:
		return ErrorUnauthorized
	case http.StatusForbidden:
		return ErrorForbidden
	case http.StatusNotFound:
		return ErrorNotFound
	case http.StatusMethodNotAllowed:
		return ErrorMethodNotAllowed
	case http.StatusConflict:
		return ErrorConflict
	case http.StatusTooManyRequests:
		return ErrorTooManyRequests
	case http.StatusInternalServerError:
		return ErrorInternalServer
	case http.StatusServiceUnavailable:
		return ErrorServiceUnavailable
	default:
		if status >= 400 && status < 500 {
			return ErrorBadRequest
		}
		return ErrorInternalServer
	}
}

// getErrorMessage returns a human-readable error message
func (h *StandardErrorHandler) getErrorMessage(code string, err error) string {
	// Use custom message if available
	if message, ok := h.errorMessages[code]; ok {
		return message
	}

	// Otherwise use the error message
	if err != nil {
		return err.Error()
	}

	return "An unexpected error occurred"
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError
}

// Error implements the error interface
func (v ValidationErrors) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}

	messages := make([]string, len(v.Errors))
	for i, err := range v.Errors {
		messages[i] = fmt.Sprintf("%s: %s", err.Field, err.Message)
	}

	return fmt.Sprintf("validation errors: %s", strings.Join(messages, "; "))
}

// NewValidationError creates a new validation error
func NewValidationError(field, message, code string, value interface{}) ValidationError {
	valueStr := ""
	if value != nil {
		valueStr = fmt.Sprintf("%v", value)
	}

	return ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
		Value:   valueStr,
	}
}

// NewValidationErrors creates a validation errors container
func NewValidationErrors(errors ...ValidationError) ValidationErrors {
	return ValidationErrors{Errors: errors}
}

// BadRequest returns a formatted 400 Bad Request error
func BadRequest(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	handler(w, r, err, http.StatusBadRequest)
}

// Unauthorized returns a formatted 401 Unauthorized error
func Unauthorized(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	handler(w, r, err, http.StatusUnauthorized)
}

// Forbidden returns a formatted 403 Forbidden error
func Forbidden(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	handler(w, r, err, http.StatusForbidden)
}

// NotFound returns a formatted 404 Not Found error
func NotFound(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	if err == nil {
		err = errors.New("resource not found")
	}
	handler(w, r, err, http.StatusNotFound)
}

// MethodNotAllowed returns a formatted 405 Method Not Allowed error
func MethodNotAllowed(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	if err == nil {
		err = fmt.Errorf("method %s not allowed for %s", r.Method, r.URL.Path)
	}
	handler(w, r, err, http.StatusMethodNotAllowed)
}

// Conflict returns a formatted 409 Conflict error
func Conflict(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	handler(w, r, err, http.StatusConflict)
}

// TooManyRequests returns a formatted 429 Too Many Requests error
func TooManyRequests(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	if err == nil {
		err = errors.New("rate limit exceeded")
	}
	handler(w, r, err, http.StatusTooManyRequests)
}

// InternalServerError returns a formatted 500 Internal Server Error
func InternalServerError(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	if err == nil {
		err = errors.New("internal server error")
	}
	handler(w, r, err, http.StatusInternalServerError)
}

// ServiceUnavailable returns a formatted 503 Service Unavailable error
func ServiceUnavailable(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {
	if handler == nil {
		handler = defaultErrorHandler
	}
	if err == nil {
		err = errors.New("service temporarily unavailable")
	}
	handler(w, r, err, http.StatusServiceUnavailable)
}

// RespondWithError sends a JSON error response
func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	errorResponse := map[string]string{
		"error":     message,
		"status":    fmt.Sprintf("%d", statusCode),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	RespondWithJSON(w, statusCode, errorResponse)
}

// RespondWithJSON sends a JSON response
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	// Set content type header
	w.Header().Set("Content-Type", "application/json")

	// Set status code
	w.WriteHeader(statusCode)

	// Convert payload to JSON and write to response
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			// If encoding fails, log and send a basic error
			fmt.Printf("Error encoding JSON response: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to encode response"}`))
		}
	}
}
