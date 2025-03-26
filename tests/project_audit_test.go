// tests/project_audit_test.go

package tests

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FileSpec defines requirements for a specific file
type FileSpec struct {
	Path             string     // Relative path to the file
	RequiredImports  []string   // Import statements that must be present
	RequiredPatterns []string   // Function signatures, type declarations, etc. that must be present
	Optional         bool       // Whether this file is optional (won't fail test if missing)
}

// ProjectStructure contains the expected file structure
func getProjectStructure() []FileSpec {
	return []FileSpec{
		// Core Infrastructure
		{
			Path: "cmd/server/main.go",
			RequiredImports: []string{
				"context",
				"flag",
				"fmt",
				"net/http",
				"os",
				"os/signal",
				"syscall",
				"time",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/api/rest",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security",
			},
			RequiredPatterns: []string{
				"func main()",
				"flag.Parse()",
				"LoadConfig",
				"NewRouter",
				"server.Shutdown",
			},
		},
		{
			Path: "internal/core/config/manager.go",
			RequiredImports: []string{
				"time",
				"github.com/spf13/viper",
			},
			RequiredPatterns: []string{
				"type Config struct",
				"type ServerConfig struct",
				"type SecurityConfig struct",
				"func LoadConfig",
				"func setDefaults",
			},
		},
		{
			Path: "internal/core/config/file_provider.go",
			RequiredImports: []string{
				"os",
				"path/filepath",
			},
			RequiredPatterns: []string{
				"type FileProvider struct",
				"func NewFileProvider",
				"func (*FileProvider) ReadFile",
				"func (*FileProvider) WriteFile",
			},
		},
		{
			Path: "internal/api/rest/router.go",
			RequiredImports: []string{
				"net/http",
				"github.com/gorilla/mux",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics",
			},
			RequiredPatterns: []string{
				"func NewRouter",
			},
		},
		{
			Path: "internal/api/rest/middleware.go",
			RequiredImports: []string{
				"net/http",
				"time",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"func LoggingMiddleware",
				"func MetricsMiddleware",
				"func RateLimitMiddleware",
			},
		},
		{
			Path: "internal/api/rest/error_handler.go",
			RequiredImports: []string{
				"encoding/json",
				"net/http",
			},
			RequiredPatterns: []string{
				"type ErrorResponse struct",
				"func RespondWithError",
				"func RespondWithJSON",
			},
		},
		
		// GraphQL Components (Optional)
		{
			Path:     "internal/api/graphql/resolver.go",
			Optional: true,
			RequiredImports: []string{
				"github.com/graphql-go/graphql",
			},
		},
		{
			Path:     "internal/api/graphql/schema.go",
			Optional: true,
			RequiredImports: []string{
				"github.com/graphql-go/graphql",
			},
		},
		
		// Bridge System
		{
			Path: "internal/bridge/bridge.go",
			RequiredImports: []string{
				"context",
				"errors",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"type Bridge interface",
				"type Message struct",
				"type MessageHandler",
				"func NewBridge",
			},
		},
		{
			Path: "internal/bridge/manager.go",
			RequiredImports: []string{
				"context",
				"fmt",
				"sync",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/discovery",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics",
			},
			RequiredPatterns: []string{
				"type Manager struct",
				"func NewManager",
				"func (*Manager) Start",
				"func (*Manager) Stop",
				"func (*Manager) CreateBridge",
			},
		},
		{
			Path: "internal/bridge/adapters/adapter.go",
			RequiredImports: []string{
				"context",
			},
			RequiredPatterns: []string{
				"type Adapter interface",
				"type AdapterConfig struct",
				"type AdapterFactory",
				"func RegisterAdapterFactory",
				"func GetAdapterFactory",
			},
		},
		{
			Path: "internal/bridge/adapters/grpc_adapter.go",
			RequiredImports: []string{
				"context",
				"fmt",
				"time",
				"google.golang.org/grpc",
			},
			RequiredPatterns: []string{
				"type GRPCAdapter struct",
				"func NewGRPCAdapter",
			},
		},
		{
			Path: "internal/bridge/adapters/websocket_adapter.go",
			RequiredImports: []string{
				"context",
				"fmt",
				"net/url",
				"time",
				"github.com/gorilla/websocket",
			},
			RequiredPatterns: []string{
				"type WebSocketAdapter struct",
				"func NewWebSocketAdapter",
			},
		},
		{
			Path: "internal/bridge/connection_pool.go",
			RequiredImports: []string{
				"context",
				"fmt",
				"sync",
				"time",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters",
			},
			RequiredPatterns: []string{
				"type ConnectionPool struct",
				"type PoolConfig struct",
				"func NewConnectionPool",
				"func (*ConnectionPool) Acquire",
				"func (*ConnectionPool) Return",
			},
		},
		{
			Path: "internal/core/discovery/service.go",
			RequiredImports: []string{
				"context",
				"time",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config",
			},
			RequiredPatterns: []string{
				"type Service struct",
				"func NewService",
				"func (*Service) Start",
				"func (*Service) RegisterService",
			},
		},
		
		// Security Components
		{
			Path: "internal/security/env_security.go",
			RequiredImports: []string{
				"errors",
				"os",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"type EnvironmentType",
				"type SecurityConfig struct",
				"func GetEnvironmentType",
				"func GetSecurityConfig",
				"func ValidateProductionSecurity",
			},
		},
		{
			Path: "internal/security/firewall/firewall.go",
			RequiredImports: []string{
				"fmt",
				"net",
				"sync",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"type Firewall struct",
				"type Rule struct",
				"func NewFirewall",
				"func (*Firewall) AddRule",
			},
		},
		{
			Path: "internal/security/firewall/rate_limiter.go",
			RequiredImports: []string{
				"net",
				"sync",
				"time",
				"golang.org/x/time/rate",
			},
			RequiredPatterns: []string{
				"type RateLimiter struct",
				"func NewRateLimiter",
				"func (*RateLimiter) GetLimiter",
			},
		},
		{
			Path: "internal/security/firewall/advanced_rate_limiter.go",
			RequiredImports: []string{
				"net",
				"sync",
				"time",
				"golang.org/x/time/rate",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics",
			},
			RequiredPatterns: []string{
				"type AdvancedRateLimiter struct",
				"func NewAdvancedRateLimiter",
				"func (*AdvancedRateLimiter) Allow",
			},
		},
		{
			Path: "internal/security/ipmasking/manager.go",
			RequiredImports: []string{
				"net",
				"sync",
				"time",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"type Manager struct",
				"func NewManager",
				"func (*Manager) Start",
				"func (*Manager) GetMaskedIP",
			},
		},
		{
			Path: "internal/security/risk/analyzer.go",
			RequiredImports: []string{
				"time",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"type Analyzer struct",
				"func NewAnalyzer",
				"func (*Analyzer) AnalyzeRequest",
			},
		},
		
		// Metrics Components
		{
			Path: "internal/core/metrics/collector.go",
			RequiredImports: []string{
				"time",
				"github.com/prometheus/client_golang/prometheus",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"type Collector struct",
				"func NewCollector",
				"func (*Collector) RecordHTTPRequest",
			},
		},
		{
			Path: "internal/core/metrics/prometheus.go",
			RequiredImports: []string{
				"github.com/prometheus/client_golang/prometheus",
			},
			RequiredPatterns: []string{
				"type BridgeRequestCounter struct",
				"type BridgeLatencyHistogram struct",
			},
		},
		{
			Path: "internal/core/metrics/adaptive_collection.go",
			RequiredImports: []string{
				"sync/atomic",
				"time",
				"github.com/prometheus/client_golang/prometheus",
				"go.uber.org/zap",
			},
			RequiredPatterns: []string{
				"type CollectionMode",
				"type AdaptiveCollector struct",
				"func NewAdaptiveCollector",
				"func (*AdaptiveCollector) UpdateConnectionCount",
			},
		},
		
		// Frontend Components
		{
			Path: "web/client/src/App.tsx",
			RequiredImports: []string{
				"React",
				"BrowserRouter",
			},
			RequiredPatterns: []string{
				"const App",
				"<Router>",
				"<Routes>",
			},
			Optional: true,
		},
		{
			Path: "web/client/src/bridge/BridgeClient.ts",
			RequiredImports: []string{
				"EventEmitter",
			},
			RequiredPatterns: []string{
				"interface BridgeOptions",
				"interface BridgeMessage",
				"class BridgeClient",
				"connect()",
				"send(",
			},
			Optional: true,
		},
		{
			Path: "web/client/src/components/Onboarding/OnboardingWizard.tsx",
			RequiredImports: []string{
				"React",
				"useState",
				"useEffect",
			},
			RequiredPatterns: []string{
				"OnboardingWizard",
				"SecurityCheck",
				"BridgeSetup",
				"AdminSetup",
			},
			Optional: true,
		},
		
		// Tests
		{
			Path: "tests/bridge_verification.go",
			RequiredImports: []string{
				"context",
				"flag",
				"fmt",
				"os",
				"time",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge",
				"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters",
			},
			RequiredPatterns: []string{
				"type TestResult struct",
				"func main()",
			},
			Optional: true,
		},
		{
			Path: "tests/load/load_test.go",
			RequiredImports: []string{
				"context",
				"fmt",
				"sync",
				"testing",
				"time",
			},
			RequiredPatterns: []string{
				"type TestConfig struct",
				"type TestResult struct",
				"func RunLoadTest",
			},
			Optional: true,
		},
		
		// Deployment Configs
		{
			Path:     "deployments/k8s/prod/deployment.yaml",
			Optional: true,
		},
		{
			Path:     "deployments/k8s/prod/service.yaml",
			Optional: true,
		},
		{
			Path:     "deployments/monitoring/grafana/provisioning/dashboards/bridge-dashboard.json",
			Optional: true,
		},
		{
			Path:     "deployments/monitoring/prometheus/prometheus.yml",
			Optional: true,
		},
	}
}

