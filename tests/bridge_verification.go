// bridge_verification.go - Bridge module verification tests
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/adapters"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge/protocol"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security/risk"
)

// TestBridgeModuleVerification verifies that the bridge module works correctly with
// the React frontend by simulating frontend requests and validating responses.
func TestBridgeModuleVerification(t *testing.T) {
	// Create risk analyzer for testing
	analyzer := risk.NewAnalyzer(risk.Config{
		BaselineRisk: 10,
		MaxRiskScore: 100,
	})

	// Create protocol for testing
	protocolConfig := protocol.StandardProtocolConfig{
		ModelID:            "test-model",
		EnableCompression:  true,
		MetricsEnabled:     true,
		AnalyzerConfig:     map[string]interface{}{"risk_threshold": 0.7},
		DefaultFormat:      protocol.MessageFormatJSON,
		EnableFrontendAPI:  true,
		CORSOrigins:        []string{"http://localhost:3000"},
		ReactEndpoints:     []string{"/api/bridge"},
	}
	
	standardProtocol := protocol.NewStandardProtocol(protocolConfig, nil, analyzer)

	// Create bridge instance
	bridgeInstance := bridge.NewBridge(&bridge.BridgeConfig{
		ID:          "test-bridge",
		Name:        "Test Bridge",
		Description: "Bridge for testing React frontend integration",
		Version:     "1.0.0",
	})

	// Add protocol to bridge
	bridgeInstance.AddProtocol(standardProtocol)

	// Start bridge
	err := bridgeInstance.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start bridge: %v", err)
	}

	// Connect protocol
	err = standardProtocol.Connect(context.Background())
	if err != nil {
		t.Fatalf("Failed to connect protocol: %v", err)
	}

	// Clean up when test finishes
	defer func() {
		standardProtocol.Disconnect(context.Background())
		bridgeInstance.Stop(context.Background())
	}()

	// Run verification tests
	t.Run("TestMessageTypesSupported", testMessageTypesSupported(standardProtocol))
	t.Run("TestAnalysisRequest", testAnalysisRequest(standardProtocol))
	t.Run("TestErrorHandling", testErrorHandling(standardProtocol))
	t.Run("TestMetricsEndpoint", testMetricsEndpoint(standardProtocol))
	t.Run("TestReactCompatibility", testReactCompatibility(standardProtocol))
}

// Test that all required message types are supported
func testMessageTypesSupported(p *protocol.StandardProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Required message types for React frontend
		requiredTypes := []string{
			"connection/init",
			"connection/status",
			"data/request",
			"data/response",
			"system/metrics",
			"system/error",
		}

		for _, msgType := range requiredTypes {
			msg := createTestMessage(msgType)
			if !p.SupportsMessageType(msg.Type) {
				t.Errorf("Protocol does not support required message type: %s", msgType)
			}
		}

		// Verify message format conversion
		msg := createTestMessage("data/request")
		jsonMsg, err := p.ConvertMessageFormat(msg, protocol.MessageFormatJSON)
		if err != nil {
			t.Errorf("Failed to convert message to JSON format: %v", err)
		}

		if jsonMsg.Format != protocol.MessageFormatJSON {
			t.Errorf("Expected message format to be JSON, got: %s", jsonMsg.Format)
		}
	}
}

// Test analysis request handling
func testAnalysisRequest(p *protocol.StandardProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Create data request message
		msg := createTestMessage("data/request")
		msg.Payload = map[string]interface{}{
			"query": "test query",
			"options": map[string]interface{}{
				"full_analysis": true,
			},
		}

		// Process message
		response, err := p.Process(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to process message: %v", err)
		}

		// Verify response
		if response.Type != "data/response" {
			t.Errorf("Expected response type to be data/response, got: %s", response.Type)
		}

		if response.RequestID != msg.RequestID {
			t.Errorf("Expected response requestID to match request, got: %s", response.RequestID)
		}

		// Validate status in payload
		statusValue, ok := response.Payload["status"]
		if !ok {
			t.Errorf("Response missing 'status' field in payload")
		} else if status, ok := statusValue.(string); !ok || status != "success" {
			t.Errorf("Expected status to be 'success', got: %v", statusValue)
		}
	}
}

// Test error handling
func testErrorHandling(p *protocol.StandardProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Create invalid message
		msg := createTestMessage("invalid/type")
		
		// Process message - should result in error
		response, err := p.Process(context.Background(), msg)
		
		// Should return error response, not actual error
		if err != nil {
			t.Fatalf("Expected error to be handled internally, got: %v", err)
		}
		
		// Verify error response
		if response.Type != "system/error" {
			t.Errorf("Expected response type to be system/error, got: %s", response.Type)
		}
		
		// Check error code and message
		errorCode, ok := response.Payload["code"]
		if !ok {
			t.Errorf("Error response missing 'code' field")
		}
		
		errorMsg, ok := response.Payload["message"]
		if !ok {
			t.Errorf("Error response missing 'message' field")
		}
		
		// Print the error details for debugging
		t.Logf("Error code: %v, message: %v", errorCode, errorMsg)
	}
}

