package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// SecurityScanResult represents the scan results from the Python script
type SecurityScanResult struct {
	Timestamp        string           `json:"timestamp"`
	Vulnerabilities  []Vulnerability  `json:"vulnerabilities"`
	SSLInfo         *SSLInfo         `json:"ssl_info"`
	HeadersSecurity *HeadersSecurity `json:"headers_security"`
}

type Vulnerability struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Header      string `json:"header,omitempty"`
}

type SSLInfo struct {
	Version string            `json:"version"`
	Cipher  []string         `json:"cipher"`
	Expiry  string           `json:"expiry"`
	Issuer  map[string]string `json:"issuer"`
}

type HeadersSecurity struct {
	StrictTransportSecurity bool `json:"Strict-Transport-Security"`
	ContentSecurityPolicy   bool `json:"Content-Security-Policy"`
	XFrameOptions          bool `json:"X-Frame-Options"`
	XContentTypeOptions    bool `json:"X-Content-Type-Options"`
	XXSSProtection        bool `json:"X-XSS-Protection"`
}

// PythonBridge manages the interaction with Python scripts
type PythonBridge struct {
	scriptDir string
	mu        sync.Mutex
}

// NewPythonBridge creates a new Python bridge instance
func NewPythonBridge(scriptDir string) (*PythonBridge, error) {
	absPath, err := filepath.Abs(scriptDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script directory path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script directory does not exist: %s", absPath)
	}

	return &PythonBridge{
		scriptDir: absPath,
	}, nil
}

// RunSecurityScan performs a security scan using the Python script
func (p *PythonBridge) RunSecurityScan(scanType string, target string) (*SecurityScanResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	scriptPath := filepath.Join(p.scriptDir, "security_scan.py")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("security scan script not found: %s", scriptPath)
	}

	cmd := exec.Command("python3", scriptPath, scanType, target)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("script failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run script: %v", err)
	}

	var result SecurityScanResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse script output: %v", err)
	}

	return &result, nil
}

// ValidateScriptEnvironment checks if the Python environment is properly set up
func (p *PythonBridge) ValidateScriptEnvironment() error {
	// Check Python version
	cmd := exec.Command("python3", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Python3 not found: %v", err)
	}

	// Check required Python packages
	requiredPackages := []string{"requests"}
	for _, pkg := range requiredPackages {
		cmd := exec.Command("python3", "-c", fmt.Sprintf("import %s", pkg))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("required Python package not found: %s", pkg)
		}
	}

	return nil
}

// CleanupTempFiles removes any temporary files created during script execution
func (p *PythonBridge) CleanupTempFiles() error {
	// Implement cleanup logic if needed
	return nil
}
