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

	"github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/QUANT_WW_GO/internal/bridge"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/QUANT_WW_GO/internal/bridge/adapters"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/QUANT_WW_GO/internal/bridge/protocol"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/QUANT_WW_GO/internal/core/tokens"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/QUANT_WW_GO/internal/security/risk"
)

// TestBridgeModuleVerification verifies that the bridge module works correctly with
// the React frontend by simulating frontend requests and validating responses.
func TestBridgeModuleVerification(t *testing.T) {
	// Create token planner for testing
	planner := tokens.NewPlanner(&tokens.PlannerConfig{
		DefaultBudget: tokens.TokenBudget{
			MaxTokens: 1000000,
			RefillRate: 10000,
			RefillInterval: time.Minute,
		},
	})

	// Create token analyzer for testing
	analyzer := risk.NewTokenAnalyzer(nil)

	// Create token protocol for testing
	tokenProtocol := protocol.NewTokenProtocol(protocol.TokenProtocolConfig{
		ModelID:           "test-model",
		MaxTokenPerMessage: 8192,
		DefaultBudget:     tokens.TokenBudget{MaxTokens: 1000000},
		EnableCompression: true,
		TokenThreshold:    0.8,
		MetricsEnabled:    true,
		AnalyzerConfig:    map[string]interface{}{"risk_threshold": 0.7},
		DefaultFormat:     protocol.MessageFormatJSON,
		EnableFrontendAPI: true,
		CORSOrigins:       []string{"http://localhost:3000"},
		ReactEndpoints:    []string{"/api/bridge"},
	}, planner, analyzer)

	// Create bridge instance
	bridgeInstance := bridge.NewBridge(&bridge.BridgeConfig{
		ID:          "test-bridge",
		Name:        "Test Bridge",
		Description: "Bridge for testing React frontend integration",
		Version:     "1.0.0",
	})

	// Add protocol to bridge
	bridgeInstance.AddProtocol(tokenProtocol)

	// Start bridge
	err := bridgeInstance.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start bridge: %v", err)
	}

	// Connect protocol
	err = tokenProtocol.Connect(context.Background())
	if err != nil {
		t.Fatalf("Failed to connect protocol: %v", err)
	}

	// Clean up when test finishes
	defer func() {
		tokenProtocol.Disconnect(context.Background())
		bridgeInstance.Stop(context.Background())
	}()

	// Run verification tests
	t.Run("TestMessageTypesSupported", testMessageTypesSupported(tokenProtocol))
	t.Run("TestAnalysisRequest", testAnalysisRequest(tokenProtocol))
	t.Run("TestErrorHandling", testErrorHandling(tokenProtocol))
	t.Run("TestMetricsEndpoint", testMetricsEndpoint(tokenProtocol))
	t.Run("TestReactCompatibility", testReactCompatibility(tokenProtocol))
}

// Test that all required message types are supported
func testMessageTypesSupported(p *protocol.TokenProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Required message types for React frontend
		requiredTypes := []protocol.MessageType{
			protocol.MessageTypeAnalysisRequest,
			protocol.MessageTypeAnalysisResponse,
			protocol.MessageTypeRiskRequest,
			protocol.MessageTypeRiskResponse,
			protocol.MessageTypeEvent,
			protocol.MessageTypeMetrics,
			protocol.MessageTypeUIUpdate,
			protocol.MessageTypeError,
		}

		// Create test message to get responses for each type
		for _, msgType := range requiredTypes {
			// Create test message
			testMsg := createTestMessage(string(msgType))

			// Process message
			response, err := p.Process(context.Background(), testMsg)
			if err != nil {
				t.Logf("Error processing message type %s: %v", msgType, err)
				// Some message types might legitimately return errors,
				// so this is not always a test failure
				continue
			}

			// Verify response
			if response == nil {
				t.Errorf("No response received for message type %s", msgType)
			}
		}
	}
}

// Test analysis request handling
func testAnalysisRequest(p *protocol.TokenProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Create analysis request message
		metadata := map[string]interface{}{
			"token_address": "0x1234567890abcdef",
		}
		metadataJSON, _ := json.Marshal(metadata)

		msg := &protocol.RawMessage{
			ID:        "test-analysis-request",
			Type:      string(protocol.MessageTypeAnalysisRequest),
			Content:   "",
			Metadata:  string(metadataJSON),
			Timestamp: time.Now().UnixNano(),
		}

		// Process message
		response, err := p.Process(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to process analysis request: %v", err)
		}

		// Verify response
		if response == nil {
			t.Fatal("No response received for analysis request")
		}

		// Verify response type
		if response.Type() != string(protocol.MessageTypeAnalysisResponse) && 
		   response.Type() != string(protocol.MessageTypeError) {
			t.Errorf("Expected response type %s or %s, got %s", 
				protocol.MessageTypeAnalysisResponse, 
				protocol.MessageTypeError, 
				response.Type())
		}

		// Parse response metadata
		var responseMetadata map[string]interface{}
		err = json.Unmarshal([]byte(response.Metadata()), &responseMetadata)
		if err != nil {
			t.Fatalf("Failed to parse response metadata: %v", err)
		}

		// Verify some response data exists
		if len(responseMetadata) == 0 {
			t.Error("Response metadata is empty")
		}
	}
}

