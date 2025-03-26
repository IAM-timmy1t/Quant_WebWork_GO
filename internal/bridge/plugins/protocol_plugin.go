// protocol_plugin.go - Protocol plugin implementation for bridge

package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Common protocols
const (
	ProtocolJSON        = "json"
	ProtocolProtobuf    = "protobuf"
	ProtocolMessagePack = "msgpack"
	ProtocolXML         = "xml"
	ProtocolBinary      = "binary"
)

// Protocol capabilities
const (
	CapabilityEncode        = "encode"
	CapabilityDecode        = "decode"
	CapabilityValidate      = "validate"
	CapabilityCompression   = "compression"
	CapabilityEncryption    = "encryption"
	CapabilityStreaming     = "streaming"
	CapabilityBidirectional = "bidirectional"
)

// MessageValidationResult represents the result of message validation
type MessageValidationResult struct {
	Valid    bool                   `json:"valid"`
	Errors   []string               `json:"errors,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// ProtocolMessage represents a message handled by the protocol
type ProtocolMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Headers   map[string]string      `json:"headers,omitempty"`
	Payload   interface{}            `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ProtocolPlugin implements a protocol plugin
type ProtocolPlugin struct {
	BasePlugin
	protocolName    string
	version         string
	contentType     string
	messageHandlers map[string]MessageHandler
	validator       MessageValidator
	encoderOptions  map[string]interface{}
	decoderOptions  map[string]interface{}
	handlersMutex   sync.RWMutex
	stats           ProtocolStats
}

// MessageHandler is a function that handles a message
type MessageHandler func(ctx context.Context, message *ProtocolMessage) (*ProtocolMessage, error)

// MessageValidator validates messages
type MessageValidator func(ctx context.Context, message *ProtocolMessage) (*MessageValidationResult, error)

// ProtocolStats tracks protocol statistics
type ProtocolStats struct {
	MessagesEncoded  int64
	MessagesDecoded  int64
	ValidationErrors int64
	EncodingErrors   int64
	DecodingErrors   int64
	TotalMessageSize int64
	LastMessageTime  time.Time
	ProcessingTimeNs int64
	ProcessingCount  int64
}

// NewProtocolPlugin creates a new protocol plugin
func NewProtocolPlugin(id string, protocolName, version, contentType string, options ...PluginOption) *ProtocolPlugin {
	metadata := PluginMetadata{
		Name:        fmt.Sprintf("%s-protocol", protocolName),
		Version:     version,
		Description: fmt.Sprintf("%s protocol implementation", protocolName),
		Tags:        []string{"protocol", protocolName},
	}

	basePlugin := NewPlugin(id, PluginTypeProtocol, append([]PluginOption{WithMetadata(metadata)}, options...)...)

	return &ProtocolPlugin{
		BasePlugin:      *basePlugin,
		protocolName:    protocolName,
		version:         version,
		contentType:     contentType,
		messageHandlers: make(map[string]MessageHandler),
		encoderOptions:  make(map[string]interface{}),
		decoderOptions:  make(map[string]interface{}),
	}
}

// Initialize initializes the protocol plugin
func (p *ProtocolPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Call base initialize
	if err := p.BasePlugin.Initialize(ctx, config); err != nil {
		return err
	}

	// Configure encoder options
	if encoderOpts, ok := config["encoder_options"].(map[string]interface{}); ok {
		for k, v := range encoderOpts {
			p.encoderOptions[k] = v
		}
	}

	// Configure decoder options
	if decoderOpts, ok := config["decoder_options"].(map[string]interface{}); ok {
		for k, v := range decoderOpts {
			p.decoderOptions[k] = v
		}
	}

	// Register built-in handlers
	p.registerBuiltInHandlers()

	return nil
}

// registerBuiltInHandlers registers built-in message handlers
func (p *ProtocolPlugin) registerBuiltInHandlers() {
	// Echo handler (for testing)
	p.RegisterMessageHandler("echo", func(ctx context.Context, message *ProtocolMessage) (*ProtocolMessage, error) {
		// Create a new message with the same payload
		response := &ProtocolMessage{
			ID:        fmt.Sprintf("echo-%s", message.ID),
			Type:      "echo-response",
			Headers:   message.Headers,
			Payload:   message.Payload,
			Metadata:  message.Metadata,
			Timestamp: time.Now(),
		}
		return response, nil
	})

	// Info handler (returns protocol info)
	p.RegisterMessageHandler("info", func(ctx context.Context, message *ProtocolMessage) (*ProtocolMessage, error) {
		// Create a response with protocol info
		response := &ProtocolMessage{
			ID:        fmt.Sprintf("info-%s", message.ID),
			Type:      "info-response",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"protocol":     p.protocolName,
				"version":      p.version,
				"contentType":  p.contentType,
				"capabilities": p.Capabilities(),
				"stats":        p.stats,
				"handlers":     getHandlerNames(p.messageHandlers),
			},
		}
		return response, nil
	})

	// Stats handler (returns protocol stats)
	p.RegisterMessageHandler("stats", func(ctx context.Context, message *ProtocolMessage) (*ProtocolMessage, error) {
		// Create a response with protocol stats
		response := &ProtocolMessage{
			ID:        fmt.Sprintf("stats-%s", message.ID),
			Type:      "stats-response",
			Timestamp: time.Now(),
			Payload:   p.stats,
		}
		return response, nil
	})
}

