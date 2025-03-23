// scanner.go - Security vulnerability scanning

package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ScannerConfig defines configuration for the security scanner
type ScannerConfig struct {
	Concurrency      int           // Maximum number of concurrent scans
	Timeout          time.Duration // Maximum time for each scan
	RetryAttempts    int           // Number of retry attempts for each scan
	RetryDelay       time.Duration // Delay between retries
	UserAgent        string        // User-Agent header for HTTP requests
	EnabledScanTypes []string      // List of enabled scan types
	MaxScanDepth     int           // Maximum depth for recursive scans
	PortRanges       []string      // Port ranges for port scanning
	IncludeCommon    bool          // Include common vulnerabilities
}

// DefaultScannerConfig returns a default scanner configuration
func DefaultScannerConfig() ScannerConfig {
	return ScannerConfig{
		Concurrency:   10,
		Timeout:       30 * time.Second,
		RetryAttempts: 2,
		RetryDelay:    5 * time.Second,
		UserAgent:     "QuantWebWorksGO-Scanner/1.0",
		EnabledScanTypes: []string{
			"port_scan",
			"header_analysis",
			"ssl_tls_check",
			"common_vulnerabilities",
			"dependency_check",
		},
		MaxScanDepth:  3,
		PortRanges:    []string{"80-100", "443-443", "8000-8100", "3000-3100"},
		IncludeCommon: true,
	}
}

// Scanner implements security vulnerability scanning
type Scanner struct {
	config      ScannerConfig
	scanModules map[string]ScanModule
	client      *http.Client
	mu          sync.RWMutex
	running     bool
	wg          sync.WaitGroup
	logger      Logger
}

// ScanModule defines the interface for vulnerability scanning modules
type ScanModule interface {
	Name() string
	Scan(ctx context.Context, target string, options map[string]interface{}) ([]Vulnerability, error)
	Description() string
	Category() string
}

// Vulnerability represents a detected security vulnerability
type Vulnerability struct {
	ID          string                 `json:"id"`
	Target      string                 `json:"target"`
	Type        string                 `json:"type"`
	Severity    RiskLevel              `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
	References  []string               `json:"references,omitempty"`
	DetectedAt  time.Time              `json:"detectedAt"`
	Score       float64                `json:"score"`
	CVE         string                 `json:"cve,omitempty"`
}

// ScanResult represents the result of a security scan
type ScanResult struct {
	Target          string          `json:"target"`
	ScanType        string          `json:"scanType"`
	StartTime       time.Time       `json:"startTime"`
	EndTime         time.Time       `json:"endTime"`
	Duration        time.Duration   `json:"duration"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Error           string          `json:"error,omitempty"`
	Status          string          `json:"status"` // success, partial, failed
	ScanID          string          `json:"scanId"`
	Summary         ScanSummary     `json:"summary"`
}

// ScanSummary provides a summary of scan results
type ScanSummary struct {
	TotalVulnerabilities int                `json:"totalVulnerabilities"`
	BySeverity           map[string]int     `json:"bySeverity"`
	ByCategory           map[string]int     `json:"byCategory"`
	HighestScore         float64            `json:"highestScore"`
	AverageScore         float64            `json:"averageScore"`
	ScannedItems         int                `json:"scannedItems"`
	FailedItems          int                `json:"failedItems"`
	Recommendations      []string           `json:"recommendations,omitempty"`
	TopVulnerabilities   []string           `json:"topVulnerabilities,omitempty"`
	MetaData             map[string]string  `json:"metaData,omitempty"`
}

// NewScanner creates a new security scanner
func NewScanner(config ScannerConfig) (*Scanner, error) {
	if config.Concurrency <= 0 {
		config.Concurrency = 5
	}
	
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	
	scanner := &Scanner{
		config:      config,
		scanModules: make(map[string]ScanModule),
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		logger: &defaultLogger{},
	}
	
	// Register default scan modules
	if err := scanner.registerDefaultModules(); err != nil {
		return nil, fmt.Errorf("failed to register default modules: %w", err)
	}
	
	return scanner, nil
}

// SetLogger sets the logger for the scanner
func (s *Scanner) SetLogger(logger Logger) {
	s.logger = logger
}

