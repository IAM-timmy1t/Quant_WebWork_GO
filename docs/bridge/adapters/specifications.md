# Bridge Adapter Specifications

## Overview

Bridge Adapters are a crucial component of the QUANT_WebWork_GO system, providing the communication layer between different parts of the application. Each adapter is responsible for handling a specific communication protocol, translating between the protocol-specific details and the bridge's internal message format.

This document specifies the requirements, interfaces, and implementation guidelines for all bridge adapters.

## Adapter Interface

All bridge adapters must implement the following interface:

```go
// Adapter defines the interface for bridge adapters
type Adapter interface {
    // Lifecycle methods
    Initialize(ctx context.Context) error
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Shutdown(ctx context.Context) error
    
    // Status methods
    Status() AdapterStatus
    Stats() AdapterStats
    
    // Configuration and information
    Name() string
    Type() string
    Metadata() AdapterMetadata
    Config() AdapterConfig
    
    // Communication methods
    Send(ctx context.Context, data []byte) ([]byte, error)
    Receive(ctx context.Context) ([]byte, error)
    
    // Error handling
    LastError() error
}
```

### Lifecycle Methods

- **Initialize**: Initializes the adapter with its configuration. This should not establish connections but should prepare the adapter for use.
- **Connect**: Establishes a connection to the target service or system.
- **Disconnect**: Disconnects from the target service or system, but keeps the adapter initialized.
- **Shutdown**: Completely shuts down the adapter, releasing all resources.

### Status Methods

- **Status**: Returns the current status of the adapter (Uninitialized, Initialized, Connected, Disconnected, Error).
- **Stats**: Returns statistics about the adapter's operation, such as message counts and response times.

### Configuration and Information

- **Name**: Returns the unique name of the adapter instance.
- **Type**: Returns the adapter type (e.g., "grpc", "rest", "websocket").
- **Metadata**: Returns metadata about the adapter, such as version and capabilities.
- **Config**: Returns the adapter's configuration (with sensitive information redacted).

### Communication Methods

- **Send**: Sends data through the adapter and returns the response, if any.
- **Receive**: Receives data from the adapter (for adapters that support asynchronous reception).

### Error Handling

- **LastError**: Returns the last error encountered by the adapter.

## Common Adapter Status Values

```go
// AdapterStatus represents the status of an adapter
type AdapterStatus string

const (
    StatusUninitialized AdapterStatus = "uninitialized"
    StatusInitialized   AdapterStatus = "initialized"
    StatusConnecting    AdapterStatus = "connecting"
    StatusConnected     AdapterStatus = "connected"
    StatusDisconnecting AdapterStatus = "disconnecting"
    StatusDisconnected  AdapterStatus = "disconnected"
    StatusError         AdapterStatus = "error"
)
```

## Adapter Configuration

Each adapter type has its own configuration structure, but all must include the following fields:

```go
// AdapterConfig contains common configuration for all adapters
type AdapterConfig struct {
    Name       string                 // Unique name for this adapter instance
    Type       string                 // Adapter type (grpc, rest, websocket, etc.)
    Protocol   string                 // Communication protocol
    Host       string                 // Target host
    Port       int                    // Target port
    Path       string                 // Path (if applicable)
    Timeout    time.Duration          // Default timeout
    RetryCount int                    // Number of retries
    RetryDelay time.Duration          // Delay between retries
    Options    map[string]interface{} // Protocol-specific options
}
```

## Adapter Factory

Adapters are created through a factory pattern:

```go
// AdapterFactory creates a new adapter
type AdapterFactory func(config AdapterConfig) (Adapter, error)
```

Adapter factories must be registered with the bridge system:

```go
// RegisterAdapterFactory registers an adapter factory
func RegisterAdapterFactory(adapterType string, factory AdapterFactory)
```

## Common Adapter Implementations

### Base Adapter

All adapters should embed the `BaseAdapter` struct, which provides common functionality:

```go
// BaseAdapter provides a basic implementation of the Adapter interface
type BaseAdapter struct {
    name        string
    adapterType string
    config      AdapterConfig
    metadata    AdapterMetadata
    status      AdapterStatus
    stats       AdapterStats
    lastError   error
    mutex       sync.RWMutex
}
```

The `BaseAdapter` implements common methods like `Name()`, `Type()`, `Status()`, etc., allowing adapter implementations to focus on protocol-specific functionality.

## Adapter Metadata

Adapters provide metadata about their capabilities and implementation:

```go
// AdapterMetadata contains metadata about an adapter
type AdapterMetadata struct {
    Version       string            // Adapter version
    Capabilities  []string          // List of capabilities
    Author        string            // Adapter author
    Documentation string            // Link to documentation
    Properties    map[string]string // Additional properties
}
```

## Adapter Statistics

Adapters collect and report statistics about their operation:

