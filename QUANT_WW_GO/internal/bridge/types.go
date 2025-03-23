// types.go - Bridge framework type definitions

package bridge

import (
    "context"
    "time"
)

// Config defines configuration for bridge operations
type Config struct {
    EventBufferSize int           // Buffer size for event channels
    Timeout         time.Duration // Default timeout for bridge operations
    RetryConfig     RetryConfig   // Configuration for retry logic
    TokenLimits     TokenLimits   // Token usage limitations
}

// RetryConfig defines retry behavior for bridge operations
type RetryConfig struct {
    MaxRetries      int           // Maximum number of retries
    BackoffInitial  time.Duration // Initial backoff duration
    BackoffFactor   float64       // Factor to increase backoff with each retry
    BackoffMax      time.Duration // Maximum backoff duration
}

// TokenLimits defines token usage limits for bridge operations
type TokenLimits struct {
    MaxInputTokens       int  // Maximum input tokens per request
    MaxOutputTokens      int  // Maximum output tokens per request
    MaxTotalTokens       int  // Maximum total tokens per request
    EnableChunking       bool // Whether to enable chunking for large payloads
    ChunkSize            int  // Token size for each chunk
    PreserveContextSize  int  // Tokens to preserve for context in chunked requests
}

// Target identifies a specific target for bridge communication
type Target struct {
    Adapter     string // Language adapter to use
    Protocol    string // Communication protocol to use
    Endpoint    string // Target endpoint
    ServiceName string // Name of the target service
}

// Request represents a bridge function call request
type Request struct {
    Target    Target      // Call target
    Function  string      // Function to call
    Params    interface{} // Function parameters
    Timestamp time.Time   // Request timestamp
    TokenInfo *TokenInfo  // Token usage information
}

// Response represents a bridge function call response
type Response struct {
    Result    interface{} // Function result
    Error     error       // Error if any
    Timestamp time.Time   // Response timestamp
    TokenInfo *TokenInfo  // Token usage information
}

// Event represents an event from a subscribed target
type Event struct {
    Type      string      // Event type
    Data      interface{} // Event data
    Source    Target      // Event source
    Timestamp time.Time   // Event timestamp
    TokenInfo *TokenInfo  // Token usage information
}

// SubscriptionRequest represents a request to subscribe to events
type SubscriptionRequest struct {
    Target    Target      // Subscription target
    EventType string      // Event type to subscribe to
    Filter    interface{} // Optional filter criteria
    Timestamp time.Time   // Request timestamp
}

// TokenInfo provides token usage information
type TokenInfo struct {
    InputTokens     int    // Number of input tokens
    OutputTokens    int    // Number of output tokens
    TotalTokens     int    // Total token count
    ModelContext    string // Model context identifier
    IsChunked       bool   // Whether content is chunked
    ChunkIndex      int    // Current chunk index
    TotalChunks     int    // Total number of chunks
    CompressionUsed bool   // Whether compression was used
}

// Adapter defines interface for language adapters
type Adapter interface {
    // Initialize sets up the adapter
    Initialize(ctx context.Context) error
    
    // Send sends a request and returns the response
    Send(ctx context.Context, request []byte) ([]byte, error)
    
    // Subscribe sets up a subscription to events
    Subscribe(ctx context.Context, request []byte, events chan<- []byte) (string, error)
    
    // Unsubscribe cancels a subscription
    Unsubscribe(subscriptionID string) error
    
    // Close releases resources
    Close() error
}

// Protocol defines interface for communication protocols
type Protocol interface {
    // Initialize sets up the protocol
    Initialize(ctx context.Context) error
    
    // Encode encodes a request
    Encode(request *Request) ([]byte, error)
    
    // Decode decodes a response
    Decode(response []byte) (*Response, error)
    
    // EncodeSubscription encodes a subscription request
    EncodeSubscription(request *SubscriptionRequest) ([]byte, error)
    
    // DecodeEvent decodes an event
    DecodeEvent(event []byte) (Event, error)
    
    // Close releases resources
    Close() error
}

// Subscription defines interface for event subscriptions
type Subscription interface {
    // Unsubscribe cancels the subscription
    Unsubscribe() error
    
    // GetID returns the subscription ID
    GetID() string
}