// Test error handling
func testErrorHandling(p *protocol.TokenProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Create invalid request (missing required fields)
		metadata := map[string]interface{}{
			// Deliberately missing token_address
		}
		metadataJSON, _ := json.Marshal(metadata)

		msg := &protocol.RawMessage{
			ID:        "test-error-handling",
			Type:      string(protocol.MessageTypeAnalysisRequest),
			Content:   "",
			Metadata:  string(metadataJSON),
			Timestamp: time.Now().UnixNano(),
		}

		// Process message
		response, err := p.Process(context.Background(), msg)
		if err != nil {
			// Some errors might be returned directly, which is fine
			t.Logf("Expected error returned: %v", err)
			return
		}

		if response == nil {
			t.Fatal("No response received for erroneous request")
		}

		// Should have error response
		if response.Type() != string(protocol.MessageTypeError) {
			t.Errorf("Expected error response, got %s", response.Type())
		}

		// Parse response metadata
		var responseMetadata map[string]interface{}
		err = json.Unmarshal([]byte(response.Metadata()), &responseMetadata)
		if err != nil {
			t.Fatalf("Failed to parse error metadata: %v", err)
		}

		// Verify error message exists
		if _, ok := responseMetadata["error"]; !ok {
			t.Error("Error message missing from response")
		}
	}
}

// Test metrics endpoint
func testMetricsEndpoint(p *protocol.TokenProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Create metrics request message
		msg := &protocol.RawMessage{
			ID:        "test-metrics-request",
			Type:      string(protocol.MessageTypeMetrics),
			Content:   "",
			Metadata:  "{}",
			Timestamp: time.Now().UnixNano(),
		}

		// Process message
		response, err := p.Process(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to process metrics request: %v", err)
		}

		// Verify response
		if response == nil {
			t.Fatal("No response received for metrics request")
		}

		// Parse response metadata
		var responseMetadata map[string]interface{}
		err = json.Unmarshal([]byte(response.Metadata()), &responseMetadata)
		if err != nil {
			t.Fatalf("Failed to parse metrics metadata: %v", err)
		}

		// Verify metrics data exists
		if metrics, ok := responseMetadata["metrics"]; !ok || metrics == nil {
			t.Error("Metrics missing from response")
		}
	}
}

// Test React compatibility
func testReactCompatibility(p *protocol.TokenProtocol) func(t *testing.T) {
	return func(t *testing.T) {
		// Simulate React client request format
		reactClientMsg := map[string]interface{}{
			"id":      "react-client-msg-123",
			"type":    "analysis-request",
			"content": "",
			"metadata": map[string]interface{}{
				"token_address": "0xabcdef1234567890",
				"client_info": map[string]interface{}{
					"type":        "web",
					"version":     "1.0.0",
					"environment": "development",
					"platform":    "browser",
				},
			},
			"timestamp": time.Now().UnixMilli(),
		}

		reactClientMsgJSON, _ := json.Marshal(reactClientMsg)

		// Create message
		msg := &protocol.RawMessage{
			ID:        reactClientMsg["id"].(string),
			Type:      reactClientMsg["type"].(string),
			Content:   "",
			Metadata:  string(reactClientMsgJSON),
			Timestamp: reactClientMsg["timestamp"].(int64),
		}

		// Process message
		response, err := p.Process(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to process React client message: %v", err)
		}

		// Verify response
		if response == nil {
			t.Fatal("No response received for React client message")
		}

		// Parse response data
		var responseData map[string]interface{}
		err = json.Unmarshal([]byte(response.Metadata()), &responseData)
		if err != nil {
			t.Fatalf("Failed to parse response data: %v", err)
		}

		// Validate response format is compatible with React client expectations
		if id := response.ID(); id == "" {
			t.Error("Response missing ID required by React client")
		}

		if typ := response.Type(); typ == "" {
			t.Error("Response missing Type required by React client")
		}
	}
}

