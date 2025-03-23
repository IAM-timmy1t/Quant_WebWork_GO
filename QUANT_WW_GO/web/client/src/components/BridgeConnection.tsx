import React, { useEffect, useState, useCallback } from 'react';
import { 
  BridgeClient, 
  createBridgeClient,
  MessageType, 
  BridgeMessage 
} from '../bridge/BridgeClient';
import metricsCollector from '../monitoring/MetricsCollector';

interface BridgeConnectionProps {
  serverUrl: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Error) => void;
  onMessage?: (message: BridgeMessage) => void;
}

/**
 * BridgeConnection component for managing connection to the Bridge Module
 */
const BridgeConnection: React.FC<BridgeConnectionProps> = ({
  serverUrl,
  onConnect,
  onDisconnect,
  onError,
  onMessage,
}) => {
  // Client state
  const [client, setClient] = useState<BridgeClient | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [metrics, setMetrics] = useState<any>(null);
  const [tokenAddress, setTokenAddress] = useState('');
  const [analysisResult, setAnalysisResult] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [renderStartTime, setRenderStartTime] = useState<number>(0);

  // Initialize metrics collector
  useEffect(() => {
    metricsCollector.initialize();
    
    // Record initial render time for performance tracking
    const startTime = performance.now();
    setRenderStartTime(startTime);
    
    return () => {
      // Record component unmount metrics
      const unmountTime = performance.now();
      const totalMountTime = unmountTime - renderStartTime;
      if (renderStartTime > 0) {
        metricsCollector.recordRenderTime('BridgeConnection_mount', totalMountTime);
      }
    };
  }, []);

  // Measure and record render time after each render
  useEffect(() => {
    const endTime = performance.now();
    if (renderStartTime > 0) {
      const renderTime = endTime - renderStartTime;
      metricsCollector.recordRenderTime('BridgeConnection', renderTime);
    }
    // Set start time for next render
    setRenderStartTime(performance.now());
  });

  // Initialize client
  useEffect(() => {
    // Create bridge client
    const bridgeClient = createBridgeClient({
      serverUrl,
      onConnect: () => {
        setConnected(true);
        setError(null);
        if (onConnect) {
          onConnect();
        }
      },
      onDisconnect: () => {
        setConnected(false);
        if (onDisconnect) {
          onDisconnect();
        }
      },
      onError: (err) => {
        setError(err);
        if (onError) {
          onError(err);
        }
      },
    });

    // Register message handler
    bridgeClient.on('message', (message: BridgeMessage) => {
      if (onMessage) {
        onMessage(message);
      }
    });

    // Set client
    setClient(bridgeClient);

    // Connect on mount and disconnect on unmount
    bridgeClient.connect().catch(setError);
    return () => {
      bridgeClient.disconnect().catch(console.error);
    };
  }, [serverUrl, onConnect, onDisconnect, onError, onMessage]);

  // Fetch metrics
  const fetchMetrics = useCallback(async () => {
    if (!client || !connected) {
      return;
    }

    try {
      setLoading(true);
      const result = await client.getMetrics();
      setMetrics(result);
      setError(null);
      metricsCollector.recordUserInteraction('get_metrics');
    } catch (err) {
      setError(err as Error);
      metricsCollector.recordError('metrics', err.message);
    } finally {
      setLoading(false);
    }
  }, [client, connected]);

  // Analyze token
  const analyzeToken = useCallback(async () => {
    if (!client || !connected || !tokenAddress) {
      return;
    }

    try {
      setLoading(true);
      const result = await client.analyzeToken(tokenAddress);
      setAnalysisResult(result);
      setError(null);
      metricsCollector.recordUserInteraction('analyze_token');
    } catch (err) {
      setError(err as Error);
      setAnalysisResult(null);
      metricsCollector.recordError('token_analysis', err.message);
    } finally {
      setLoading(false);
    }
  }, [client, connected, tokenAddress]);

  // Fetch metrics on connection
  useEffect(() => {
    if (connected) {
      fetchMetrics();
    }
  }, [connected, fetchMetrics]);

  // Render component
  return (
    <div className="bridge-connection">
      <h2>Bridge Connection</h2>
      
      {/* Connection Status */}
      <div className="connection-status">
        <span className={`status-indicator ${connected ? 'connected' : 'disconnected'}`}></span>
        <span>{connected ? 'Connected' : 'Disconnected'}</span>
        {error && <div className="error-message">{error.message}</div>}
      </div>

      {/* Connection Actions */}
      <div className="connection-actions">
        <button 
          onClick={() => client?.connect()} 
          disabled={connected || !client}
        >
          Connect
        </button>
        <button 
          onClick={() => client?.disconnect()} 
          disabled={!connected || !client}
        >
          Disconnect
        </button>
        <button 
          onClick={fetchMetrics} 
          disabled={!connected || !client || loading}
        >
          Refresh Metrics
        </button>
      </div>

      {/* Token Analysis */}
      <div className="token-analysis">
        <h3>Token Analysis</h3>
        <div className="input-group">
          <label htmlFor="token-address">Token Address:</label>
          <input
            id="token-address"
            type="text"
            value={tokenAddress}
            onChange={(e) => setTokenAddress(e.target.value)}
            placeholder="Enter token address"
            disabled={!connected || loading}
          />
          <button
            onClick={analyzeToken}
            disabled={!connected || !tokenAddress || loading}
          >
            Analyze
          </button>
        </div>

        {loading && <div className="loading">Loading...</div>}
        
        {/* Analysis Result */}
        {analysisResult && (
          <div className="analysis-result">
            <h4>Analysis Result</h4>
            <pre>{JSON.stringify(analysisResult, null, 2)}</pre>
          </div>
        )}
      </div>

      {/* Metrics Display */}
      {metrics && (
        <div className="metrics">
          <h3>Bridge Metrics</h3>
          <table>
            <tbody>
              <tr>
                <td>Messages Processed</td>
                <td>{metrics.MessagesProcessed || 0}</td>
              </tr>
              <tr>
                <td>Tokens Processed</td>
                <td>{metrics.TokensProcessed || 0}</td>
              </tr>
              <tr>
                <td>Chunked Messages</td>
                <td>{metrics.ChunkedMessages || 0}</td>
              </tr>
              <tr>
                <td>Average Risk Score</td>
                <td>{metrics.AverageRiskScore?.toFixed(2) || 0}</td>
              </tr>
              <tr>
                <td>Last Updated</td>
                <td>{metrics.LastUpdated ? new Date(metrics.LastUpdated).toLocaleString() : 'N/A'}</td>
              </tr>
            </tbody>
          </table>
          
          {/* Additional Metrics */}
          <details>
            <summary>Additional Metrics</summary>
            <pre>{JSON.stringify(metrics, null, 2)}</pre>
          </details>
        </div>
      )}

      <style jsx>{`
        .bridge-connection {
          font-family: 'Inter', sans-serif;
          padding: 20px;
          border-radius: 8px;
          background-color: #f8f9fa;
          box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
          max-width: 800px;
          margin: 0 auto;
        }
        
        h2 {
          color: #333;
          margin-top: 0;
          border-bottom: 1px solid #ddd;
          padding-bottom: 10px;
        }
        
        .connection-status {
          display: flex;
          align-items: center;
          margin-bottom: 20px;
        }
        
        .status-indicator {
          width: 12px;
          height: 12px;
          border-radius: 50%;
          margin-right: 8px;
        }
        
        .connected {
          background-color: #28a745;
        }
        
        .disconnected {
          background-color: #dc3545;
        }
        
        .connection-actions {
          display: flex;
          gap: 10px;
          margin-bottom: 20px;
        }
        
        button {
          padding: 8px 16px;
          border: none;
          border-radius: 4px;
          background-color: #0066cc;
          color: white;
          cursor: pointer;
          font-weight: 500;
          transition: background-color 0.2s;
        }
        
        button:hover:not(:disabled) {
          background-color: #0055aa;
        }
        
        button:disabled {
          background-color: #cccccc;
          cursor: not-allowed;
        }
        
        .error-message {
          margin-left: 10px;
          color: #dc3545;
          font-size: 14px;
        }
        
        .token-analysis {
          margin-top: 30px;
          padding: 15px;
          background-color: white;
          border-radius: 6px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
        }
        
        .input-group {
          display: flex;
          align-items: center;
          margin-bottom: 15px;
        }
        
        label {
          margin-right: 10px;
          font-weight: 500;
        }
        
        input {
          flex: 1;
          padding: 8px 12px;
          border: 1px solid #ddd;
          border-radius: 4px;
          margin-right: 10px;
        }
        
        .loading {
          margin: 10px 0;
          color: #666;
          font-style: italic;
        }
        
        .analysis-result, .metrics {
          margin-top: 20px;
          background-color: white;
          border-radius: 6px;
          padding: 15px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
        }
        
        pre {
          background-color: #f5f5f5;
          padding: 10px;
          border-radius: 4px;
          overflow: auto;
          font-size: 14px;
        }
        
        table {
          width: 100%;
          border-collapse: collapse;
          margin-bottom: 15px;
        }
        
        td {
          padding: 8px;
          border-bottom: 1px solid #eee;
        }
        
        td:first-child {
          font-weight: 500;
        }
        
        details {
          margin-top: 15px;
        }
        
        summary {
          cursor: pointer;
          color: #0066cc;
          font-weight: 500;
          margin-bottom: 10px;
        }
      `}</style>
    </div>
  );
};

export default BridgeConnection;
