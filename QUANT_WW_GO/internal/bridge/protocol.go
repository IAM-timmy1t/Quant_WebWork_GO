// protocol.go - Protocol definitions for bridge communication

package protocol

import (
	"context"
	"time"
)

// ProtocolType represents the type of protocol
type ProtocolType string

// Protocol types
const (
	TypeREST     ProtocolType = "rest"
	TypeGraphQL  ProtocolType = "graphql"
	TypeGRPC     ProtocolType = "grpc"
	TypeWebhook  ProtocolType = "webhook"
	TypeAMQP     ProtocolType = "amqp"
	TypeMQTT     ProtocolType = "mqtt"
	TypeWebSocket ProtocolType = "websocket"
	TypeSSE      ProtocolType = "sse"
	TypeCustom   ProtocolType = "custom"
)

// ProtocolConfig contains configuration for a bridge protocol
type ProtocolConfig struct {
	Type        ProtocolType           // Type of the protocol
	Name        string                 // Unique name for the protocol
	Description string                 // Description of the protocol
	Version     string                 // Protocol version
	Options     map[string]interface{} // Additional options for the protocol
	Timeout     time.Duration          // Default timeout for protocol operations
	RetryConfig RetryConfig            // Configuration for retry behavior
}

// RetryConfig defines retry behavior for protocols
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retries
	InitialInterval time.Duration // Initial interval between retries
	MaxInterval     time.Duration // Maximum interval between retries
	Multiplier      float64       // Factor by which to increase interval between retries
	RandomizeFactor float64       // Factor by which to randomize interval
}

// ProtocolStats contains operational statistics for a protocol
type ProtocolStats struct {
	MessagesProcessed int64         // Number of messages processed
	ErrorCount        int64         // Number of errors encountered
	LastActivity      time.Time     // Last activity timestamp
	AverageLatency    time.Duration // Average latency for operations
}

// ProtocolMetadata contains metadata about a protocol
type ProtocolMetadata struct {
	Type               ProtocolType          // Protocol type
	Name               string                // Protocol name
	Version            string                // Protocol version
	Specifications     []string              // Links to specification documents
	MediaTypes         []string              // Supported media types
	SecurityFeatures   []string              // Security features supported
	CompressionSupport []string              // Compression methods supported
	EncodingSupport    []string              // Encoding methods supported
	SchemaDefinition   map[string]interface{} // Protocol schema definition
}

// Message represents a protocol message
type Message struct {
	ID          string                 // Unique message ID
	Type        string                 // Message type
	Source      string                 // Source of the message
	Destination string                 // Intended destination
	Timestamp   time.Time              // When the message was created
	Payload     interface{}            // Message payload
	Headers     map[string]string      // Message headers
	Metadata    map[string]interface{} // Additional metadata
	TTL         time.Duration          // Time to live
	Priority    int                    // Message priority
}

// MessageTransformer transforms messages between formats
type MessageTransformer interface {
	// Transform transforms a message from one format to another
	Transform(ctx context.Context, message *Message) (*Message, error)
	
	// SupportsTransformation checks if the transformer supports the given transformation
	SupportsTransformation(sourceType, targetType string) bool
}

// Protocol defines the interface for communication protocols
type Protocol interface {
	// Lifecycle methods
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
	
	// Status methods
	Stats() ProtocolStats
	
	// Configuration and information
	Type() ProtocolType
	Name() string
	Version() string
	Metadata() ProtocolMetadata
	Config() ProtocolConfig
	
	// Message handling
	EncodeMessage(ctx context.Context, message *Message) ([]byte, error)
	DecodeMessage(ctx context.Context, data []byte) (*Message, error)
	ValidateMessage(ctx context.Context, message *Message) error
	
	// Communication primitives
	Send(ctx context.Context, message *Message) error
	Receive(ctx context.Context) (*Message, error)
	
	// Error handling
	LastError() error
}

