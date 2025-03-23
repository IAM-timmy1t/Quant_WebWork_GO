import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import BridgeVerification from '../BridgeVerification';

// Mock the BridgeClient
jest.mock('../../bridge/BridgeClient', () => {
  const originalModule = jest.requireActual('../../bridge/BridgeClient');
  
  // Mock client instance
  const mockClient = {
    connect: jest.fn().mockResolvedValue(undefined),
    disconnect: jest.fn().mockResolvedValue(undefined),
    getMetrics: jest.fn().mockResolvedValue({
      totalMessages: 42,
      activeConnections: 3,
      messagesPerSecond: 1.5
    }),
    analyzeToken: jest.fn().mockResolvedValue({
      tokenAddress: '0x1234567890abcdef',
      analysisResult: {
        risk: 'low',
        score: 0.2,
        confidence: 0.9
      }
    }),
    sendMessage: jest.fn().mockImplementation((type, content, metadata) => {
      // For testing error handling
      if (type === 'analysis-request' && !metadata.token_address) {
        return Promise.resolve({
          id: 'error-response',
          type: 'error',
          content: '',
          metadata: { error: 'Missing token_address parameter' },
          timestamp: Date.now()
        });
      }
      
      // Default case
      return Promise.resolve({
        id: 'test-response',
        type: `${type}-response`,
        content: 'Test response content',
        metadata: { status: 'success' },
        timestamp: Date.now()
      });
    })
  };
  
  return {
    ...originalModule,
    createBridgeClient: jest.fn().mockReturnValue(mockClient)
  };
});

describe('BridgeVerification Component', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders verification interface with server URL input', () => {
    render(<BridgeVerification />);
    
    expect(screen.getByText(/Bridge Module Verification/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Server URL/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Run Verification/i })).toBeInTheDocument();
  });

  test('updates server URL when input changes', () => {
    render(<BridgeVerification />);
    
    const urlInput = screen.getByLabelText(/Server URL/i);
    fireEvent.change(urlInput, { target: { value: 'http://test-server:8080' } });
    
    expect(urlInput).toHaveValue('http://test-server:8080');
  });

  test('runs verification and shows results when button is clicked', async () => {
    render(<BridgeVerification />);
    
    // Click the verify button
    const verifyButton = screen.getByRole('button', { name: /Run Verification/i });
    fireEvent.click(verifyButton);
    
    // First it should show "Running Tests..."
    expect(screen.getByText(/Running Tests/i)).toBeInTheDocument();
    
    // Then it should complete and show results
    await waitFor(() => {
      // Check test results are displayed
      expect(screen.getByText(/Test Results/i)).toBeInTheDocument();
      expect(screen.getByText(/Verification Log/i)).toBeInTheDocument();
      
      // Connection test should pass
      expect(screen.getByText(/Connection/i)).toBeInTheDocument();
      expect(screen.getByText(/SUCCESS/i)).toBeInTheDocument();
      
      // Metrics test should pass
      expect(screen.getByText(/Metrics Endpoint/i)).toBeInTheDocument();
      
      // The log should have entries
      const logContainer = screen.getByText(/Starting verification with server/).closest('.log-container');
      expect(logContainer).toBeInTheDocument();
    });
  });

  test('displays error when connection fails', async () => {
    // Mock connection failure
    const { createBridgeClient } = require('../../bridge/BridgeClient');
    createBridgeClient.mockReturnValueOnce({
      connect: jest.fn().mockRejectedValue(new Error('Failed to connect to bridge server')),
      disconnect: jest.fn().mockResolvedValue(undefined)
    });
    
    render(<BridgeVerification />);
    
    // Click the verify button
    const verifyButton = screen.getByRole('button', { name: /Run Verification/i });
    fireEvent.click(verifyButton);
    
    await waitFor(() => {
      // Connection test should fail
      expect(screen.getByText(/Connection/i)).toBeInTheDocument();
      expect(screen.getByText(/FAILURE/i)).toBeInTheDocument();
      
      // Error message should be in the log
      expect(screen.getByText(/Failed to connect to bridge server/i)).toBeInTheDocument();
    });
  });

  test('tests error handling by sending invalid request', async () => {
    render(<BridgeVerification />);
    
    // Click the verify button
    const verifyButton = screen.getByRole('button', { name: /Run Verification/i });
    fireEvent.click(verifyButton);
    
    await waitFor(() => {
      // Error handling test should succeed because the server correctly returned an error
      expect(screen.getByText(/Error Handling/i)).toBeInTheDocument();
      expect(screen.getByText(/Server correctly returned error response/i)).toBeInTheDocument();
    });
  });

  test('displays custom message type test results', async () => {
    render(<BridgeVerification />);
    
    // Click the verify button
    const verifyButton = screen.getByRole('button', { name: /Run Verification/i });
    fireEvent.click(verifyButton);
    
    await waitFor(() => {
      // Custom message types test should succeed
      expect(screen.getByText(/Custom Message Types/i)).toBeInTheDocument();
      expect(screen.getByText(/Server handled/i)).toBeInTheDocument();
      
      // Details should be available
      const detailsButton = screen.getAllByText(/Details/i)[0];
      fireEvent.click(detailsButton);
      
      // Details should show the response information
      expect(screen.getByText(/"type":/i)).toBeInTheDocument();
    });
  });
});
