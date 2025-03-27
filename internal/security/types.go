// types.go - Security related types

package security

import (
    "context"
    "time"
)

// SecurityEventType defines the type of security event
type SecurityEventType string

// String returns the string representation of SecurityEventType
func (s SecurityEventType) String() string {
    return string(s)
}

const (
    // AuthFailure represents an authentication failure
    AuthFailure SecurityEventType = "auth_failure"
    
    // AccessDenied represents an access denied event
    AccessDenied SecurityEventType = "access_denied"
    
    // BruteForceAttempt represents a brute force attack attempt
    BruteForceAttempt SecurityEventType = "brute_force"
    
    // AnomalyDetected represents an anomaly detection event
    AnomalyDetected SecurityEventType = "anomaly_detected"
    
    // VulnerabilityFound represents a vulnerability detection
    VulnerabilityFound SecurityEventType = "vulnerability_found"
)

// RiskLevel defines the severity of a security risk
type RiskLevel int

const (
    // RiskLow represents a low-risk security event
    RiskLow RiskLevel = iota
    
    // RiskMedium represents a medium-risk security event
    RiskMedium
    
    // RiskHigh represents a high-risk security event
    RiskHigh
    
    // RiskCritical represents a critical-risk security event
    RiskCritical
)

// Event represents a security event
type Event struct {
    ID            string
    Type          SecurityEventType
    Source        string
    Timestamp     time.Time
    UserID        string
    IPAddress     string
    RequestID     string
    ResourceID    string
    Description   string
    RawData       map[string]interface{}
    Tags          map[string]string
    RiskScore     float64
    RiskLevel     RiskLevel
    
    // Additional fields needed for risk analysis
    Severity      RiskLevel      // Severity level of the event
    ClientIP      string         // Client IP address (alias for IPAddress for compatibility)
    RequestPath   string         // Path of the HTTP request if applicable
}

// EnhancedEvent contains an event with additional analysis data
type EnhancedEvent struct {
    Event       Event
    Analysis    AnalysisResult
    PreviousIPs []string
    UserHistory []Event
    Patterns    []string
    RelatedIDs  []string
}

// AnalysisResult contains the result of a risk analysis
type AnalysisResult struct {
    Score       float64
    Level       RiskLevel
    Factors     map[string]float64
    Confidence  float64
    Explanation string
}

// AlertStatus defines the status of an alert
type AlertStatus string

const (
    // AlertStatusNew represents a new alert
    AlertStatusNew AlertStatus = "new"
    
    // AlertStatusAcknowledged represents an acknowledged alert
    AlertStatusAcknowledged AlertStatus = "acknowledged"
    
    // AlertStatusInvestigating represents an alert under investigation
    AlertStatusInvestigating AlertStatus = "investigating"
    
    // AlertStatusResolved represents a resolved alert
    AlertStatusResolved AlertStatus = "resolved"
    
    // AlertStatusFalsePositive represents a false positive alert
    AlertStatusFalsePositive AlertStatus = "false_positive"
)

// Alert represents a security alert
type Alert struct {
    ID          string
    Type        string
    Level       RiskLevel
    Source      string
    Timestamp   time.Time
    Description string
    EventIDs    []string
    Status      AlertStatus
    AssignedTo  string
    Tags        map[string]string
    Metadata    map[string]interface{}
}

// RiskAnalyzer is responsible for analyzing security events and assessing risk
type RiskAnalyzer interface {
    // AnalyzeRisk analyzes a security event and returns a risk score
    AnalyzeRisk(ctx context.Context, event Event, history []Event) (AnalysisResult, error)
    
    // UpdateFactors updates the risk factors based on new data
    UpdateFactors(factors map[string]float64) error
    
    // GetThresholds returns the current risk level thresholds
    GetThresholds() map[RiskLevel]float64
}

// EventProcessor processes security events and enriches them with additional context
type EventProcessor interface {
    // Process enriches an event with additional context
    Process(ctx context.Context, event Event) (EnhancedEvent, error)
    
    // StoreEvent stores an event for future reference
    StoreEvent(event EnhancedEvent) error
    
    // GetHistory retrieves historical events for a given context
    GetHistory(ctx context.Context, filter map[string]interface{}, limit int) ([]Event, error)
}