// BaseProtocol provides a basic implementation of the Protocol interface
type BaseProtocol struct {
	protocolType   ProtocolType
	name           string
	version        string
	config         ProtocolConfig
	metadata       ProtocolMetadata
	stats          ProtocolStats
	lastError      error
	transformers   []MessageTransformer
}

// NewBaseProtocol creates a new base protocol
func NewBaseProtocol(name string, protocolType ProtocolType, config ProtocolConfig, metadata ProtocolMetadata) *BaseProtocol {
	return &BaseProtocol{
		name:         name,
		protocolType: protocolType,
		config:       config,
		metadata:     metadata,
	}
}

// Type returns the protocol type
func (p *BaseProtocol) Type() ProtocolType {
	return p.protocolType
}

// Name returns the protocol name
func (p *BaseProtocol) Name() string {
	return p.name
}

// Version returns the protocol version
func (p *BaseProtocol) Version() string {
	return p.version
}

// Metadata returns the protocol metadata
func (p *BaseProtocol) Metadata() ProtocolMetadata {
	return p.metadata
}

// Config returns the protocol configuration
func (p *BaseProtocol) Config() ProtocolConfig {
	return p.config
}

// Stats returns the protocol statistics
func (p *BaseProtocol) Stats() ProtocolStats {
	return p.stats
}

// LastError returns the last error encountered by the protocol
func (p *BaseProtocol) LastError() error {
	return p.lastError
}

// AddTransformer adds a message transformer to the protocol
func (p *BaseProtocol) AddTransformer(transformer MessageTransformer) {
	p.transformers = append(p.transformers, transformer)
}

// FindTransformer finds a transformer that supports the given transformation
func (p *BaseProtocol) FindTransformer(sourceType, targetType string) MessageTransformer {
	for _, transformer := range p.transformers {
		if transformer.SupportsTransformation(sourceType, targetType) {
			return transformer
		}
	}
	return nil
}

// incrementMessagesProcessed increments the messages processed counter
func (p *BaseProtocol) incrementMessagesProcessed() {
	p.stats.MessagesProcessed++
	p.stats.LastActivity = time.Now()
}

// incrementErrorCount increments the error counter
func (p *BaseProtocol) incrementErrorCount() {
	p.stats.ErrorCount++
}

// recordLatency records the latency for an operation
func (p *BaseProtocol) recordLatency(duration time.Duration) {
	// Simple moving average for latency
	if p.stats.AverageLatency == 0 {
		p.stats.AverageLatency = duration
	} else {
		p.stats.AverageLatency = (p.stats.AverageLatency*9 + duration) / 10
	}
}

// setError sets the last error
func (p *BaseProtocol) setError(err error) {
	p.lastError = err
	p.incrementErrorCount()
}

// Initialize provides a default implementation of Initialize
func (p *BaseProtocol) Initialize(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// Shutdown provides a default implementation of Shutdown
func (p *BaseProtocol) Shutdown(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// EncodeMessage provides a default implementation of EncodeMessage
func (p *BaseProtocol) EncodeMessage(ctx context.Context, message *Message) ([]byte, error) {
	// Default implementation does nothing
	p.incrementMessagesProcessed()
	return nil, nil
}

// DecodeMessage provides a default implementation of DecodeMessage
func (p *BaseProtocol) DecodeMessage(ctx context.Context, data []byte) (*Message, error) {
	// Default implementation does nothing
	p.incrementMessagesProcessed()
	return nil, nil
}

// ValidateMessage provides a default implementation of ValidateMessage
func (p *BaseProtocol) ValidateMessage(ctx context.Context, message *Message) error {
	// Default implementation does nothing
	return nil
}

// Send provides a default implementation of Send
func (p *BaseProtocol) Send(ctx context.Context, message *Message) error {
	// Default implementation does nothing
	p.incrementMessagesProcessed()
	return nil
}

// Receive provides a default implementation of Receive
func (p *BaseProtocol) Receive(ctx context.Context) (*Message, error) {
	// Default implementation does nothing
	p.incrementMessagesProcessed()
	return nil, nil
}
