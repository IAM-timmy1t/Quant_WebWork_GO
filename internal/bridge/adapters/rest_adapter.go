// rest_adapter.go - REST adapter implementation for bridge system

package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
)

// Record metrics extensions for metrics.Collector
// These are temporary extensions until the metrics package is properly implemented

// RecordHTTP records HTTP request metrics
func (c *metrics.Collector) RecordHTTP(method, url string, statusCode int, durationSec float64) {
	tags := map[string]string{
		"method": method,
		"url":    url,
		"status": fmt.Sprintf("%d", statusCode),
	}
	c.Collect("rest_request", "request_duration", durationSec, tags)
}

// RESTAdapterConfig contains configuration for the REST adapter
type RESTAdapterConfig struct {
	BaseURL         string
	Timeout         time.Duration
	Headers         map[string]string
	DefaultEndpoint string
	RetryCount      int
	RetryDelay      time.Duration
	MaxConnections  int
	KeepAlive       time.Duration
	TLSSkipVerify   bool
}

// DefaultRESTAdapterConfig returns the default configuration
func DefaultRESTAdapterConfig() *RESTAdapterConfig {
	return &RESTAdapterConfig{
		Timeout:        30 * time.Second,
		Headers:        make(map[string]string),
		RetryCount:     3,
		RetryDelay:     time.Second,
		MaxConnections: 10,
		KeepAlive:      60 * time.Second,
	}
}

// RESTAdapter implements a REST communication adapter
type RESTAdapter struct {
	name             string
	config           *RESTAdapterConfig
	client           *http.Client
	metricsCollector *metrics.Collector
	requestsMutex    sync.Mutex
	initialized      bool
	activeRequests   map[string]context.CancelFunc
	logger           AdapterLogger
}

// NewRESTAdapter creates a new REST adapter
func NewRESTAdapter(name string, config *RESTAdapterConfig, metricsCollector *metrics.Collector, logger AdapterLogger) (*RESTAdapter, error) {
	if config == nil {
		config = DefaultRESTAdapterConfig()
	}

	// Validate configuration
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL cannot be empty")
	}

	adapter := &RESTAdapter{
		name:             name,
		config:           config,
		metricsCollector: metricsCollector,
		activeRequests:   make(map[string]context.CancelFunc),
		logger:           logger,
	}

	return adapter, nil
}

// Initialize initializes the adapter
func (a *RESTAdapter) Initialize(ctx context.Context) error {
	a.logger.Info(fmt.Sprintf("Initializing REST adapter '%s'", a.name), nil)

	// Create HTTP transport with custom settings
	transport := &http.Transport{
		MaxIdleConns:        a.config.MaxConnections,
		MaxIdleConnsPerHost: a.config.MaxConnections,
		IdleConnTimeout:     a.config.KeepAlive,
	}

	// Create HTTP client
	a.client = &http.Client{
		Timeout:   a.config.Timeout,
		Transport: transport,
	}

	a.initialized = true
	return nil
}

// Name returns the adapter name
func (a *RESTAdapter) Name() string {
	return a.name
}

// Type returns the adapter type
func (a *RESTAdapter) Type() string {
	return "rest"
}

