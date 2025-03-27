// message.go - Message structure and handler for bridge communications

package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType defines the type of message
type MessageType string

// Message types
const (
	TypeRequest    MessageType = "request"
	TypeResponse   MessageType = "response"
	TypeEvent      MessageType = "event"
	TypeHeartbeat  MessageType = "heartbeat"
	TypeError      MessageType = "error"
	TypeMetric     MessageType = "metric"
	TypeControl    MessageType = "control"
	TypeLog        MessageType = "log"
	TypeSubscribe  MessageType = "subscribe"
	TypeUnsubscribe MessageType = "unsubscribe"
)

// MessageStatus defines the status of a message
type MessageStatus string

// Message statuses
const (
	StatusPending   MessageStatus = "pending"
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusProcessed MessageStatus = "processed"
	StatusFailed    MessageStatus = "failed"
	StatusRejected  MessageStatus = "rejected"
	StatusTimedOut  MessageStatus = "timed_out"
)

// MessagePriority defines the priority of a message
type MessagePriority int

// Message priorities
const (
	PriorityLow     MessagePriority = 1
	PriorityNormal  MessagePriority = 2
	PriorityHigh    MessagePriority = 3
	PriorityCritical MessagePriority = 4
)

// Message represents a bridge communication message
type Message struct {
	ID          string            `json:"id"`
	Type        MessageType       `json:"type"`
	Status      MessageStatus     `json:"status"`
	Priority    MessagePriority   `json:"priority"`
	Source      *BridgeTarget     `json:"source,omitempty"`
	Destination *BridgeTarget     `json:"destination,omitempty"`
	Payload     []byte            `json:"payload"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Expiration  time.Time         `json:"expiration,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Error       string            `json:"error,omitempty"`
	Attempts    int               `json:"attempts,omitempty"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	var payloadBytes []byte
	var err error

	// Convert payload to bytes if not already
	switch p := payload.(type) {
	case []byte:
		payloadBytes = p
	case string:
		payloadBytes = []byte(p)
	default:
		payloadBytes, err = json.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	return &Message{
		ID:        uuid.New().String(),
		Type:      msgType,
		Status:    StatusPending,
		Priority:  PriorityNormal,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}, nil
}

// SetPayload sets the message payload
func (m *Message) SetPayload(payload interface{}) error {
	var payloadBytes []byte
	var err error

	// Convert payload to bytes if not already
	switch p := payload.(type) {
	case []byte:
		payloadBytes = p
	case string:
		payloadBytes = []byte(p)
	default:
		payloadBytes, err = json.Marshal(p)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	m.Payload = payloadBytes
	return nil
}

// GetPayload decodes the payload into the provided target
func (m *Message) GetPayload(target interface{}) error {
	if len(m.Payload) == 0 {
		return errors.New("empty payload")
	}

	return json.Unmarshal(m.Payload, target)
}

// SetExpiration sets the message expiration time
func (m *Message) SetExpiration(duration time.Duration) {
	m.Expiration = time.Now().Add(duration)
}

// IsExpired checks if the message has expired
func (m *Message) IsExpired() bool {
	return !m.Expiration.IsZero() && time.Now().After(m.Expiration)
}

// AddMetadata adds a key-value pair to the message metadata
func (m *Message) AddMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
}

// GetMetadata gets a metadata value by key
func (m *Message) GetMetadata(key string) (string, bool) {
	if m.Metadata == nil {
		return "", false
	}
	value, exists := m.Metadata[key]
	return value, exists
}

// CreateResponse creates a response message for this message
func (m *Message) CreateResponse(payload interface{}) (*Message, error) {
	response, err := NewMessage(TypeResponse, payload)
	if err != nil {
		return nil, err
	}

	response.CorrelationID = m.ID
	
	// Swap source and destination
	response.Source = m.Destination
	response.Destination = m.Source
	
	return response, nil
}

// CreateErrorResponse creates an error response for this message
func (m *Message) CreateErrorResponse(errMsg string) (*Message, error) {
	response, err := NewMessage(TypeError, nil)
	if err != nil {
		return nil, err
	}

	response.CorrelationID = m.ID
	response.Error = errMsg
	
	// Swap source and destination
	response.Source = m.Destination
	response.Destination = m.Source
	
	return response, nil
}

// MessageHandler is a function that processes a message
type MessageHandler func(ctx context.Context, msg *Message) (*Message, error)

// MessageRouter routes messages to handlers based on message type or other criteria
type MessageRouter struct {
	handlers       map[MessageType]MessageHandler
	defaultHandler MessageHandler
}

// NewMessageRouter creates a new message router
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers: make(map[MessageType]MessageHandler),
	}
}

// RegisterHandler registers a handler for a specific message type
func (r *MessageRouter) RegisterHandler(msgType MessageType, handler MessageHandler) {
	r.handlers[msgType] = handler
}

// RegisterDefaultHandler registers a handler for messages with no specific handler
func (r *MessageRouter) RegisterDefaultHandler(handler MessageHandler) {
	r.defaultHandler = handler
}

// Route routes a message to the appropriate handler
func (r *MessageRouter) Route(ctx context.Context, msg *Message) (*Message, error) {
	// Check for a specific handler
	if handler, exists := r.handlers[msg.Type]; exists {
		return handler(ctx, msg)
	}

	// Use default handler if available
	if r.defaultHandler != nil {
		return r.defaultHandler(ctx, msg)
	}

	// No handler available
	return msg.CreateErrorResponse(fmt.Sprintf("no handler registered for message type: %s", msg.Type))
}