// TestProjectStructure verifies all expected files are present and contain required components
func TestProjectStructure(t *testing.T) {
	// Get the root directory of the project
	rootDir, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	
	// Get the expected project structure
	fileSpecs := getProjectStructure()
	
	// Check each file specification
	for _, spec := range fileSpecs {
		filePath := filepath.Join(rootDir, spec.Path)
		
		// Check if file exists
		fileExists, err := fileExists(filePath)
		if err != nil {
			t.Errorf("Error checking file %s: %v", spec.Path, err)
			continue
		}
		
		if !fileExists {
			if spec.Optional {
				t.Logf("Optional file not found: %s", spec.Path)
				continue
			}
			t.Errorf("Required file not found: %s", spec.Path)
			continue
		}
		
		// If file exists and has requirements, check them
		if len(spec.RequiredImports) > 0 || len(spec.RequiredPatterns) > 0 {
			content, err := readFileContent(filePath)
			if err != nil {
				t.Errorf("Error reading file %s: %v", spec.Path, err)
				continue
			}
			
			// Check required imports
			for _, imp := range spec.RequiredImports {
				if !containsImport(content, imp) {
					t.Errorf("File %s missing required import: %s", spec.Path, imp)
				}
			}
			
			// Check required patterns
			for _, pattern := range spec.RequiredPatterns {
				if !strings.Contains(content, pattern) {
					t.Errorf("File %s missing required pattern: %s", spec.Path, pattern)
				}
			}
		}
	}
}