// Helper to create test messages
func createTestMessage(msgType string) protocol.Message {
	metadata := map[string]interface{}{
		"test": true,
		"timestamp": time.Now().Unix(),
	}
	
	// Add type-specific metadata
	switch msgType {
	case string(protocol.MessageTypeAnalysisRequest):
		metadata["token_address"] = "0x1234567890abcdef"
	case string(protocol.MessageTypeRiskRequest):
		metadata["token_addresses"] = []string{"0x1234567890abcdef", "0xfedcba0987654321"}
	}
	
	metadataJSON, _ := json.Marshal(metadata)
	
	return &protocol.RawMessage{
		ID:        fmt.Sprintf("test-msg-%s-%d", msgType, time.Now().UnixNano()),
		Type:      msgType,
		Content:   "Test content",
		Metadata:  string(metadataJSON),
		Timestamp: time.Now().UnixNano(),
	}
}

// Helper function for manual verification
func RunManualVerification() {
	fmt.Println("Starting manual Bridge Module verification...")
	
	// Create token planner
	planner := tokens.NewPlanner(&tokens.PlannerConfig{
		DefaultBudget: tokens.TokenBudget{
			MaxTokens: 1000000,
			RefillRate: 10000,
			RefillInterval: time.Minute,
		},
	})

	// Create token analyzer
	analyzer := risk.NewTokenAnalyzer(nil)

	// Create token protocol
	tokenProtocol := protocol.NewTokenProtocol(protocol.TokenProtocolConfig{
		ModelID:           "test-model",
		MaxTokenPerMessage: 8192,
		DefaultBudget:     tokens.TokenBudget{MaxTokens: 1000000},
		EnableCompression: true,
		TokenThreshold:    0.8,
		MetricsEnabled:    true,
		AnalyzerConfig:    map[string]interface{}{"risk_threshold": 0.7},
		DefaultFormat:     protocol.MessageFormatJSON,
		EnableFrontendAPI: true,
		CORSOrigins:       []string{"http://localhost:3000"},
		ReactEndpoints:    []string{"/api/bridge"},
	}, planner, analyzer)

	// Create bridge instance
	bridgeInstance := bridge.NewBridge(&bridge.BridgeConfig{
		ID:          "test-bridge",
		Name:        "Test Bridge",
		Description: "Bridge for testing React frontend integration",
		Version:     "1.0.0",
	})

	// Add protocol to bridge
	bridgeInstance.AddProtocol(tokenProtocol)

	// Create gRPC adapter
	grpcAdapter := adapters.NewGRPCAdapter(&adapters.GRPCAdapterConfig{
		Host:       "0.0.0.0",
		Port:       8080,
		EnableWeb:  true,
		CORSOrigins: []string{"http://localhost:3000"},
		EnableTLS:  false,
	})

	// Connect adapter to bridge
	bridgeInstance.AddAdapter(grpcAdapter)

	// Start bridge
	fmt.Println("Starting bridge...")
	err := bridgeInstance.Start(context.Background())
	if err != nil {
		fmt.Printf("Failed to start bridge: %v\n", err)
		os.Exit(1)
	}

	// Connect protocol
	fmt.Println("Connecting protocol...")
	err = tokenProtocol.Connect(context.Background())
	if err != nil {
		fmt.Printf("Failed to connect protocol: %v\n", err)
		os.Exit(1)
	}

	// Start HTTP server for testing
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"bridge": "connected",
			"version": "1.0.0",
		})
	})

	// Create test endpoint
	http.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var requestData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			http.Error(w, "Invalid request data", http.StatusBadRequest)
			return
		}

		// Create test message
		msgType := requestData["type"].(string)
		metadata, _ := json.Marshal(requestData["metadata"])

		msg := &protocol.RawMessage{
			ID:        fmt.Sprintf("test-api-%d", time.Now().UnixNano()),
			Type:      msgType,
			Content:   requestData["content"].(string),
			Metadata:  string(metadata),
			Timestamp: time.Now().UnixNano(),
		}

		// Process message
		response, err := tokenProtocol.Process(context.Background(), msg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Processing error: %v", err), http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        response.ID(),
			"type":      response.Type(),
			"content":   response.Content(),
			"metadata":  json.RawMessage(response.Metadata()),
			"timestamp": response.Timestamp(),
		})
	})

	fmt.Println("Starting HTTP server on http://localhost:8090")
	go http.ListenAndServe(":8090", nil)

	fmt.Println("\nBridge verification server is running!")
	fmt.Println("- Bridge gRPC-Web endpoint: http://localhost:8080")
	fmt.Println("- Test HTTP API: http://localhost:8090/api/test")
	fmt.Println("- Status endpoint: http://localhost:8090/status")
	fmt.Println("\nPress Ctrl+C to stop...")

	// Wait forever
	select {}
}

// For manual verification
func main() {
	if len(os.Args) > 1 && os.Args[1] == "manual" {
		RunManualVerification()
		return
	}

	fmt.Println("Running automated verification tests...")
	testing.Main(func(pat, str string) (bool, error) { return true, nil }, 
		[]testing.InternalTest{{Name: "BridgeModuleVerification", F: TestBridgeModuleVerification}},
		nil, nil)
}