// getHandlerNames returns the names of all registered handlers
func getHandlerNames(handlers map[string]MessageHandler) []string {
	names := make([]string, 0, len(handlers))
	for name := range handlers {
		names = append(names, name)
	}
	return names
}

// RegisterMessageHandler registers a message handler
func (p *ProtocolPlugin) RegisterMessageHandler(messageType string, handler MessageHandler) {
	p.handlersMutex.Lock()
	defer p.handlersMutex.Unlock()
	p.messageHandlers[messageType] = handler
}

// UnregisterMessageHandler unregisters a message handler
func (p *ProtocolPlugin) UnregisterMessageHandler(messageType string) {
	p.handlersMutex.Lock()
	defer p.handlersMutex.Unlock()
	delete(p.messageHandlers, messageType)
}

// SetValidator sets the message validator
func (p *ProtocolPlugin) SetValidator(validator MessageValidator) {
	p.validator = validator
}

// Encode encodes a message
func (p *ProtocolPlugin) Encode(ctx context.Context, message *ProtocolMessage) ([]byte, error) {
	startTime := time.Now()

	// Validate message if validator is set
	if p.validator != nil {
		result, err := p.validator(ctx, message)
		if err != nil {
			p.stats.ValidationErrors++
			return nil, fmt.Errorf("message validation error: %w", err)
		}

		if !result.Valid {
			p.stats.ValidationErrors++
			return nil, fmt.Errorf("invalid message: %v", result.Errors)
		}
	}

	// Set timestamp if not set
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Encode the message
	var data []byte
	var err error
	switch p.protocolName {
	case ProtocolJSON:
		data, err = json.Marshal(message)
	case ProtocolMessagePack:
		// MessagePack encoding would go here
		err = fmt.Errorf("MessagePack encoding not implemented")
	case ProtocolProtobuf:
		// Protobuf encoding would go here
		err = fmt.Errorf("Protobuf encoding not implemented")
	case ProtocolXML:
		// XML encoding would go here
		err = fmt.Errorf("XML encoding not implemented")
	default:
		// Default to JSON
		data, err = json.Marshal(message)
	}

	if err != nil {
		p.stats.EncodingErrors++
		return nil, fmt.Errorf("message encoding error: %w", err)
	}

	// Update statistics
	p.stats.MessagesEncoded++
	p.stats.TotalMessageSize += int64(len(data))
	p.stats.LastMessageTime = time.Now()

	processingTime := time.Since(startTime).Nanoseconds()
	p.stats.ProcessingTimeNs += processingTime
	p.stats.ProcessingCount++

	return data, nil
}

