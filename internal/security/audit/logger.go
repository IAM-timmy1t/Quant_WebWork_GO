package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EventSeverity represents the severity level of an audit event
type EventSeverity string

const (
	// SeverityInfo is for normal operations
	SeverityInfo EventSeverity = "INFO"
	// SeverityWarning is for events that might indicate a problem
	SeverityWarning EventSeverity = "WARNING"
	// SeverityError is for events that indicate a serious problem
	SeverityError EventSeverity = "ERROR"
	// SeverityCritical is for events that require immediate attention
	SeverityCritical EventSeverity = "CRITICAL"
)

// EventCategory represents the category of an audit event
type EventCategory string

const (
	// CategoryAuthentication is for authentication events
	CategoryAuthentication EventCategory = "AUTHENTICATION"
	// CategoryAuthorization is for authorization events
	CategoryAuthorization EventCategory = "AUTHORIZATION"
	// CategoryDataAccess is for data access events
	CategoryDataAccess EventCategory = "DATA_ACCESS"
	// CategoryConfiguration is for configuration changes
	CategoryConfiguration EventCategory = "CONFIGURATION"
	// CategorySecurity is for general security events
	CategorySecurity EventCategory = "SECURITY"
	// CategorySystem is for system events
	CategorySystem EventCategory = "SYSTEM"
)

// Event represents a security audit event
type Event struct {
	Timestamp   time.Time     `json:"timestamp"`
	UserID      string        `json:"user_id,omitempty"`
	IPAddress   string        `json:"ip_address,omitempty"`
	Action      string        `json:"action"`
	Resource    string        `json:"resource,omitempty"`
	Result      string        `json:"result"` // "success", "failure", etc.
	Category    EventCategory `json:"category"`
	Severity    EventSeverity `json:"severity"`
	Description string        `json:"description"`
	Details     interface{}   `json:"details,omitempty"`
	SessionID   string        `json:"session_id,omitempty"`
	Environment string        `json:"environment"`
	RequestID   string        `json:"request_id,omitempty"`
}

// VerbosityLevel determines how much detail is included in audit logs
type VerbosityLevel string

const (
	// VerbosityOff disables audit logging
	VerbosityOff VerbosityLevel = "off"
	// VerbosityBasic logs only essential events
	VerbosityBasic VerbosityLevel = "basic"
	// VerbosityDetailed logs detailed information for all events
	VerbosityDetailed VerbosityLevel = "detailed"
	// VerbosityVerbose logs everything including debug information
	VerbosityVerbose VerbosityLevel = "verbose"
)

// Config contains configuration for the audit logger
type Config struct {
	// Verbosity level for different environments
	DevelopmentVerbosity VerbosityLevel
	StagingVerbosity     VerbosityLevel
	ProductionVerbosity  VerbosityLevel
	// Whether to log to standard output
	LogToStdout bool
	// Whether to log to a file
	LogToFile bool
	// Path to log file
	LogFilePath string
	// Whether to log to a database
	LogToDatabase bool
	// Database connection string
	DatabaseURL string
}

// DefaultConfig returns a default audit logger configuration
func DefaultConfig() Config {
	return Config{
		DevelopmentVerbosity: VerbosityVerbose,
		StagingVerbosity:     VerbosityDetailed,
		ProductionVerbosity:  VerbosityDetailed,
		LogToStdout:          true,
		LogToFile:            true,
		LogFilePath:          "logs/audit.log",
		LogToDatabase:        false,
	}
}

// Logger handles audit logging
type Logger struct {
	config      Config
	verbosity   VerbosityLevel
	environment security.EnvironmentType
	baseLogger  *zap.SugaredLogger
}

// NewLogger creates a new audit logger
func NewLogger(config Config, baseLogger *zap.SugaredLogger) *Logger {
	// Determine current environment
	environment := security.GetEnvironmentType()

	// Set verbosity based on environment
	var verbosity VerbosityLevel
	switch environment {
	case security.EnvProduction:
		verbosity = config.ProductionVerbosity
	case security.EnvStaging:
		verbosity = config.StagingVerbosity
	default:
		verbosity = config.DevelopmentVerbosity
	}

	return &Logger{
		config:      config,
		verbosity:   verbosity,
		environment: environment,
		baseLogger:  baseLogger.With("component", "audit"),
	}
}

// Log records an audit event with the given details
func (l *Logger) Log(ctx context.Context, event Event) {
	// Skip logging if verbosity is off
	if l.verbosity == VerbosityOff {
		return
	}

	// Set environment
	event.Environment = string(l.environment)

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Extract request ID from context if present
	if requestID, ok := ctx.Value("request_id").(string); ok && event.RequestID == "" {
		event.RequestID = requestID
	}

	// Convert severity to zap level
	var level zapcore.Level
	switch event.Severity {
	case SeverityCritical:
		level = zapcore.FatalLevel
	case SeverityError:
		level = zapcore.ErrorLevel
	case SeverityWarning:
		level = zapcore.WarnLevel
	default:
		level = zapcore.InfoLevel
	}

	// Log to zap with appropriate level
	fields := []interface{}{
		"user_id", event.UserID,
		"ip_address", event.IPAddress,
		"action", event.Action,
		"resource", event.Resource,
		"result", event.Result,
		"category", event.Category,
		"session_id", event.SessionID,
		"request_id", event.RequestID,
	}

	// Add details if verbosity is sufficient and details exist
	if (l.verbosity == VerbosityDetailed || l.verbosity == VerbosityVerbose) && event.Details != nil {
		// Serialize details to JSON if it's a struct
		details, err := json.Marshal(event.Details)
		if err == nil {
			fields = append(fields, "details", string(details))
		}
	}

	switch level {
	case zapcore.FatalLevel:
		l.baseLogger.Errorw(event.Description, fields...)
	case zapcore.ErrorLevel:
		l.baseLogger.Errorw(event.Description, fields...)
	case zapcore.WarnLevel:
		l.baseLogger.Warnw(event.Description, fields...)
	default:
		l.baseLogger.Infow(event.Description, fields...)
	}

	// In a production implementation, we would also:
	// 1. Write to a specialized audit log file
	// 2. Send to a database or external audit system
	// 3. Potentially forward to a security monitoring solution
}