// findProjectRoot attempts to find the root directory of the project
func findProjectRoot() (string, error) {
	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	
	// Check for common root indicators
	for {
		// Check if go.mod exists
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		
		// Check if we've reached the root directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	
	// If we couldn't find a definitive root, return the current directory
	return os.Getwd()
}

// fileExists checks if a file exists
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// readFileContent reads the content of a file
func readFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// containsImport checks if a file contains a specific import
func containsImport(content, importPath string) bool {
	// Look for import statements in different formats
	scanner := bufio.NewScanner(strings.NewReader(content))
	inImportBlock := false
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Check for single import statements
		if strings.HasPrefix(line, "import ") && strings.Contains(line, importPath) {
			return true
		}
		
		// Check for import blocks
		if line == "import (" {
			inImportBlock = true
			continue
		}
		
		if inImportBlock {
			if line == ")" {
				inImportBlock = false
				continue
			}
			
			// Extract the import path from the line
			importLine := strings.TrimSpace(line)
			if strings.HasPrefix(importLine, "\"") && strings.HasSuffix(importLine, "\"") {
				// Simple import "path"
				extractedPath := importLine[1 : len(importLine)-1]
				if extractedPath == importPath {
					return true
				}
			} else if strings.Contains(importLine, "\"") {
				// Aliased import name "path"
				parts := strings.Split(importLine, "\"")
				if len(parts) >= 3 && parts[1] == importPath {
					return true
				}
			}
		}
	}
	
	return false
}

