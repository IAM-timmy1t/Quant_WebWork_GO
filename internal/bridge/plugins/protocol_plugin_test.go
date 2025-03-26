package plugins

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolPlugin(t *testing.T) {
	// Create a new protocol plugin
	plugin := NewProtocolPlugin("test-plugin", ProtocolJSON, "1.0", "application/json",
		WithCapabilities(CapabilityEncode, CapabilityDecode, CapabilityValidate))

	// Test plugin ID and type
	assert.Equal(t, "test-plugin", plugin.ID())
	assert.Equal(t, PluginTypeProtocol, plugin.Type())
	assert.Equal(t, "json-protocol", plugin.metadata.Name)
	assert.Equal(t, "1.0", plugin.version)
	assert.Equal(t, "application/json", plugin.contentType)

	// Test capabilities
	assert.Contains(t, plugin.Capabilities(), CapabilityEncode)
	assert.Contains(t, plugin.Capabilities(), CapabilityDecode)
	assert.Contains(t, plugin.Capabilities(), CapabilityValidate)
	assert.True(t, plugin.SupportsCapability(CapabilityEncode))
	assert.False(t, plugin.SupportsCapability(CapabilityCompression))

	// Test initialize
	ctx := context.Background()
	config := map[string]interface{}{
		"encoder_options": map[string]interface{}{
			"pretty": true,
		},
	}
	err := plugin.Initialize(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, PluginStatusInitialized, plugin.Status())
	assert.Equal(t, true, plugin.encoderOptions["pretty"])

	// Test starting the plugin
	err = plugin.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, PluginStatusStarted, plugin.Status())

	// Test custom message handler
	testPayload := map[string]interface{}{
		"greeting": "Hello, world!",
	}
	handlerCalled := false
	plugin.RegisterMessageHandler("test", func(ctx context.Context, message *ProtocolMessage) (*ProtocolMessage, error) {
		handlerCalled = true
		assert.Equal(t, "test-message", message.ID)
		assert.Equal(t, "test", message.Type)
		assert.Equal(t, testPayload, message.Payload)

		// Create response
		response := &ProtocolMessage{
			ID:      "test-response",
			Type:    "test-response",
			Payload: map[string]interface{}{"status": "success"},
		}
		return response, nil
	})

	// Create test message
	message := &ProtocolMessage{
		ID:        "test-message",
		Type:      "test",
		Payload:   testPayload,
		Timestamp: time.Now(),
	}

	// Test encoding
	encoded, err := plugin.Encode(ctx, message)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	// Test decoding
	decoded, err := plugin.Decode(ctx, encoded)
	require.NoError(t, err)
	assert.Equal(t, message.ID, decoded.ID)
	assert.Equal(t, message.Type, decoded.Type)
	assert.Equal(t, message.Payload, decoded.Payload)

	// Test message processing
	processed, err := plugin.ProcessMessage(ctx, encoded)
	require.NoError(t, err)
	assert.True(t, handlerCalled)

	// Decode and verify response
	var response ProtocolMessage
	err = json.Unmarshal(processed, &response)
	require.NoError(t, err)
	assert.Equal(t, "test-response", response.ID)
	assert.Equal(t, "test-response", response.Type)
	assert.Equal(t, map[string]interface{}{"status": "success"}, response.Payload)

	// Test stats
	stats := plugin.GetStats()
	assert.Equal(t, int64(2), stats.MessagesEncoded) // 1 for the test message, 1 for the response
	assert.Equal(t, int64(1), stats.MessagesDecoded) // 1 for the test message
	assert.True(t, stats.ProcessingTimeNs > 0)
	assert.NotZero(t, stats.LastMessageTime)

	// Test stopping the plugin
	err = plugin.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, PluginStatusStopped, plugin.Status())

	// Test cleanup
	err = plugin.Cleanup(ctx)
	require.NoError(t, err)
}