// Send sends a REST request
func (a *RESTAdapter) Send(ctx context.Context, data []byte) ([]byte, error) {
	if !a.initialized {
		return nil, fmt.Errorf("adapter not initialized")
	}

	// Parse request data
	var requestData struct {
		Method   string            `json:"method"`
		Endpoint string            `json:"endpoint"`
		Headers  map[string]string `json:"headers"`
		Body     json.RawMessage   `json:"body"`
	}

	if err := json.Unmarshal(data, &requestData); err != nil {
		return nil, fmt.Errorf("failed to parse request data: %w", err)
	}

	// Set default method if not specified
	if requestData.Method == "" {
		requestData.Method = "GET"
	}

	// Set default endpoint if not specified
	if requestData.Endpoint == "" {
		requestData.Endpoint = a.config.DefaultEndpoint
	}

	// Construct full URL
	url := a.config.BaseURL
	if url[len(url)-1] != '/' && requestData.Endpoint[0] != '/' {
		url += "/"
	}
	url += requestData.Endpoint

	// Create request
	var reqBody io.Reader
	if len(requestData.Body) > 0 && requestData.Body[0] != 'n' { // Check if not "null"
		reqBody = bytes.NewReader(requestData.Body)
	}

	req, err := http.NewRequestWithContext(ctx, requestData.Method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add default headers
	for key, value := range a.config.Headers {
		req.Header.Set(key, value)
	}

	// Add request-specific headers
	for key, value := range requestData.Headers {
		req.Header.Set(key, value)
	}

	// Set content type if not already set
	if req.Header.Get("Content-Type") == "" && reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Register active request
	requestID := fmt.Sprintf("%s-%d", requestData.Method, time.Now().UnixNano())
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.requestsMutex.Lock()
	a.activeRequests[requestID] = cancel
	a.requestsMutex.Unlock()

	defer func() {
		a.requestsMutex.Lock()
		delete(a.activeRequests, requestID)
		a.requestsMutex.Unlock()
	}()

	// Execute request with retry logic
	var response *http.Response
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= a.config.RetryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-requestCtx.Done():
				return nil, requestCtx.Err()
			case <-time.After(a.config.RetryDelay):
				// Continue with retry
			}
		}

		response, err = a.client.Do(req)
		if err == nil {
			break
		}

		lastErr = err
		a.logger.Warn(fmt.Sprintf("REST request failed (attempt %d/%d): %v",
			attempt+1, a.config.RetryCount+1, err), map[string]interface{}{
			"url":     url,
			"method":  requestData.Method,
			"adapter": a.name,
		})
	}

	if response == nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", a.config.RetryCount+1, lastErr)
	}

	// Record metrics
	if a.metricsCollector != nil {
		duration := time.Since(startTime).Seconds()
		a.metricsCollector.RecordHTTP(requestData.Method, url, response.StatusCode, duration)
	}

	// Read response body
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error status codes
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status code %d: %s",
			response.StatusCode, string(respBody))
	}

	// Create response data
	responseData := struct {
		StatusCode int               `json:"status_code"`
		Headers    map[string]string `json:"headers"`
		Body       json.RawMessage   `json:"body"`
	}{
		StatusCode: response.StatusCode,
		Headers:    make(map[string]string),
		Body:       respBody,
	}

	// Copy headers
	for key, values := range response.Header {
		if len(values) > 0 {
			responseData.Headers[key] = values[0]
		}
	}

	// Serialize response
	return json.Marshal(responseData)
}

// Receive is not used for the REST adapter as it's request/response based
func (a *RESTAdapter) Receive(ctx context.Context) ([]byte, error) {
	return nil, fmt.Errorf("receive operation not supported for REST adapter")
}

// CancelRequest cancels an ongoing request
func (a *RESTAdapter) CancelRequest(requestID string) error {
	a.requestsMutex.Lock()
	defer a.requestsMutex.Unlock()

	cancel, exists := a.activeRequests[requestID]
	if !exists {
		return fmt.Errorf("request not found: %s", requestID)
	}

	cancel()
	delete(a.activeRequests, requestID)
	return nil
}

// Close closes the adapter
func (a *RESTAdapter) Close() error {
	a.requestsMutex.Lock()
	defer a.requestsMutex.Unlock()

	// Cancel all active requests
	for _, cancel := range a.activeRequests {
		cancel()
	}
	a.activeRequests = make(map[string]context.CancelFunc)

	a.initialized = false
	return nil
}

// AdapterLogger interface for adapter logging
type AdapterLogger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// Config returns adapter configuration
func (a *RESTAdapter) Config() map[string]interface{} {
	return map[string]interface{}{
		"name":        a.name,
		"type":        "rest",
		"base_url":    a.config.BaseURL,
		"timeout":     a.config.Timeout.String(),
		"retry":       a.config.RetryCount,
		"connections": a.config.MaxConnections,
	}
}

// Extend metrics.Collector with required methods for the adapter
// These should be implemented properly in the metrics package
func init() {
	// This registration will be moved to a central adapter registry implementation
	// RegisterAdapterFactory("rest", func(config AdapterConfig) (Adapter, error) {
	//     // Implementation will go here once the AdapterFactory and AdapterConfig are properly defined
	// })
}