// Test metrics endpoint
func testMetricsEndpoint(p *protocol.StandardProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Create metrics request
		msg := createTestMessage("system/metrics")
		
		// Process message
		response, err := p.Process(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to get metrics: %v", err)
		}
		
		// Verify response type
		if response.Type != "system/metrics" {
			t.Errorf("Expected response type to be system/metrics, got: %s", response.Type)
		}
		
		// Verify metrics data in payload
		metricsData, ok := response.Payload["metrics"]
		if !ok {
			t.Errorf("Metrics response missing 'metrics' field")
		}
		
		// Check that we have some metrics data
		metricsMap, ok := metricsData.(map[string]interface{})
		if !ok {
			t.Errorf("Expected metrics to be a map, got: %T", metricsData)
		} else if len(metricsMap) == 0 {
			t.Errorf("Expected metrics data to be non-empty")
		}
	}
}

// Test React compatibility
func testReactCompatibility(p *protocol.StandardProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Simulate React frontend initialization
		initMsg := createTestMessage("connection/init")
		initMsg.Payload = map[string]interface{}{
			"client": "react-frontend",
			"version": "1.0.0",
			"capabilities": []string{
				"compression",
				"streaming",
				"metrics",
			},
		}
		
		// Process initialization
		response, err := p.Process(context.Background(), initMsg)
		if err != nil {
			t.Fatalf("Failed to initialize connection: %v", err)
		}
		
		// Verify connection status
		if response.Type != "connection/status" {
			t.Errorf("Expected response type to be connection/status, got: %s", response.Type)
		}
		
		statusValue, ok := response.Payload["status"]
		if !ok {
			t.Errorf("Connection status missing 'status' field")
		} else if status, ok := statusValue.(string); !ok || status != "connected" {
			t.Errorf("Expected connection status to be 'connected', got: %v", statusValue)
		}
		
		// Test React data request flow
		dataMsg := createTestMessage("data/request")
		dataMsg.Payload = map[string]interface{}{
			"action": "get_data",
			"params": map[string]interface{}{
				"id": "test-id",
			},
		}
		
		// Process data request
		dataResponse, err := p.Process(context.Background(), dataMsg)
		if err != nil {
			t.Fatalf("Failed to process data request: %v", err)
		}
		
		// Verify data response
		if dataResponse.Type != "data/response" {
			t.Errorf("Expected response type to be data/response, got: %s", dataResponse.Type)
		}
		
		// Verify data payload exists
		_, ok = dataResponse.Payload["data"]
		if !ok {
			t.Errorf("Data response missing 'data' field")
		}
	}
}

// Helper to create test messages
func createTestMessage(msgType string) protocol.Message {
	return protocol.Message{
		Type:      msgType,
		RequestID: fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
		Format:    protocol.MessageFormatJSON,
		Payload:   map[string]interface{}{},
	}
}

// RunManualVerification provides a simple way to manually verify bridge operations
// This can be run as a standalone executable for development and debugging
func RunManualVerification() {
	// Create risk analyzer for testing
	analyzer := risk.NewAnalyzer(risk.Config{
		BaselineRisk: 10,
		MaxRiskScore: 100,
	})

	// Create protocol for testing
	protocolConfig := protocol.StandardProtocolConfig{
		ModelID:            "test-model",
		EnableCompression:  true,
		MetricsEnabled:     true,
		AnalyzerConfig:     map[string]interface{}{"risk_threshold": 0.7},
		DefaultFormat:      protocol.MessageFormatJSON,
		EnableFrontendAPI:  true,
		CORSOrigins:        []string{"http://localhost:3000"},
		ReactEndpoints:     []string{"/api/bridge"},
	}
	
	standardProtocol := protocol.NewStandardProtocol(protocolConfig, nil, analyzer)

	// Create bridge instance
	bridgeInstance := bridge.NewBridge(&bridge.BridgeConfig{
		ID:          "test-bridge",
		Name:        "Test Bridge",
		Description: "Bridge for manual verification",
		Version:     "1.0.0",
	})

	// Add protocol to bridge
	bridgeInstance.AddProtocol(standardProtocol)

	// Setup HTTP adapter
	httpAdapter := adapters.NewHTTPAdapter(&adapters.HTTPAdapterConfig{
		Port:            8080,
		EnableCORS:      true,
		AllowedOrigins:  []string{"*"},
		AllowCredentials: true,
	})

	// Add adapter to bridge
	bridgeInstance.AddAdapter(httpAdapter)

	// Start bridge
	ctx := context.Background()
	err := bridgeInstance.Start(ctx)
	if err != nil {
		fmt.Printf("Failed to start bridge: %v\n", err)
		os.Exit(1)
	}

	// Connect protocol
	err = standardProtocol.Connect(ctx)
	if err != nil {
		fmt.Printf("Failed to connect protocol: %v\n", err)
		os.Exit(1)
	}

	// Print information
	fmt.Println("Bridge is running for manual verification")
	fmt.Println("Bridge ID:", bridgeInstance.GetID())
	fmt.Println("Bridge Version:", bridgeInstance.GetVersion())
	fmt.Println("HTTP Adapter listening on port 8080")
	fmt.Println("Available endpoints:")
	fmt.Println("- http://localhost:8080/api/bridge")
	fmt.Println("- http://localhost:8080/metrics")
	fmt.Println("- http://localhost:8080/health")
	fmt.Println("\nPress Ctrl+C to stop the bridge")

	// Wait for interrupt signal
	<-ctx.Done()

	// Cleanup
	standardProtocol.Disconnect(ctx)
	bridgeInstance.Stop(ctx)
}

// For manual verification, can be built as a standalone executable
func main() {
	// Only run main() when explicitly built as an executable
	if os.Getenv("GO_BRIDGE_VERIFIER") == "1" {
		RunManualVerification()
	}
}
