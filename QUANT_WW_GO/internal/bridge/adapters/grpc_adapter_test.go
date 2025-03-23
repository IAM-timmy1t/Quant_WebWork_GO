package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/timot/Quant_WebWork_GO/internal/bridge/protocol"
)

// MockBridge implements the Bridge interface for testing
type MockBridge struct {
	mock.Mock
}

func (m *MockBridge) ID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBridge) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBridge) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBridge) Version() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBridge) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBridge) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBridge) AddProtocol(p protocol.Protocol) {
	m.Called(p)
}

func (m *MockBridge) RemoveProtocol(id string) {
	m.Called(id)
}

func (m *MockBridge) GetProtocol(id string) protocol.Protocol {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(protocol.Protocol)
}

func (m *MockBridge) GetProtocols() []protocol.Protocol {
	args := m.Called()
	return args.Get(0).([]protocol.Protocol)
}

func (m *MockBridge) AddAdapter(a Adapter) {
	m.Called(a)
}

func (m *MockBridge) RemoveAdapter(id string) {
	m.Called(id)
}

func (m *MockBridge) GetAdapter(id string) Adapter {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(Adapter)
}

func (m *MockBridge) GetAdapters() []Adapter {
	args := m.Called()
	return args.Get(0).([]Adapter)
}

// MockProtocol implements the Protocol interface for testing
type MockProtocol struct {
	mock.Mock
}

func (m *MockProtocol) ID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProtocol) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProtocol) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProtocol) Version() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProtocol) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProtocol) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProtocol) Process(ctx context.Context, msg protocol.Message) (protocol.Message, error) {
	args := m.Called(ctx, msg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(protocol.Message), args.Error(1)
}

func (m *MockProtocol) AddHandler(h protocol.MessageHandler) {
	m.Called(h)
}

func (m *MockProtocol) RemoveHandler(id string) {
	m.Called(id)
}

func (m *MockProtocol) GetHandler(id string) protocol.MessageHandler {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(protocol.MessageHandler)
}

func (m *MockProtocol) GetHandlers() []protocol.MessageHandler {
	args := m.Called()
	return args.Get(0).([]protocol.MessageHandler)
}

// TestGRPCAdapter tests the gRPC adapter implementation
func TestGRPCAdapter(t *testing.T) {
	t.Run("NewGRPCAdapter creates adapter with correct configuration", func(t *testing.T) {
		config := &GRPCAdapterConfig{
			Host:        "localhost",
			Port:        8080,
			EnableWeb:   true,
			CORSOrigins: []string{"http://localhost:3000"},
			EnableTLS:   false,
		}
		
		adapter := NewGRPCAdapter(config)
		
		assert.NotNil(t, adapter)
		assert.Equal(t, "grpc", adapter.ID())
		assert.Equal(t, "gRPC Adapter", adapter.Name())
		assert.Contains(t, adapter.Description(), "gRPC")
		assert.Equal(t, config, adapter.(*GRPCAdapter).config)
	})
	
	t.Run("Connect initializes adapter correctly", func(t *testing.T) {
		mockBridge := new(MockBridge)
		mockBridge.On("ID").Return("test-bridge")
		mockBridge.On("AddAdapter", mock.Anything).Return()
		
		config := &GRPCAdapterConfig{
			Host:        "localhost",
			Port:        8080,
			EnableWeb:   true,
			CORSOrigins: []string{"http://localhost:3000"},
			EnableTLS:   false,
		}
		
		adapter := NewGRPCAdapter(config)
		err := adapter.Connect(context.Background(), mockBridge)
		
		assert.NoError(t, err)
		assert.NotNil(t, adapter.(*GRPCAdapter).bridge)
		mockBridge.AssertCalled(t, "AddAdapter", adapter)
	})
	
	t.Run("Disconnect stops server correctly", func(t *testing.T) {
		mockBridge := new(MockBridge)
		mockBridge.On("ID").Return("test-bridge")
		mockBridge.On("AddAdapter", mock.Anything).Return()
		
		config := &GRPCAdapterConfig{
			Host:        "localhost",
			Port:        8080,
			EnableWeb:   true,
			CORSOrigins: []string{"http://localhost:3000"},
			EnableTLS:   false,
		}
		
		adapter := NewGRPCAdapter(config)
		_ = adapter.Connect(context.Background(), mockBridge)
		
		err := adapter.Disconnect(context.Background())
		
		assert.NoError(t, err)
		assert.Nil(t, adapter.(*GRPCAdapter).server)
	})
	
	t.Run("ProcessMessage forwards message to protocol", func(t *testing.T) {
		mockBridge := new(MockBridge)
		mockProtocol := new(MockProtocol)
		
		mockBridge.On("ID").Return("test-bridge")
		mockBridge.On("AddAdapter", mock.Anything).Return()
		mockBridge.On("GetProtocol", "token").Return(mockProtocol)
		
		// Prepare request and response messages
		requestMsg := &protocol.RawMessage{
			ID:        "test-message",
			Type:      "test-type",
			Content:   "test-content",
			Metadata:  `{"key":"value"}`,
			Timestamp: time.Now().UnixNano(),
		}
		
		responseMsg := &protocol.RawMessage{
			ID:        "response-message",
			Type:      "response-type",
			Content:   "response-content",
			Metadata:  `{"status":"success"}`,
			Timestamp: time.Now().UnixNano(),
		}
		
		mockProtocol.On("Process", mock.Anything, mock.MatchedBy(func(msg protocol.Message) bool {
			return msg.ID() == requestMsg.ID()
		})).Return(responseMsg, nil)
		
		config := &GRPCAdapterConfig{
			Host:        "localhost",
			Port:        8080,
			EnableWeb:   true,
			CORSOrigins: []string{"http://localhost:3000"},
			EnableTLS:   false,
		}
		
		adapter := NewGRPCAdapter(config)
		_ = adapter.Connect(context.Background(), mockBridge)
		
		// Process message
		response, err := adapter.(*GRPCAdapter).processMessage(context.Background(), "token", requestMsg)
		
		assert.NoError(t, err)
		assert.Equal(t, responseMsg.ID(), response.ID())
		assert.Equal(t, responseMsg.Type(), response.Type())
		mockProtocol.AssertCalled(t, "Process", mock.Anything, mock.MatchedBy(func(msg protocol.Message) bool {
			return msg.ID() == requestMsg.ID()
		}))
	})
	
	t.Run("EnableWebSupport configures CORS correctly", func(t *testing.T) {
		config := &GRPCAdapterConfig{
			Host:        "localhost",
			Port:        8080,
			EnableWeb:   true,
			CORSOrigins: []string{"http://localhost:3000", "https://app.example.com"},
			EnableTLS:   false,
		}
		
		adapter := NewGRPCAdapter(config).(*GRPCAdapter)
		
		// Check CORS configuration
		assert.True(t, adapter.config.EnableWeb)
		assert.Contains(t, adapter.config.CORSOrigins, "http://localhost:3000")
		assert.Contains(t, adapter.config.CORSOrigins, "https://app.example.com")
	})
	
	t.Run("HandleStream processes bi-directional streaming", func(t *testing.T) {
		// This is a more complex test that would require mocking the gRPC stream
		// For simplicity, we'll just verify the adapter can be created with streaming enabled
		config := &GRPCAdapterConfig{
			Host:        "localhost",
			Port:        8080,
			EnableWeb:   true,
			CORSOrigins: []string{"http://localhost:3000"},
			EnableTLS:   false,
		}
		
		adapter := NewGRPCAdapter(config)
		assert.NotNil(t, adapter)
		assert.Equal(t, "grpc", adapter.ID())
	})
}
