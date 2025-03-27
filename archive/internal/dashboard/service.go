package dashboard

import (
	"context"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/monitoring"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/storage"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/tracing"
)

// Service manages dashboard state and operations
type Service struct {
	mu sync.RWMutex

	// Dependencies
	monitor  *monitoring.ResourceMonitor
	security *monitoring.SecurityMonitor
	storage  *storage.MetricsStorage
	tracer   *tracing.Tracer

	// Channels for real-time updates
	metricsChan  chan monitoring.ResourceMetrics
	securityChan chan monitoring.SecurityEvent
	
	// Subscribers for updates
	subscribers map[chan<- Message]struct{}
	
	// Control
	stopChan chan struct{}
}

// NewService creates a new dashboard service
func NewService(monitor *monitoring.ResourceMonitor, security *monitoring.SecurityMonitor, storage *storage.MetricsStorage, tracer *tracing.Tracer) *Service {
	return &Service{
		monitor:      monitor,
		security:    security,
		storage:     storage,
		tracer:      tracer,
		metricsChan: make(chan monitoring.ResourceMetrics, 100),
		securityChan: make(chan monitoring.SecurityEvent, 100),
		subscribers: make(map[chan<- Message]struct{}),
		stopChan:    make(chan struct{}),
	}
}

// Start begins processing metrics and events
func (s *Service) Start(ctx context.Context) {
	go s.run(ctx)
}

// Stop stops the service and closes all subscriptions
func (s *Service) Stop() {
	close(s.stopChan)
}

// Subscribe creates a new subscription for dashboard updates
func (s *Service) Subscribe(bufSize int) chan Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan Message, bufSize)
	s.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscription
func (s *Service) Unsubscribe(ch chan Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, ch)
	close(ch)
}

// GetLatestMetrics returns the most recent system metrics
func (s *Service) GetLatestMetrics(ctx context.Context) (monitoring.ResourceMetrics, error) {
	return s.monitor.GetLatestMetrics()
}

// GetMetricsRange returns metrics for a specific time range
func (s *Service) GetMetricsRange(ctx context.Context, start, end time.Time) ([]monitoring.ResourceMetrics, error) {
	return s.storage.GetMetricsRange(ctx, start, end)
}

// GetSecurityEvents returns security events for a time range
func (s *Service) GetSecurityEvents(ctx context.Context, start, end time.Time) ([]monitoring.SecurityEvent, error) {
	return s.storage.GetSecurityEvents(ctx, start, end)
}

// GetAggregatedMetrics returns aggregated metrics based on the specified period
func (s *Service) GetAggregatedMetrics(ctx context.Context, start, end time.Time, aggregation MetricsAggregation) ([]monitoring.ResourceMetrics, error) {
	metrics, err := s.storage.GetMetricsRange(ctx, start, end)
	if err != nil {
		return nil, err
	}

	// Implement aggregation logic based on the specified period
	// This is a placeholder for the actual implementation
	return metrics, nil
}

func (s *Service) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case metrics := <-s.metricsChan:
			s.broadcast(Message{
				Type:    MessageTypeMetrics,
				Payload: metrics,
				Time:    time.Now(),
			})
		case event := <-s.securityChan:
			s.broadcast(Message{
				Type:    MessageTypeSecurityEvent,
				Payload: event,
				Time:    time.Now(),
			})
		}
	}
}

func (s *Service) broadcast(msg Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for ch := range s.subscribers {
		select {
		case ch <- msg:
		default:
			// Channel is full, skip this message for this subscriber
		}
	}
}