```go
// AdapterStats contains statistics about an adapter
type AdapterStats struct {
    MessagesSent        int64
    MessagesReceived    int64
    BytesSent           int64
    BytesReceived       int64
    Errors              int64
    ConnectCount        int64
    DisconnectCount     int64
    LastConnectTime     time.Time
    LastDisconnectTime  time.Time
    AverageResponseTime time.Duration
    MaxResponseTime     time.Duration
    MinResponseTime     time.Duration
    Uptime              time.Duration
    CustomStats         map[string]interface{}
}
```

## Error Handling

Adapters must handle errors appropriately:

1. **Transient Errors**: Errors that may resolve themselves (e.g., network timeouts) should trigger retries according to the adapter's retry policy.
2. **Permanent Errors**: Errors that cannot be resolved through retries (e.g., authentication failures) should be reported immediately.
3. **Critical Errors**: Errors that affect the adapter's ability to function should transition the adapter to the `StatusError` state and may trigger a disconnect.

All errors should be logged with appropriate context and returned to the caller.

## Thread Safety

Adapters must be thread-safe. All methods that access or modify the adapter's state must use appropriate synchronization mechanisms.

## Connection Management

Adapters that maintain persistent connections should implement the following behaviors:

1. **Automatic Reconnection**: Adapters should automatically attempt to reconnect if the connection is lost, with configurable backoff.
2. **Connection Pooling**: When appropriate, adapters should use connection pooling to improve performance.
3. **Graceful Shutdown**: Adapters should release resources gracefully during shutdown.

## Logging

Adapters should log significant events using the provided logger:

```go
// Logger defines a logging interface for adapters
type Logger interface {
    Debug(msg string, fields map[string]interface{})
    Info(msg string, fields map[string]interface{})
    Warn(msg string, fields map[string]interface{})
    Error(msg string, fields map[string]interface{})
}
```

## Metrics

Adapters should collect and report metrics using the provided metrics collector:

```go
// Metrics collection
adapter.metrics.RecordRequest(method, url, statusCode, duration)
adapter.metrics.RecordError(errorType)
adapter.metrics.RecordBytes(direction, count)
```

## Implementation Guidelines

### gRPC Adapter

The gRPC adapter connects to gRPC services and translates between the bridge message format and gRPC requests/responses.

**Key Features:**
- Supports unary, server streaming, client streaming, and bidirectional streaming
- Automatic connection management
- Supports TLS with certificate validation
- Supports authentication via various mechanisms

### REST Adapter

The REST adapter connects to HTTP/HTTPS services and translates between the bridge message format and HTTP requests/responses.

**Key Features:**
- Supports all HTTP methods (GET, POST, PUT, DELETE, etc.)
- Handles request/response headers
- Supports query parameters and URL templates
- Configurable authentication (Basic, Bearer, API Key, etc.)
- Automatic handling of cookies and sessions

### WebSocket Adapter

The WebSocket adapter establishes WebSocket connections and maintains bidirectional communication.

**Key Features:**
- Handles connection establishment and maintenance
- Automatic reconnection with configurable backoff
- Ping/pong for connection health monitoring
- Support for binary and text messages
- Optional message compression

## Adapter Testing

Adapters should include comprehensive tests:

1. **Unit Tests**: Test adapter methods in isolation with mocked dependencies.
2. **Integration Tests**: Test adapter interaction with actual services (may require test doubles).
3. **Performance Tests**: Measure adapter performance under various loads.
4. **Error Handling Tests**: Verify adapter behavior in error scenarios.

## Example Adapter Implementation