// AlertManager handles the creation, management, and delivery of security alerts
type AlertManager interface {
    // CreateAlert creates a new alert from a security event
    CreateAlert(ctx context.Context, event EnhancedEvent) (Alert, error)
    
    // UpdateAlertStatus updates the status of an alert
    UpdateAlertStatus(alertID string, status AlertStatus) error
    
    // AssignAlert assigns an alert to a user/team
    AssignAlert(alertID string, assignee string) error
    
    // GetActiveAlerts retrieves active alerts based on filter criteria
    GetActiveAlerts(filter map[string]interface{}) ([]Alert, error)
}

// BruteForceDetectorConfig contains configuration for brute force detection
type BruteForceDetectorConfig struct {
    MaxAttempts        int
    WindowDuration     time.Duration
    LockoutDuration    time.Duration
    ResetAfterSuccess  bool
}

// BruteForceDetector detects brute force attacks
type BruteForceDetector interface {
    // CheckAttempt checks if the current attempt is part of a brute force attack
    CheckAttempt(userID string, resource string, success bool) (bool, error)
    
    // GetAttemptCount gets the number of failed attempts for a user/resource
    GetAttemptCount(userID string, resource string) (int, error)
    
    // Reset resets the attempt counter for a user/resource
    Reset(userID string, resource string) error
}

// AnomalyDetector detects anomalies in user and system behavior
type AnomalyDetector interface {
    // DetectAnomaly checks if an event represents anomalous behavior
    DetectAnomaly(ctx context.Context, event Event, history []Event) (bool, float64, string, error)
    
    // TrainModel trains the anomaly detection model with new data
    TrainModel(events []Event) error
    
    // GetBaseline returns the baseline for a given metric
    GetBaseline(metricType string, dimensions map[string]string) (float64, error)
}

// VulnerabilityDetector detects potential vulnerabilities
type VulnerabilityDetector interface {
    // ScanResource scans a resource for vulnerabilities
    ScanResource(ctx context.Context, resourceType string, resourceID string) ([]Vulnerability, error)
    
    // TrackExploitAttempt records an exploit attempt against a vulnerability
    TrackExploitAttempt(vulnID string, event Event) error
    
    // GetActiveVulnerabilities gets the list of active vulnerabilities
    GetActiveVulnerabilities(resourceType string) ([]Vulnerability, error)
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
    ID               string
    Type             string
    Severity         RiskLevel
    ResourceType     string
    ResourceID       string
    DiscoveredAt     time.Time
    Description      string
    Status           string
    ExploitAttempts  int
    CVE              string
    RemedationSteps  []string
    Metadata         map[string]interface{}
}

// SecurityMetrics contains metrics related to security monitoring
type SecurityMetrics struct {
    EventsProcessed      int64
    HighRiskEvents       int64
    AlertsGenerated      int64
    FalsePositives       int64
    ResponseTime         float64
    VulnerabilitiesFound int64
}

// AnalyzeSecurityEvent analyzes a security event and returns an enhanced version with risk assessment
func AnalyzeSecurityEvent(ctx context.Context, event Event, analyzer RiskAnalyzer, processor EventProcessor) (EnhancedEvent, error) {
    // Process the event to enrich it with additional data
    enhancedEvent, err := processor.Process(ctx, event)
    if err != nil {
        return EnhancedEvent{}, err
    }
    
    // Analyze risk using the processed event
    analysis, err := analyzer.AnalyzeRisk(ctx, event, enhancedEvent.UserHistory)
    if err != nil {
        return EnhancedEvent{}, err
    }
    
    // Update the enhanced event with the analysis results
    enhancedEvent.Analysis = analysis
    
    return enhancedEvent, nil
}

// Detector interface for security event detection
type Detector interface {
    // Detect runs detection and returns security events
    Detect(ctx context.Context) ([]Event, error)
}