// Decode decodes a message
func (p *ProtocolPlugin) Decode(ctx context.Context, data []byte) (*ProtocolMessage, error) {
	startTime := time.Now()

	// Decode the message
	var message ProtocolMessage
	var err error
	switch p.protocolName {
	case ProtocolJSON:
		err = json.Unmarshal(data, &message)
	case ProtocolMessagePack:
		// MessagePack decoding would go here
		err = fmt.Errorf("MessagePack decoding not implemented")
	case ProtocolProtobuf:
		// Protobuf decoding would go here
		err = fmt.Errorf("Protobuf decoding not implemented")
	case ProtocolXML:
		// XML decoding would go here
		err = fmt.Errorf("XML decoding not implemented")
	default:
		// Default to JSON
		err = json.Unmarshal(data, &message)
	}

	if err != nil {
		p.stats.DecodingErrors++
		return nil, fmt.Errorf("message decoding error: %w", err)
	}

	// Validate message if validator is set
	if p.validator != nil {
		result, err := p.validator(ctx, &message)
		if err != nil {
			p.stats.ValidationErrors++
			return nil, fmt.Errorf("message validation error: %w", err)
		}

		if !result.Valid {
			p.stats.ValidationErrors++
			return nil, fmt.Errorf("invalid message: %v", result.Errors)
		}
	}

	// Update statistics
	p.stats.MessagesDecoded++
	p.stats.LastMessageTime = time.Now()

	processingTime := time.Since(startTime).Nanoseconds()
	p.stats.ProcessingTimeNs += processingTime
	p.stats.ProcessingCount++

	return &message, nil
}

// ProcessMessage processes a message using a registered handler
func (p *ProtocolPlugin) ProcessMessage(ctx context.Context, data []byte) ([]byte, error) {
	// Decode the message
	message, err := p.Decode(ctx, data)
	if err != nil {
		return nil, err
	}

	// Find handler for message type
	p.handlersMutex.RLock()
	handler, exists := p.messageHandlers[message.Type]
	p.handlersMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler registered for message type '%s'", message.Type)
	}

	// Handle the message
	response, err := handler(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("error handling message: %w", err)
	}

	// Encode the response
	return p.Encode(ctx, response)
}

// GetStats returns protocol statistics
func (p *ProtocolPlugin) GetStats() ProtocolStats {
	return p.stats
}

// GetAverageProcessingTime returns the average processing time in nanoseconds
func (p *ProtocolPlugin) GetAverageProcessingTime() int64 {
	if p.stats.ProcessingCount == 0 {
		return 0
	}
	return p.stats.ProcessingTimeNs / p.stats.ProcessingCount
}

// CreateJSONProtocolPlugin creates a new JSON protocol plugin
func CreateJSONProtocolPlugin(id string) *ProtocolPlugin {
	plugin := NewProtocolPlugin(id, ProtocolJSON, "1.0", "application/json",
		WithCapabilities(CapabilityEncode, CapabilityDecode, CapabilityValidate))

	// Add JSON-specific validator
	plugin.SetValidator(func(ctx context.Context, message *ProtocolMessage) (*MessageValidationResult, error) {
		// Perform basic validation
		if message.ID == "" {
			return &MessageValidationResult{
				Valid:  false,
				Errors: []string{"message ID cannot be empty"},
			}, nil
		}

		if message.Type == "" {
			return &MessageValidationResult{
				Valid:  false,
				Errors: []string{"message type cannot be empty"},
			}, nil
		}

		return &MessageValidationResult{
			Valid: true,
		}, nil
	})

	return plugin
}

// PluginFactory implementations
func init() {
	// Register JSON protocol plugin factory
	jsonProtocolFactory := func(id string, config map[string]interface{}) (Plugin, error) {
		plugin := CreateJSONProtocolPlugin(id)
		return plugin, nil
	}

	// Add to registry when available
	if registry, ok := globalRegistry.(*Registry); ok {
		registry.RegisterFactory("json-protocol", jsonProtocolFactory)
	}
}

// Global registry (if available)
var globalRegistry interface{}