func TestJSONValidator(t *testing.T) {
	// Create a JSON protocol plugin
	plugin := CreateJSONProtocolPlugin("json-test")
	ctx := context.Background()

	// Initialize plugin
	err := plugin.Initialize(ctx, nil)
	require.NoError(t, err)

	// Test with valid message
	validMessage := &ProtocolMessage{
		ID:      "test-id",
		Type:    "test-type",
		Payload: map[string]interface{}{"data": "test"},
	}

	encoded, err := plugin.Encode(ctx, validMessage)
	require.NoError(t, err)

	// Test with invalid message (missing ID)
	invalidMessage := &ProtocolMessage{
		Type:    "test-type",
		Payload: map[string]interface{}{"data": "test"},
	}

	_, err = plugin.Encode(ctx, invalidMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message ID cannot be empty")

	// Test with invalid message (missing type)
	invalidMessage = &ProtocolMessage{
		ID:      "test-id",
		Payload: map[string]interface{}{"data": "test"},
	}

	_, err = plugin.Encode(ctx, invalidMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message type cannot be empty")

	// Test echo handler
	echoMessage := &ProtocolMessage{
		ID:      "echo-id",
		Type:    "echo",
		Payload: map[string]interface{}{"echo": "test"},
	}

	encoded, err = plugin.Encode(ctx, echoMessage)
	require.NoError(t, err)

	processed, err := plugin.ProcessMessage(ctx, encoded)
	require.NoError(t, err)

	var response ProtocolMessage
	err = json.Unmarshal(processed, &response)
	require.NoError(t, err)
	assert.Equal(t, "echo-response", response.Type)
	assert.Equal(t, map[string]interface{}{"echo": "test"}, response.Payload)

	// Test info handler
	infoMessage := &ProtocolMessage{
		ID:      "info-id",
		Type:    "info",
		Payload: map[string]interface{}{},
	}

	encoded, err = plugin.Encode(ctx, infoMessage)
	require.NoError(t, err)

	processed, err = plugin.ProcessMessage(ctx, encoded)
	require.NoError(t, err)

	err = json.Unmarshal(processed, &response)
	require.NoError(t, err)
	assert.Equal(t, "info-response", response.Type)

	// Check info payload
	payload, ok := response.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, ProtocolJSON, payload["protocol"])
	assert.Equal(t, "1.0", payload["version"])
	assert.Equal(t, "application/json", payload["contentType"])
}

func TestMessageValidation(t *testing.T) {
	// Create a custom protocol plugin with validator
	plugin := NewProtocolPlugin("validation-test", ProtocolJSON, "1.0", "application/json")

	// Add custom validator
	plugin.SetValidator(func(ctx context.Context, message *ProtocolMessage) (*MessageValidationResult, error) {
		errors := []string{}
		warnings := []string{}

		// Check ID
		if message.ID == "" {
			errors = append(errors, "ID is required")
		}

		// Check Type
		if message.Type == "" {
			errors = append(errors, "Type is required")
		}

		// Check payload
		if message.Payload == nil {
			errors = append(errors, "Payload is required")
		}

		// Add warning for missing timestamp
		if message.Timestamp.IsZero() {
			warnings = append(warnings, "Timestamp is missing")
		}

		return &MessageValidationResult{
			Valid:    len(errors) == 0,
			Errors:   errors,
			Warnings: warnings,
		}, nil
	})

	ctx := context.Background()
	err := plugin.Initialize(ctx, nil)
	require.NoError(t, err)

	// Test with valid message
	validMessage := &ProtocolMessage{
		ID:      "test-id",
		Type:    "test-type",
		Payload: "Hello world",
	}

	_, err = plugin.Encode(ctx, validMessage)
	assert.NoError(t, err)

	// Test with invalid message (missing everything)
	invalidMessage := &ProtocolMessage{}

	_, err = plugin.Encode(ctx, invalidMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID is required")
	assert.Contains(t, err.Error(), "Type is required")
	assert.Contains(t, err.Error(), "Payload is required")
}