// RegisterModule registers a custom scan module
func (s *Scanner) RegisterModule(module ScanModule) error {
	if module == nil {
		return fmt.Errorf("module cannot be nil")
	}
	
	name := module.Name()
	if name == "" {
		return fmt.Errorf("module must have a name")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.scanModules[name] = module
	return nil
}

// Scan performs a security scan on a target
func (s *Scanner) Scan(ctx context.Context, target string, scanTypes []string, options map[string]interface{}) (*ScanResult, error) {
	if target == "" {
		return nil, fmt.Errorf("target cannot be empty")
	}
	
	// Normalize target
	normalizedTarget, err := s.normalizeTarget(target)
	if err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}
	
	// Determine scan types to run
	scanTypesToRun := s.determineScanTypes(scanTypes)
	if len(scanTypesToRun) == 0 {
		return nil, fmt.Errorf("no scan types specified or enabled")
	}
	
	result := &ScanResult{
		Target:    normalizedTarget,
		ScanType:  strings.Join(scanTypesToRun, ","),
		StartTime: time.Now(),
		ScanID:    generateID(),
		Status:    "running",
		Summary: ScanSummary{
			BySeverity: make(map[string]int),
			ByCategory: make(map[string]int),
			MetaData:   make(map[string]string),
		},
	}
	
	// Create a separate context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	
	// Run scan modules concurrently
	vulnerabilities, err := s.runScanModules(scanCtx, normalizedTarget, scanTypesToRun, options)
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Vulnerabilities = vulnerabilities
	
	if err != nil {
		result.Error = err.Error()
		result.Status = "failed"
	} else {
		result.Status = "success"
	}
	
	// Generate summary
	s.generateSummary(result)
	
	return result, nil
}

// ScanAsync performs an asynchronous security scan on a target
func (s *Scanner) ScanAsync(ctx context.Context, target string, scanTypes []string, options map[string]interface{}, resultCh chan<- *ScanResult) (string, error) {
	if target == "" {
		return "", fmt.Errorf("target cannot be empty")
	}
	
	if resultCh == nil {
		return "", fmt.Errorf("result channel cannot be nil")
	}
	
	// Normalize target
	normalizedTarget, err := s.normalizeTarget(target)
	if err != nil {
		return "", fmt.Errorf("invalid target: %w", err)
	}
	
	scanID := generateID()
	
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		
		result, err := s.Scan(ctx, normalizedTarget, scanTypes, options)
		if err != nil {
			s.logger.Error("Scan failed", map[string]interface{}{
				"target": normalizedTarget,
				"error":  err.Error(),
				"scanID": scanID,
			})
			
			// Send error result
			resultCh <- &ScanResult{
				Target:    normalizedTarget,
				ScanType:  strings.Join(scanTypes, ","),
				StartTime: time.Now(),
				EndTime:   time.Now(),
				Error:     err.Error(),
				Status:    "failed",
				ScanID:    scanID,
			}
			return
		}
		
		// Update scan ID
		result.ScanID = scanID
		
		// Send result
		resultCh <- result
	}()
	
	return scanID, nil
}

// Wait waits for all scans to complete
func (s *Scanner) Wait() {
	s.wg.Wait()
}

// Stop stops all running scans
func (s *Scanner) Stop() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// ListModules returns a list of available scan modules
func (s *Scanner) ListModules() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	modules := make(map[string]string)
	for name, module := range s.scanModules {
		modules[name] = module.Description()
	}
	
	return modules
}

// normalizeTarget normalizes a scan target
func (s *Scanner) normalizeTarget(target string) (string, error) {
	// Check if it's an IP
	if net.ParseIP(target) != nil {
		return target, nil
	}
	
	// Check if it's a URL
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		parsedURL, err := url.Parse(target)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		return parsedURL.String(), nil
	}
	
	// Check if it's a domain
	if strings.Contains(target, ".") && !strings.Contains(target, "/") {
		return "https://" + target, nil
	}
	
	// Check if it's a file path
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "./") {
		return target, nil
	}
	
	return target, nil
}

// determineScanTypes determines which scan types to run
func (s *Scanner) determineScanTypes(requestedTypes []string) []string {
	if len(requestedTypes) == 0 {
		return s.config.EnabledScanTypes
	}
	
	// Filter requested types by enabled types
	enabledTypeMap := make(map[string]bool)
	for _, t := range s.config.EnabledScanTypes {
		enabledTypeMap[t] = true
	}
	
	var filteredTypes []string
	for _, t := range requestedTypes {
		if enabledTypeMap[t] {
			filteredTypes = append(filteredTypes, t)
		}
	}
	
	return filteredTypes
}

