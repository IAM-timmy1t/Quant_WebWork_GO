package eventbus

import (
	"context"
	"sync"
	"time"
)

// Event represents a message in the event bus
type Event struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
}

// Subscriber is a function that handles events
type Subscriber func(Event)

// EventBus manages event publishing and subscription
type EventBus struct {
	subscribers map[string][]Subscriber
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// New creates a new event bus instance
func New() *EventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventBus{
		subscribers: make(map[string][]Subscriber),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Subscribe registers a subscriber for specific event types
func (b *EventBus) Subscribe(eventType string, subscriber Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventType] = append(b.subscribers[eventType], subscriber)
}

// Publish sends an event to all subscribers
func (b *EventBus) Publish(eventType string, data interface{}) {
	event := Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	b.mu.RLock()
	subscribers := b.subscribers[eventType]
	b.mu.RUnlock()

	// Notify subscribers concurrently
	for _, subscriber := range subscribers {
		go func(s Subscriber) {
			s(event)
		}(subscriber)
	}
}

// Shutdown gracefully stops the event bus
func (b *EventBus) Shutdown() {
	b.cancel()
}

// UnsubscribeAll removes all subscribers for a specific event type
func (b *EventBus) UnsubscribeAll(eventType string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, eventType)
}

// SubscriberCount returns the number of subscribers for an event type
func (b *EventBus) SubscriberCount(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subscribers[eventType])
}
