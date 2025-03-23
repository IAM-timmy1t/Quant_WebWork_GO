// bridge.go - Cross-language integration framework

package bridge

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
)

// Bridge provides a unified interface for cross-language communication
type Bridge struct {
	name           string
	adapters       map[string]Adapter
	protocols      map[string]Protocol
	config         *Config
	metricsCollector *metrics.BridgeMetrics
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewBridge creates a new bridge with the given configuration
func NewBridge(name string, config *Config) *Bridge {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	bridge := &Bridge{
		name:      name,
		adapters:  make(map[string]Adapter),
		protocols: make(map[string]Protocol),
		config:    config,
		metricsCollector: metrics.NewBridgeMetrics(name),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	return bridge
}

// RegisterAdapter adds a language adapter to the bridge
func (b *Bridge) RegisterAdapter(name string, adapter Adapter) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, exists := b.adapters[name]; exists {
		return fmt.Errorf("adapter '%s' already registered", name)
	}
	
	b.adapters[name] = adapter
	return nil
}

// RegisterProtocol adds a communication protocol to the bridge
func (b *Bridge) RegisterProtocol(name string, protocol Protocol) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, exists := b.protocols[name]; exists {
		return fmt.Errorf("protocol '%s' already registered", name)
	}
	
	b.protocols[name] = protocol
	return nil
}

// SetMetricsCollector sets the metrics collector for the bridge
func (b *Bridge) SetMetricsCollector(collector *metrics.Collector) {
	b.metricsCollector.SetCollector(collector)
}

// Start initializes all adapters and protocols
func (b *Bridge) Start(ctx context.Context) error {
	parentCtx, cancel := context.WithCancel(ctx)
	b.ctx = parentCtx
	b.cancel = cancel
	
	// Initialize adapters
	for name, adapter := range b.adapters {
		if err := adapter.Initialize(b.ctx); err != nil {
			return fmt.Errorf("failed to initialize adapter '%s': %w", name, err)
		}
	}
	
	// Initialize protocols
	for name, protocol := range b.protocols {
		if err := protocol.Initialize(b.ctx); err != nil {
			return fmt.Errorf("failed to initialize protocol '%s': %w", name, err)
		}
	}
	
	return nil
}

// Stop gracefully shuts down the bridge
func (b *Bridge) Stop() {
	b.cancel()
}

// Call invokes a function through the appropriate adapter and protocol
func (b *Bridge) Call(ctx context.Context, target Target, function string, params interface{}) (interface{}, error) {
	b.mu.RLock()
	adapter, adapterExists := b.adapters[target.Adapter]
	protocol, protocolExists := b.protocols[target.Protocol]
	b.mu.RUnlock()
	
	if !adapterExists {
		return nil, fmt.Errorf("adapter '%s' not found", target.Adapter)
	}
	
	if !protocolExists {
		return nil, fmt.Errorf("protocol '%s' not found", target.Protocol)
	}
	
	// Prepare request
	request := &Request{
		Target:    target,
		Function:  function,
		Params:    params,
		Timestamp: time.Now(),
	}
	
	startTime := time.Now()
	
	// Encode request using protocol
	encodedRequest, err := protocol.Encode(request)
	if err != nil {
		b.metricsCollector.RecordError("encode_error")
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}
	
	// Send request through adapter
	encodedResponse, err := adapter.Send(ctx, encodedRequest)
	if err != nil {
		b.metricsCollector.RecordError("send_error")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Decode response using protocol
	response, err := protocol.Decode(encodedResponse)
	if err != nil {
		b.metricsCollector.RecordError("decode_error")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	elapsed := time.Since(startTime).Milliseconds()
	b.metricsCollector.RecordRequest(function, float64(elapsed), err == nil)
	
	return response.Result, response.Error
}

// CallAsync asynchronously invokes a function
func (b *Bridge) CallAsync(ctx context.Context, target Target, function string, params interface{}, callback func(interface{}, error)) {
	go func() {
		result, err := b.Call(ctx, target, function, params)
		if callback != nil {
			callback(result, err)
		}
	}()
}

// Subscribe sets up a subscription to events from a target
func (b *Bridge) Subscribe(ctx context.Context, target Target, eventType string, handler func(Event)) (Subscription, error) {
	b.mu.RLock()
	adapter, adapterExists := b.adapters[target.Adapter]
	protocol, protocolExists := b.protocols[target.Protocol]
	b.mu.RUnlock()
	
	if !adapterExists {
		return nil, fmt.Errorf("adapter '%s' not found", target.Adapter)
	}
	
	if !protocolExists {
		return nil, fmt.Errorf("protocol '%s' not found", target.Protocol)
	}
	
	// Create subscription request
	subRequest := &SubscriptionRequest{
		Target:    target,
		EventType: eventType,
		Timestamp: time.Now(),
	}
	
	// Encode request using protocol
	encodedRequest, err := protocol.EncodeSubscription(subRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to encode subscription request: %w", err)
	}
	
	// Create channel for events
	eventCh := make(chan []byte, b.config.EventBufferSize)
	
	// Set up adapter subscription
	subID, err := adapter.Subscribe(ctx, encodedRequest, eventCh)
	if err != nil {
		close(eventCh)
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	
	// Start handler goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case encodedEvent, ok := <-eventCh:
				if !ok {
					return
				}
				
				// Decode event using protocol
				event, err := protocol.DecodeEvent(encodedEvent)
				if err != nil {
					b.metricsCollector.RecordError("decode_event_error")
					continue
				}
				
				// Call handler
				handler(event)
			}
		}
	}()
	
	// Create subscription object
	subscription := &bridgeSubscription{
		id:      subID,
		adapter: adapter,
		eventCh: eventCh,
	}
	
	return subscription, nil
}

// bridgeSubscription implements the Subscription interface
type bridgeSubscription struct {
	id      string
	adapter Adapter
	eventCh chan []byte
}

// Unsubscribe cancels the subscription
func (s *bridgeSubscription) Unsubscribe() error {
	err := s.adapter.Unsubscribe(s.id)
	close(s.eventCh)
	return err
}

// GetID returns the subscription ID
func (s *bridgeSubscription) GetID() string {
	return s.id
}

// DefaultConfig returns the default bridge configuration
func DefaultConfig() *Config {
	return &Config{
		EventBufferSize: 100,
		Timeout:         30 * time.Second,
	}
}