// runScanModules runs all requested scan modules
func (s *Scanner) runScanModules(ctx context.Context, target string, scanTypes []string, options map[string]interface{}) ([]Vulnerability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(scanTypes) == 0 {
		return nil, fmt.Errorf("no scan types specified")
	}
	
	var allVulnerabilities []Vulnerability
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(scanTypes))
	
	// Set up semaphore for concurrency control
	semaphore := make(chan struct{}, s.config.Concurrency)
	
	for _, scanType := range scanTypes {
		module, exists := s.scanModules[scanType]
		if !exists {
			s.logger.Warn("Skipping unknown scan type", map[string]interface{}{
				"scanType": scanType,
				"target":   target,
			})
			continue
		}
		
		wg.Add(1)
		go func(m ScanModule) {
			defer wg.Done()
			
			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			moduleName := m.Name()
			
			s.logger.Info("Starting scan module", map[string]interface{}{
				"module": moduleName,
				"target": target,
			})
			
			vulnerabilities, err := m.Scan(ctx, target, options)
			if err != nil {
				s.logger.Error("Scan module failed", map[string]interface{}{
					"module": moduleName,
					"target": target,
					"error":  err.Error(),
				})
				
				errCh <- fmt.Errorf("module %s failed: %w", moduleName, err)
				return
			}
			
			if len(vulnerabilities) > 0 {
				s.logger.Info("Vulnerabilities found", map[string]interface{}{
					"module": moduleName,
					"count":  len(vulnerabilities),
					"target": target,
				})
				
				mu.Lock()
				allVulnerabilities = append(allVulnerabilities, vulnerabilities...)
				mu.Unlock()
			}
		}(module)
	}
	
	// Wait for all scans to complete
	wg.Wait()
	close(errCh)
	
	// Check for errors
	var errMessages []string
	for err := range errCh {
		errMessages = append(errMessages, err.Error())
	}
	
	if len(errMessages) > 0 {
		return allVulnerabilities, fmt.Errorf("some scans failed: %s", strings.Join(errMessages, "; "))
	}
	
	return allVulnerabilities, nil
}

// generateSummary generates a summary from scan results
func (s *Scanner) generateSummary(result *ScanResult) {
	if result == nil {
		return
	}
	
	summary := &result.Summary
	summary.TotalVulnerabilities = len(result.Vulnerabilities)
	
	var totalScore float64
	for _, vuln := range result.Vulnerabilities {
		// Count by severity
		severity := string(vuln.Severity)
		summary.BySeverity[severity]++
		
		// Count by category
		parts := strings.Split(vuln.Type, ".")
		if len(parts) > 0 {
			category := parts[0]
			summary.ByCategory[category]++
		}
		
		// Track highest score
		if vuln.Score > summary.HighestScore {
			summary.HighestScore = vuln.Score
		}
		
		// Sum scores for average
		totalScore += vuln.Score
		
		// Track top vulnerabilities
		if vuln.Severity == RiskCritical || vuln.Severity == RiskHigh {
			if len(summary.TopVulnerabilities) < 5 {
				summary.TopVulnerabilities = append(summary.TopVulnerabilities, vuln.Title)
			}
		}
	}
	
	// Calculate average score
	if summary.TotalVulnerabilities > 0 {
		summary.AverageScore = totalScore / float64(summary.TotalVulnerabilities)
	}
	
	// Generate recommendations based on vulnerability types
	s.generateRecommendations(result)
}

// generateRecommendations generates recommendations based on vulnerabilities
func (s *Scanner) generateRecommendations(result *ScanResult) {
	if result == nil || len(result.Vulnerabilities) == 0 {
		return
	}
	
	// Count vulnerability types
	typeCount := make(map[string]int)
	for _, vuln := range result.Vulnerabilities {
		typeCount[vuln.Type]++
	}
	
	// Generate recommendations
	var recommendations []string
	
	// Check for SSL/TLS issues
	if typeCount["ssl_tls.weak_cipher"] > 0 || typeCount["ssl_tls.outdated"] > 0 {
		recommendations = append(recommendations, 
			"Update SSL/TLS configuration to use strong ciphers and protocols")
	}
	
	// Check for header issues
	if typeCount["header.missing_security_headers"] > 0 {
		recommendations = append(recommendations, 
			"Implement security headers such as Content-Security-Policy, X-XSS-Protection")
	}
	
	// Check for open ports
	if typeCount["port.unnecessary_open"] > 0 {
		recommendations = append(recommendations, 
			"Close unnecessary open ports to reduce attack surface")
	}
	
	// Check for outdated dependencies
	if typeCount["dependency.outdated"] > 0 {
		recommendations = append(recommendations, 
			"Update outdated dependencies to latest secure versions")
	}
	
	// Check for common vulnerabilities
	if typeCount["common.xss"] > 0 || typeCount["common.injection"] > 0 {
		recommendations = append(recommendations, 
			"Implement input validation and output encoding to prevent injection attacks")
	}
	
	// Limit to top 5 recommendations
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}
	
	result.Summary.Recommendations = recommendations
}