// TestGenerateImplementationReport generates a report of implementation status
func TestGenerateImplementationReport(t *testing.T) {
	if os.Getenv("GENERATE_REPORT") != "true" {
		t.Skip("Skipping report generation. Set GENERATE_REPORT=true to enable.")
	}
	
	// Get the root directory of the project
	rootDir, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	
	// Get the expected project structure
	fileSpecs := getProjectStructure()
	
	// Create a report file
	reportFile, err := os.Create(filepath.Join(rootDir, "implementation_report.md"))
	if err != nil {
		t.Fatalf("Failed to create report file: %v", err)
	}
	defer reportFile.Close()
	
	// Write report header
	reportFile.WriteString("# QUANT_WebWork_GO Implementation Status Report\n\n")
	reportFile.WriteString("| Component | Status | Missing Imports | Missing Patterns |\n")
	reportFile.WriteString("|-----------|--------|-----------------|------------------|\n")
	
	// Check each file specification
	for _, spec := range fileSpecs {
		filePath := filepath.Join(rootDir, spec.Path)
		fileExists, _ := fileExists(filePath)
		
		status := "✅ Implemented"
		missingImports := []string{}
		missingPatterns := []string{}
		
		if !fileExists {
			if spec.Optional {
				status = "⚠️ Optional (Not Found)"
			} else {
				status = "❌ Missing"
			}
		} else if len(spec.RequiredImports) > 0 || len(spec.RequiredPatterns) > 0 {
			content, _ := readFileContent(filePath)
			
			// Check required imports
			for _, imp := range spec.RequiredImports {
				if !containsImport(content, imp) {
					missingImports = append(missingImports, imp)
				}
			}
			
			// Check required patterns
			for _, pattern := range spec.RequiredPatterns {
				if !strings.Contains(content, pattern) {
					missingPatterns = append(missingPatterns, pattern)
				}
			}
			
			if len(missingImports) > 0 || len(missingPatterns) > 0 {
				status = "⚠️ Incomplete"
			}
		}
		
		// Format missing imports and patterns for the report
		missingImportsStr := strings.Join(missingImports, "<br>")
		missingPatternsStr := strings.Join(missingPatterns, "<br>")
		
		// Write the component's status to the report
		reportFile.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", 
			spec.Path,
			status,
			missingImportsStr,
			missingPatternsStr,
		))
	}
	
	t.Logf("Implementation report generated at: %s", filepath.Join(rootDir, "implementation_report.md"))
}
