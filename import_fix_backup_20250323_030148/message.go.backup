// message.go - Standardized messaging

package websocket

import (
    "time"
)

// MessageType defines the type of WebSocket message
type MessageType string

// Standard message types
const (
    MessageTypeData     MessageType = "data"
    MessageTypeControl  MessageType = "control"
    MessageTypeError    MessageType = "error"
    MessageTypeMetrics  MessageType = "metrics"
    MessageTypeSecurity MessageType = "security"
    MessageTypeSystem   MessageType = "system"
)

// Message represents a WebSocket message with standardized format
type Message struct {
    Channel   string          `json:"channel,omitempty"` // Target channel
    Type      MessageType     `json:"type"`              // Message type
    Data      interface{}     `json:"data,omitempty"`    // Message payload
    RequestID string          `json:"request_id,omitempty"` // For request/response pattern
    Time      time.Time       `json:"time"`              // Message timestamp
    TokenInfo *MessageTokenInfo `json:"token_info,omitempty"` // Token usage information
}

// MessageTokenInfo provides token usage information for large payloads
type MessageTokenInfo struct {
    EstimatedTokens int    `json:"estimated_tokens"` // Estimated token count
    ModelContext    string `json:"model_context"`    // Model context identifier
    Truncated       bool   `json:"truncated"`        // Whether content was truncated
    ChunkIndex      int    `json:"chunk_index"`      // For multi-part messages
    TotalChunks     int    `json:"total_chunks"`     // Total chunks in sequence
}

// ChannelConfig defines configuration for a WebSocket channel
type ChannelConfig struct {
    Name               string            // Channel name
    Description        string            // Channel description
    RequiresAuth       bool              // Whether authentication is required
    AllowedMessageTypes []MessageType    // Allowed message types
    MaxTokensPerMessage int              // Maximum tokens allowed per message
    TokenCalculator    TokenCalculator   // Token estimation function
}

// Config defines WebSocket hub configuration
type Config struct {
    WriteWait         time.Duration       // Time allowed to write a message
    PongWait          time.Duration       // Time allowed to read the next pong message
    PingPeriod        time.Duration       // Send pings with this period
    MaxMessageSize    int64               // Maximum message size allowed
    MessageBufferSize int                 // Buffer size for message broadcasting
    TokenSettings     TokenSettings       // Token management settings
}

// TokenSettings defines token management configuration
type TokenSettings struct {
    EnableTokenManagement bool   // Whether to enable token management
    DefaultModelContext   string // Default model for token calculation
    MaxTokensPerMessage   int    // Maximum tokens per message
    ChunkSize             int    // Size of chunks for large messages
}

// HubStats contains statistics about WebSocket hub operations
type HubStats struct {
    CurrentConnections  int       // Current number of active connections
    TotalConnections    int       // Total number of connections since start
    DroppedConnections  int       // Number of dropped connections
    MessagesSent        int       // Total messages sent
    Timestamp           time.Time // When stats were last updated
}

// TokenCalculator calculates token count for different content types
type TokenCalculator func(content interface{}, modelContext string) int

// DefaultTokenCalculator provides a simple token estimation
func DefaultTokenCalculator(content interface{}, modelContext string) int {
    // This is a simplified implementation
    // A real implementation would use model-specific token counting logic
    switch v := content.(type) {
    case string:
        // Rough approximation: 1 token per 4 characters for text
        return len(v) / 4
    case map[string]interface{}:
        // Rough estimate for JSON objects
        size := 0
        for key, value := range v {
            size += len(key)
            if strVal, ok := value.(string); ok {
                size += len(strVal)
            }
        }
        return size / 4
    default:
        // Default conservative estimate
        return 100
    }
}

// NewMessage creates a new message with the current timestamp
func NewMessage(msgType MessageType, data interface{}) Message {
    return Message{
        Type: msgType,
        Data: data,
        Time: time.Now(),
    }
}

// WithChannel adds a channel to the message
func (m Message) WithChannel(channel string) Message {
    m.Channel = channel
    return m
}

// WithRequestID adds a request ID to the message
func (m Message) WithRequestID(requestID string) Message {
    m.RequestID = requestID
    return m
}

// WithTokenInfo adds token usage information to the message
func (m Message) WithTokenInfo(tokenInfo *MessageTokenInfo) Message {
    m.TokenInfo = tokenInfo
    return m
}

// DefaultConfig returns the default WebSocket configuration
func DefaultConfig() Config {
    return Config{
        WriteWait:         10 * time.Second,
        PongWait:          60 * time.Second,
        PingPeriod:        (60 * time.Second * 9) / 10,
        MaxMessageSize:    512 * 1024, // 512KB
        MessageBufferSize: 256,
        TokenSettings: TokenSettings{
            EnableTokenManagement: true,
            DefaultModelContext:   "default",
            MaxTokensPerMessage:   8000,
            ChunkSize:             4000,
        },
    }
}