// registerDefaultModules registers the default scan modules
func (s *Scanner) registerDefaultModules() error {
	// Port scanner module
	if err := s.RegisterModule(&PortScanModule{
		scanner: s,
	}); err != nil {
		return err
	}
	
	// Header analysis module
	if err := s.RegisterModule(&HeaderAnalysisModule{
		scanner: s,
	}); err != nil {
		return err
	}
	
	// SSL/TLS check module
	if err := s.RegisterModule(&SSLTLSCheckModule{
		scanner: s,
	}); err != nil {
		return err
	}
	
	// Common vulnerabilities module
	if s.config.IncludeCommon {
		if err := s.RegisterModule(&CommonVulnerabilitiesModule{
			scanner: s,
		}); err != nil {
			return err
		}
	}
	
	// Dependency check module
	if err := s.RegisterModule(&DependencyCheckModule{
		scanner: s,
	}); err != nil {
		return err
	}
	
	return nil
}

// Helper function to generate unique IDs
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Default scan module implementations
type PortScanModule struct {
	scanner *Scanner
}

func (m *PortScanModule) Name() string {
	return "port_scan"
}

func (m *PortScanModule) Description() string {
	return "Scans for open ports on the target"
}

func (m *PortScanModule) Category() string {
	return "network"
}

func (m *PortScanModule) Scan(ctx context.Context, target string, options map[string]interface{}) ([]Vulnerability, error) {
	// Implementation would go here
	// This is a simplified version
	return []Vulnerability{}, nil
}

type HeaderAnalysisModule struct {
	scanner *Scanner
}

func (m *HeaderAnalysisModule) Name() string {
	return "header_analysis"
}

func (m *HeaderAnalysisModule) Description() string {
	return "Analyzes HTTP response headers for security issues"
}

func (m *HeaderAnalysisModule) Category() string {
	return "web"
}

func (m *HeaderAnalysisModule) Scan(ctx context.Context, target string, options map[string]interface{}) ([]Vulnerability, error) {
	// Implementation would go here
	return []Vulnerability{}, nil
}

type SSLTLSCheckModule struct {
	scanner *Scanner
}

func (m *SSLTLSCheckModule) Name() string {
	return "ssl_tls_check"
}

func (m *SSLTLSCheckModule) Description() string {
	return "Checks SSL/TLS configuration for security issues"
}

func (m *SSLTLSCheckModule) Category() string {
	return "crypto"
}

func (m *SSLTLSCheckModule) Scan(ctx context.Context, target string, options map[string]interface{}) ([]Vulnerability, error) {
	// Implementation would go here
	return []Vulnerability{}, nil
}

type CommonVulnerabilitiesModule struct {
	scanner *Scanner
}

func (m *CommonVulnerabilitiesModule) Name() string {
	return "common_vulnerabilities"
}

func (m *CommonVulnerabilitiesModule) Description() string {
	return "Checks for common web vulnerabilities (XSS, CSRF, etc.)"
}

func (m *CommonVulnerabilitiesModule) Category() string {
	return "web"
}

func (m *CommonVulnerabilitiesModule) Scan(ctx context.Context, target string, options map[string]interface{}) ([]Vulnerability, error) {
	// Implementation would go here
	return []Vulnerability{}, nil
}

type DependencyCheckModule struct {
	scanner *Scanner
}

func (m *DependencyCheckModule) Name() string {
	return "dependency_check"
}

func (m *DependencyCheckModule) Description() string {
	return "Checks project dependencies for known vulnerabilities"
}

func (m *DependencyCheckModule) Category() string {
	return "dependencies"
}

func (m *DependencyCheckModule) Scan(ctx context.Context, target string, options map[string]interface{}) ([]Vulnerability, error) {
	// Implementation would go here
	return []Vulnerability{}, nil
}