```go
// Example REST adapter implementation
type RESTAdapter struct {
    BaseAdapter
    client    *http.Client
    baseURL   string
    headers   map[string]string
    mutex     sync.RWMutex
}

func NewRESTAdapter(config AdapterConfig) (*RESTAdapter, error) {
    if config.Host == "" {
        return nil, errors.New("host is required")
    }
    
    baseURL := fmt.Sprintf("%s://%s:%d%s", 
        config.Protocol, config.Host, config.Port, config.Path)
    
    adapter := &RESTAdapter{
        BaseAdapter: BaseAdapter{
            name:        config.Name,
            adapterType: "rest",
            config:      config,
            status:      StatusUninitialized,
        },
        baseURL: baseURL,
        headers: make(map[string]string),
    }
    
    return adapter, nil
}

func (a *RESTAdapter) Initialize(ctx context.Context) error {
    a.mutex.Lock()
    defer a.mutex.Unlock()
    
    if a.status != StatusUninitialized {
        return fmt.Errorf("adapter already initialized")
    }
    
    transport := &http.Transport{
        MaxIdleConns:        10,
        IdleConnTimeout:     30 * time.Second,
        DisableCompression:  false,
        TLSHandshakeTimeout: 10 * time.Second,
    }
    
    a.client = &http.Client{
        Transport: transport,
        Timeout:   a.config.Timeout,
    }
    
    // Set default headers
    a.headers["User-Agent"] = "QUANT_WebWork_GO-Bridge-REST-Adapter/1.0"
    a.headers["Accept"] = "application/json"
    
    // Apply custom headers from config
    if customHeaders, ok := a.config.Options["headers"].(map[string]string); ok {
        for k, v := range customHeaders {
            a.headers[k] = v
        }
    }
    
    a.status = StatusInitialized
    return nil
}

func (a *RESTAdapter) Connect(ctx context.Context) error {
    a.mutex.Lock()
    defer a.mutex.Unlock()
    
    if a.status != StatusInitialized && a.status != StatusDisconnected {
        return fmt.Errorf("adapter not ready for connection")
    }
    
    // Perform a health check request to verify connectivity
    req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL, nil)
    if err != nil {
        a.lastError = err
        return err
    }
    
    // Add headers
    for k, v := range a.headers {
        req.Header.Set(k, v)
    }
    
    resp, err := a.client.Do(req)
    if err != nil {
        a.lastError = err
        a.status = StatusError
        return err
    }
    defer resp.Body.Close()
    
    a.status = StatusConnected
    a.stats.ConnectCount++
    a.stats.LastConnectTime = time.Now()
    
    return nil
}

func (a *RESTAdapter) Send(ctx context.Context, data []byte) ([]byte, error) {
    a.mutex.RLock()
    if a.status != StatusConnected {
        a.mutex.RUnlock()
        return nil, fmt.Errorf("adapter not connected")
    }
    a.mutex.RUnlock()
    
    // Parse request data
    var requestData struct {
        Method  string            `json:"method"`
        Path    string            `json:"path"`
        Headers map[string]string `json:"headers"`
        Body    json.RawMessage   `json:"body"`
    }
    
    if err := json.Unmarshal(data, &requestData); err != nil {
        return nil, err
    }
    
    // Build URL
    url := a.baseURL
    if requestData.Path != "" {
        if !strings.HasPrefix(requestData.Path, "/") {
            url += "/"
        }
        url += requestData.Path
    }
    
    // Create request
    var bodyReader io.Reader
    if len(requestData.Body) > 0 {
        bodyReader = bytes.NewReader(requestData.Body)
    }
    
    req, err := http.NewRequestWithContext(ctx, requestData.Method, url, bodyReader)
    if err != nil {
        return nil, err
    }
    
    // Add default headers
    for k, v := range a.headers {
        req.Header.Set(k, v)
    }
    
    // Add request-specific headers
    for k, v := range requestData.Headers {
        req.Header.Set(k, v)
    }
    
    // Send request
    startTime := time.Now()
    resp, err := a.client.Do(req)
    duration := time.Since(startTime)
    
    // Update stats
    a.mutex.Lock()
    a.stats.MessagesSent++
    a.mutex.Unlock()
    
    if err != nil {
        a.mutex.Lock()
        a.stats.Errors++
        a.lastError = err
        a.mutex.Unlock()
        return nil, err
    }
    
    defer resp.Body.Close()
    
    // Read response body
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    // Update stats
    a.mutex.Lock()
    a.stats.BytesSent += int64(len(data))
    a.stats.BytesReceived += int64(len(respBody))
    a.mutex.Unlock()
    
    // Track response time
    a.updateResponseTime(duration)
    
    // Create response structure
    responseData := struct {
        StatusCode int               `json:"status_code"`
        Headers    map[string]string `json:"headers"`
        Body       json.RawMessage   `json:"body"`
    }{
        StatusCode: resp.StatusCode,
        Headers:    make(map[string]string),
        Body:       respBody,
    }
    
    // Copy headers
    for k, v := range resp.Header {
        if len(v) > 0 {
            responseData.Headers[k] = v[0]
        }
    }
    
    return json.Marshal(responseData)
}

func (a *RESTAdapter) updateResponseTime(duration time.Duration) {
    a.mutex.Lock()
    defer a.mutex.Unlock()
    
    // Update average response time
    count := float64(a.stats.MessagesSent)
    if count == 1 {
        a.stats.AverageResponseTime = duration
        a.stats.MinResponseTime = duration
        a.stats.MaxResponseTime = duration
    } else {
        // Incremental average
        a.stats.AverageResponseTime = time.Duration(
            (float64(a.stats.AverageResponseTime) * (count - 1) + float64(duration)) / count,
        )
        
        // Update min/max
        if duration < a.stats.MinResponseTime {
            a.stats.MinResponseTime = duration
        }
        if duration > a.stats.MaxResponseTime {
            a.stats.MaxResponseTime = duration
        }
    }
}

// Other methods omitted for brevity
```

## Conclusion

This document provides specifications and guidelines for implementing bridge adapters in the QUANT_WebWork_GO system. By following these guidelines, developers can create adapters that integrate seamlessly with the bridge system and provide reliable communication between different components of the application. 