// LogAuthAttempt logs an authentication attempt
func (l *Logger) LogAuthAttempt(ctx context.Context, userID, ipAddress, sessionID string, success bool, details interface{}) {
	result := "failure"
	severity := SeverityWarning
	description := "Failed authentication attempt"

	if success {
		result = "success"
		severity = SeverityInfo
		description = "Successful authentication"
	}

	l.Log(ctx, Event{
		UserID:      userID,
		IPAddress:   ipAddress,
		Action:      "login",
		Result:      result,
		Category:    CategoryAuthentication,
		Severity:    severity,
		Description: description,
		Details:     details,
		SessionID:   sessionID,
	})
}

// LogAccessDenied logs an access denied event
func (l *Logger) LogAccessDenied(ctx context.Context, userID, ipAddress, resource, reason string) {
	l.Log(ctx, Event{
		UserID:      userID,
		IPAddress:   ipAddress,
		Action:      "access",
		Resource:    resource,
		Result:      "denied",
		Category:    CategoryAuthorization,
		Severity:    SeverityWarning,
		Description: "Access denied: " + reason,
	})
}

// LogConfigChange logs a configuration change event
func (l *Logger) LogConfigChange(ctx context.Context, userID, ipAddress, component string, oldValue, newValue interface{}) {
	l.Log(ctx, Event{
		UserID:      userID,
		IPAddress:   ipAddress,
		Action:      "modify",
		Resource:    component,
		Result:      "success",
		Category:    CategoryConfiguration,
		Severity:    SeverityInfo,
		Description: "Configuration changed",
		Details: map[string]interface{}{
			"old_value": oldValue,
			"new_value": newValue,
		},
	})
}

// LogSecurityEvent logs a general security event
func (l *Logger) LogSecurityEvent(ctx context.Context, severity EventSeverity, description string, details interface{}) {
	l.Log(ctx, Event{
		Action:      "security_event",
		Result:      "detected",
		Category:    CategorySecurity,
		Severity:    severity,
		Description: description,
		Details:     details,
	})
}

// Middleware creates HTTP middleware for audit logging
func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if verbosity is too low for HTTP requests
		if l.verbosity == VerbosityOff {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Create a response wrapper to capture status code
		wrapper := newResponseWrapper(w)

		// Extract user information
		userID := "anonymous"
		if user, ok := r.Context().Value("user_id").(string); ok {
			userID = user
		}

		// Get IP address
		ipAddress := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ipAddress = forwardedFor
		}

		// Get request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}

		// Create context with request ID
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		// Serve the request
		next.ServeHTTP(wrapper, r)

		// Skip detailed logging for common and successful requests if verbosity is not high
		if l.verbosity != VerbosityVerbose && wrapper.status >= 200 && wrapper.status < 300 {
			return
		}

		// Determine severity based on status code
		severity := SeverityInfo
		if wrapper.status >= 500 {
			severity = SeverityError
		} else if wrapper.status >= 400 {
			severity = SeverityWarning
		}

		// Create details based on verbosity
		var details interface{}
		if l.verbosity == VerbosityVerbose {
			details = map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"query":      r.URL.RawQuery,
				"user_agent": r.UserAgent(),
				"referer":    r.Referer(),
				"duration":   time.Since(start).Milliseconds(),
				"bytes":      wrapper.written,
			}
		}

		// Log the HTTP request
		l.Log(ctx, Event{
			UserID:      userID,
			IPAddress:   ipAddress,
			Action:      "http_request",
			Resource:    r.URL.Path,
			Result:      fmt.Sprintf("%d", wrapper.status),
			Category:    CategorySystem,
			Severity:    severity,
			Description: fmt.Sprintf("%s %s -> %d", r.Method, r.URL.Path, wrapper.status),
			Details:     details,
			RequestID:   requestID,
		})
	})
}

// responseWrapper wraps http.ResponseWriter to capture status code and bytes written
type responseWrapper struct {
	http.ResponseWriter
	status  int
	written int64
}

// newResponseWrapper creates a new responseWrapper
func newResponseWrapper(w http.ResponseWriter) *responseWrapper {
	return &responseWrapper{ResponseWriter: w, status: http.StatusOK}
}

// WriteHeader captures the status code
func (r *responseWrapper) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write captures the number of bytes written
func (r *responseWrapper) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}
