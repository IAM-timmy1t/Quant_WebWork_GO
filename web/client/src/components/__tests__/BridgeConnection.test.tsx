import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import BridgeConnection from '../BridgeConnection';

// Mock the BridgeClient
jest.mock('../../bridge/BridgeClient', () => {
  const originalModule = jest.requireActual('../../bridge/BridgeClient');
  
  // Mock client instance
  const mockClient = {
    connect: jest.fn().mockResolvedValue(undefined),
    disconnect: jest.fn().mockResolvedValue(undefined),
    sendMessage: jest.fn().mockResolvedValue({
      id: 'test-response',
      type: 'test-response',
      content: 'Test response content',
      metadata: { status: 'success' },
      timestamp: Date.now()
    }),
    analyzeToken: jest.fn().mockResolvedValue({
      tokenAddress: '0x1234567890abcdef',
      analysisResult: {
        risk: 'low',
        score: 0.2,
        confidence: 0.9
      }
    }),
    getMetrics: jest.fn().mockResolvedValue({
      totalMessages: 42,
      activeConnections: 3,
      messagesPerSecond: 1.5
    })
  };
  
  return {
    ...originalModule,
    createBridgeClient: jest.fn().mockReturnValue(mockClient)
  };
});

describe('BridgeConnection Component', () => {
  const defaultProps = {
    serverUrl: 'http://localhost:8080',
    onConnect: jest.fn(),
    onDisconnect: jest.fn(),
    onError: jest.fn(),
    onMessage: jest.fn()
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders connection form with server URL', () => {
    render(<BridgeConnection {...defaultProps} />);
    
    expect(screen.getByText(/Bridge Connection/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /connect/i })).toBeInTheDocument();
    expect(screen.getByText(/server url/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue('http://localhost:8080')).toBeInTheDocument();
  });

  test('connects to bridge server when Connect button is clicked', async () => {
    render(<BridgeConnection {...defaultProps} />);
    
    const connectButton = screen.getByRole('button', { name: /connect/i });
    fireEvent.click(connectButton);
    
    await waitFor(() => {
      // Check that onConnect was called
      expect(defaultProps.onConnect).toHaveBeenCalledTimes(1);
      
      // Button should now say Disconnect
      expect(screen.getByRole('button', { name: /disconnect/i })).toBeInTheDocument();
      
      // Connection status should be updated
      expect(screen.getByText(/connected/i)).toBeInTheDocument();
    });
  });

  test('disconnects from bridge server when Disconnect button is clicked', async () => {
    render(<BridgeConnection {...defaultProps} />);
    
    // First connect
    const connectButton = screen.getByRole('button', { name: /connect/i });
    fireEvent.click(connectButton);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /disconnect/i })).toBeInTheDocument();
    });
    
    // Then disconnect
    const disconnectButton = screen.getByRole('button', { name: /disconnect/i });
    fireEvent.click(disconnectButton);
    
    await waitFor(() => {
      // Check that onDisconnect was called
      expect(defaultProps.onDisconnect).toHaveBeenCalledTimes(1);
      
      // Button should now say Connect again
      expect(screen.getByRole('button', { name: /connect/i })).toBeInTheDocument();
      
      // Connection status should be updated
      expect(screen.getByText(/disconnected/i)).toBeInTheDocument();
    });
  });

  test('sends a message when Send Message button is clicked', async () => {
    render(<BridgeConnection {...defaultProps} />);
    
    // First connect
    const connectButton = screen.getByRole('button', { name: /connect/i });
    fireEvent.click(connectButton);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /disconnect/i })).toBeInTheDocument();
    });
    
    // Fill out message form
    const messageInput = screen.getByLabelText(/message/i);
    fireEvent.change(messageInput, { target: { value: 'Test message' } });
    
    const messageTypeSelect = screen.getByLabelText(/message type/i);
    fireEvent.change(messageTypeSelect, { target: { value: 'query' } });
    
    // Send message
    const sendButton = screen.getByRole('button', { name: /send message/i });
    fireEvent.click(sendButton);
    
    await waitFor(() => {
      // Check that onMessage was called with the response
      expect(defaultProps.onMessage).toHaveBeenCalledWith(expect.objectContaining({
        id: 'test-response',
        type: 'test-response'
      }));
      
      // Message response should be displayed
      expect(screen.getByText(/response received/i)).toBeInTheDocument();
    });
  });

  test('analyzes a token when Analyze Token button is clicked', async () => {
    render(<BridgeConnection {...defaultProps} />);
    
    // First connect
    const connectButton = screen.getByRole('button', { name: /connect/i });
    fireEvent.click(connectButton);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /disconnect/i })).toBeInTheDocument();
    });
    
    // Fill out token address
    const tokenInput = screen.getByLabelText(/token address/i);
    fireEvent.change(tokenInput, { target: { value: '0x1234567890abcdef' } });
    
    // Analyze token
    const analyzeButton = screen.getByRole('button', { name: /analyze token/i });
    fireEvent.click(analyzeButton);
    
    await waitFor(() => {
      // Analysis result should be displayed
      expect(screen.getByText(/analysis result/i)).toBeInTheDocument();
      expect(screen.getByText(/risk: low/i)).toBeInTheDocument();
    });
  });

  test('displays error message when connection fails', async () => {
    // Mock implementation to simulate error
    const { createBridgeClient } = require('../../bridge/BridgeClient');
    createBridgeClient.mockReturnValueOnce({
      connect: jest.fn().mockRejectedValue(new Error('Connection failed')),
      disconnect: jest.fn().mockResolvedValue(undefined)
    });
    
    render(<BridgeConnection {...defaultProps} />);
    
    const connectButton = screen.getByRole('button', { name: /connect/i });
    fireEvent.click(connectButton);
    
    await waitFor(() => {
      // Check that onError was called
      expect(defaultProps.onError).toHaveBeenCalledWith(expect.any(Error));
      
      // Error message should be displayed
      expect(screen.getByText(/connection failed/i)).toBeInTheDocument();
    });
  });
});
