import React, { useState, useEffect } from 'react';
import { createBridgeClient, MessageType, BridgeMessage } from '../bridge/BridgeClient';

interface VerificationResult {
  name: string;
  status: 'success' | 'failure' | 'pending';
  message: string;
  timestamp: Date;
  details?: any;
}

const BridgeVerification: React.FC = () => {
  const [results, setResults] = useState<VerificationResult[]>([]);
  const [isVerifying, setIsVerifying] = useState(false);
  const [serverUrl, setServerUrl] = useState('http://localhost:8080');
  const [log, setLog] = useState<string[]>([]);

  // Add log entry
  const addLog = (message: string) => {
    setLog(prev => [
      `[${new Date().toLocaleTimeString()}] ${message}`,
      ...prev
    ]);
  };

  // Add verification result
  const addResult = (result: VerificationResult) => {
    setResults(prev => [...prev, result]);
    addLog(`Test '${result.name}': ${result.status} - ${result.message}`);
  };

  // Clear results and logs
  const clearResults = () => {
    setResults([]);
    setLog([]);
  };

  // Run verification tests
  const runVerification = async () => {
    clearResults();
    setIsVerifying(true);
    addLog(`Starting verification with server: ${serverUrl}`);

    try {
      const client = createBridgeClient({
        serverUrl,
        onConnect: () => addLog('Connected to bridge server'),
        onDisconnect: () => addLog('Disconnected from bridge server'),
        onError: (error) => addLog(`Error: ${error.message}`),
      });

      // Test 1: Connection
      try {
        addLog('Testing connection...');
        await client.connect();
        addResult({
          name: 'Connection',
          status: 'success',
          message: 'Successfully connected to bridge server',
          timestamp: new Date(),
        });
      } catch (error) {
        addResult({
          name: 'Connection',
          status: 'failure',
          message: `Failed to connect: ${(error as Error).message}`,
          timestamp: new Date(),
        });
        setIsVerifying(false);
        return;
      }

      // Test 2: Metrics Endpoint
      try {
        addLog('Testing metrics endpoint...');
        const metrics = await client.getMetrics();
        addResult({
          name: 'Metrics Endpoint',
          status: 'success',
          message: 'Successfully retrieved metrics',
          timestamp: new Date(),
          details: metrics,
        });
      } catch (error) {
        addResult({
          name: 'Metrics Endpoint',
          status: 'failure',
          message: `Failed to get metrics: ${(error as Error).message}`,
          timestamp: new Date(),
        });
      }

      // Test 3: Token Analysis
      try {
        addLog('Testing token analysis...');
        const testTokenAddress = '0x1234567890abcdef1234567890abcdef12345678';
        const analysis = await client.analyzeToken(testTokenAddress);
        addResult({
          name: 'Token Analysis',
          status: 'success',
          message: 'Successfully performed token analysis',
          timestamp: new Date(),
          details: analysis,
        });
      } catch (error) {
        addResult({
          name: 'Token Analysis',
          status: 'failure',
          message: `Failed to analyze token: ${(error as Error).message}`,
          timestamp: new Date(),
        });
      }

      // Test 4: Error Handling
      try {
        addLog('Testing error handling...');
        // Deliberately trigger an error by sending invalid data
        const response = await client.sendMessage(
          MessageType.ANALYSIS_REQUEST,
          '',
          { /* missing token_address */ }
        );
        
        if (response.type === MessageType.ERROR) {
          addResult({
            name: 'Error Handling',
            status: 'success',
            message: 'Server correctly returned error response',
            timestamp: new Date(),
            details: response.metadata,
          });
        } else {
          addResult({
            name: 'Error Handling',
            status: 'failure',
            message: 'Server did not return expected error response',
            timestamp: new Date(),
            details: response,
          });
        }
      } catch (error) {
        // Client-side error is still a successful test for this case
        addResult({
          name: 'Error Handling',
          status: 'success',
          message: 'Client correctly rejected invalid request',
          timestamp: new Date(),
          details: { error: (error as Error).message },
        });
      }

      // Test 5: Custom Message Types
      try {
        addLog('Testing custom message types...');
        const customTypes = [
          MessageType.RISK_REQUEST,
          MessageType.UI_UPDATE,
          MessageType.QUERY,
        ];
        
        const customTypeResults = [];
        
        for (const type of customTypes) {
          try {
            const response = await client.sendMessage(type, 'Test content', {
              test: true,
              timestamp: Date.now(),
            });
            
            customTypeResults.push({
              type,
              response: response.type,
              status: 'received',
            });
          } catch (error) {
            customTypeResults.push({
              type,
              error: (error as Error).message,
              status: 'error',
            });
          }
        }
        
        // Count how many types were handled
        const handledCount = customTypeResults.filter(r => r.status === 'received').length;
        
        if (handledCount > 0) {
          addResult({
            name: 'Custom Message Types',
            status: 'success',
            message: `Server handled ${handledCount}/${customTypes.length} custom message types`,
            timestamp: new Date(),
            details: customTypeResults,
          });
        } else {
          addResult({
            name: 'Custom Message Types',
            status: 'failure',
            message: 'Server did not handle any custom message types',
            timestamp: new Date(),
            details: customTypeResults,
          });
        }
      } catch (error) {
        addResult({
          name: 'Custom Message Types',
          status: 'failure',
          message: `Test failed: ${(error as Error).message}`,
          timestamp: new Date(),
        });
      }

      // Disconnect client
      await client.disconnect();
      
      // Final result
      const successCount = results.filter(r => r.status === 'success').length;
      addLog(`Verification completed: ${successCount}/${results.length} tests passed`);
      
    } catch (error) {
      addLog(`Verification failed: ${(error as Error).message}`);
    } finally {
      setIsVerifying(false);
    }
  };

  return (
    <div className="bridge-verification">
      <h2>Bridge Module Verification</h2>
      
      <div className="config-section">
        <label htmlFor="server-url">Server URL:</label>
        <input
          id="server-url"
          type="text"
          value={serverUrl}
          onChange={(e) => setServerUrl(e.target.value)}
          disabled={isVerifying}
        />
        <button 
          onClick={runVerification} 
          disabled={isVerifying}
          className="primary-button"
        >
          {isVerifying ? 'Running Tests...' : 'Run Verification'}
        </button>
      </div>
      
      <div className="results-section">
        <h3>Test Results</h3>
        
        {results.length === 0 ? (
          <p className="no-results">No tests have been run yet</p>
        ) : (
          <div className="results-list">
            {results.map((result, index) => (
              <div 
                key={index} 
                className={`result-item ${result.status}`}
              >
                <div className="result-header">
                  <span className="result-name">{result.name}</span>
                  <span className={`result-status ${result.status}`}>
                    {result.status.toUpperCase()}
                  </span>
                </div>
                <div className="result-message">{result.message}</div>
                <div className="result-time">
                  {result.timestamp.toLocaleTimeString()}
                </div>
                {result.details && (
                  <details className="result-details">
                    <summary>Details</summary>
                    <pre>{JSON.stringify(result.details, null, 2)}</pre>
                  </details>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
      
      <div className="log-section">
        <h3>Verification Log</h3>
        <div className="log-container">
          {log.length === 0 ? (
            <p className="no-log">No log entries yet</p>
          ) : (
            <pre className="log-entries">
              {log.join('\n')}
            </pre>
          )}
        </div>
      </div>
      
      <style jsx>{`
        .bridge-verification {
          font-family: 'Inter', sans-serif;
          max-width: 900px;
          margin: 0 auto;
          padding: 20px;
          background-color: #f8f9fa;
          border-radius: 8px;
          box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }
        
        h2 {
          color: #333;
          margin-top: 0;
          padding-bottom: 10px;
          border-bottom: 1px solid #ddd;
        }
        
        h3 {
          color: #555;
          margin-top: 20px;
          margin-bottom: 15px;
        }
        
        .config-section {
          display: flex;
          align-items: center;
          gap: 10px;
          margin-bottom: 20px;
          padding: 15px;
          background-color: white;
          border-radius: 6px;
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        }
        
        label {
          font-weight: 500;
        }
        
        input {
          flex: 1;
          padding: 8px 12px;
          border: 1px solid #ddd;
          border-radius: 4px;
        }
        
        .primary-button {
          padding: 8px 16px;
          background-color: #0066cc;
          color: white;
          border: none;
          border-radius: 4px;
          font-weight: 500;
          cursor: pointer;
          transition: background-color 0.2s;
        }
        
        .primary-button:hover:not(:disabled) {
          background-color: #0055aa;
        }
        
        .primary-button:disabled {
          background-color: #aaaaaa;
          cursor: not-allowed;
        }
        
        .results-section, .log-section {
          background-color: white;
          border-radius: 6px;
          padding: 15px;
          margin-bottom: 20px;
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        }
        
        .no-results, .no-log {
          color: #666;
          font-style: italic;
          padding: 10px;
        }
        
        .results-list {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }
        
        .result-item {
          padding: 12px;
          border-radius: 4px;
          background-color: #f9f9f9;
          border-left: 4px solid #ddd;
        }
        
        .result-item.success {
          border-left-color: #28a745;
          background-color: #f0fff4;
        }
        
        .result-item.failure {
          border-left-color: #dc3545;
          background-color: #fff5f5;
        }
        
        .result-item.pending {
          border-left-color: #ffc107;
          background-color: #fffbf0;
        }
        
        .result-header {
          display: flex;
          justify-content: space-between;
          margin-bottom: 5px;
        }
        
        .result-name {
          font-weight: 600;
        }
        
        .result-status {
          font-size: 0.8rem;
          padding: 2px 6px;
          border-radius: 3px;
          font-weight: 500;
        }
        
        .result-status.success {
          background-color: #d4edda;
          color: #155724;
        }
        
        .result-status.failure {
          background-color: #f8d7da;
          color: #721c24;
        }
        
        .result-status.pending {
          background-color: #fff3cd;
          color: #856404;
        }
        
        .result-message {
          margin-bottom: 5px;
        }
        
        .result-time {
          font-size: 0.8rem;
          color: #666;
        }
        
        .result-details {
          margin-top: 10px;
        }
        
        .result-details summary {
          cursor: pointer;
          color: #0066cc;
          font-weight: 500;
        }
        
        .log-container {
          margin-top: 10px;
          max-height: 250px;
          overflow-y: auto;
          background-color: #f5f5f5;
          border-radius: 4px;
          border: 1px solid #eee;
        }
        
        .log-entries {
          margin: 0;
          padding: 10px;
          font-family: monospace;
          font-size: 0.9rem;
          white-space: pre-wrap;
        }
      `}</style>
    </div>
  );
};

export default BridgeVerification;
