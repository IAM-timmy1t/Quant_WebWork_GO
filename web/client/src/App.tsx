import React, { useState, lazy, Suspense } from 'react';
import { BridgeMessage } from './bridge/BridgeClient';

// Lazy load components
const BridgeConnection = lazy(() => import('./components/BridgeConnection'));
const BridgeVerification = lazy(() => import('./components/BridgeVerification'));

// Loading component
const LoadingComponent = () => (
  <div className="loading-container">
    <div className="loading-spinner"></div>
    <p>Loading component...</p>
    <style jsx>{`
      .loading-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 300px;
        background-color: #f9f9f9;
        border-radius: 8px;
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
      }
      
      .loading-spinner {
        width: 40px;
        height: 40px;
        border: 3px solid #f3f3f3;
        border-top: 3px solid #0066cc;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 16px;
      }
      
      @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
      }
    `}</style>
  </div>
);

const App: React.FC = () => {
  const [messages, setMessages] = useState<BridgeMessage[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [activeTab, setActiveTab] = useState('bridge');

  // Server URL - This should be configured based on environment
  const serverUrl = process.env.REACT_APP_SERVER_URL || 'http://localhost:8080';

  // Handle connection events
  const handleConnect = () => {
    setIsConnected(true);
    console.log('Connected to bridge server');
  };

  const handleDisconnect = () => {
    setIsConnected(false);
    console.log('Disconnected from bridge server');
  };

  const handleError = (error: Error) => {
    console.error('Bridge error:', error);
  };

  const handleMessage = (message: BridgeMessage) => {
    setMessages(prev => [message, ...prev].slice(0, 100)); // Keep last 100 messages
    console.log('Received message:', message);
  };

  return (
    <div className="app-container">
      <header className="app-header">
        <h1>Quant WebWorks GO</h1>
        <div className="connection-indicator">
          <span className={`status-dot ${isConnected ? 'connected' : 'disconnected'}`}></span>
          <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
        </div>
      </header>

      <nav className="app-nav">
        <button 
          className={activeTab === 'bridge' ? 'active' : ''}
          onClick={() => setActiveTab('bridge')}
        >
          Bridge Connection
        </button>
        <button 
          className={activeTab === 'verification' ? 'active' : ''}
          onClick={() => setActiveTab('verification')}
        >
          Bridge Verification
        </button>
        <button 
          className={activeTab === 'messages' ? 'active' : ''}
          onClick={() => setActiveTab('messages')}
        >
          Messages
        </button>
        <button 
          className={activeTab === 'settings' ? 'active' : ''}
          onClick={() => setActiveTab('settings')}
        >
          Settings
        </button>
      </nav>

      <main className="app-content">
        <Suspense fallback={<LoadingComponent />}>
          {activeTab === 'bridge' && (
            <BridgeConnection
              serverUrl={serverUrl}
              onConnect={handleConnect}
              onDisconnect={handleDisconnect}
              onError={handleError}
              onMessage={handleMessage}
            />
          )}
          
          {activeTab === 'verification' && (
            <BridgeVerification />
          )}

          {activeTab === 'messages' && (
            <div className="messages-panel">
              <h2>Message History</h2>
              {messages.length === 0 ? (
                <p className="no-messages">No messages received yet</p>
              ) : (
                <div className="message-list">
                  {messages.map((msg, index) => (
                    <div key={index} className="message-item">
                      <div className="message-header">
                        <span className="message-id">{msg.id}</span>
                        <span className="message-type">{msg.type}</span>
                        <span className="message-time">
                          {new Date(msg.timestamp).toLocaleTimeString()}
                        </span>
                      </div>
                      <div className="message-content">
                        {msg.content || <em>No content</em>}
                      </div>
                      <details>
                        <summary>Metadata</summary>
                        <pre>{JSON.stringify(msg.metadata, null, 2)}</pre>
                      </details>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === 'settings' && (
            <div className="settings-panel">
              <h2>Application Settings</h2>
              
              <div className="settings-section">
                <h3>Bridge Connection</h3>
                <div className="setting-item">
                  <label htmlFor="server-url">Server URL:</label>
                  <input 
                    id="server-url" 
                    type="text" 
                    value={serverUrl} 
                    disabled 
                    readOnly 
                  />
                  <small>
                    Configure this in your environment settings (.env file) as REACT_APP_SERVER_URL
                  </small>
                </div>
              </div>
              
              <div className="settings-section">
                <h3>Interface Settings</h3>
                <div className="setting-item checkbox">
                  <input 
                    id="dark-mode" 
                    type="checkbox" 
                  />
                  <label htmlFor="dark-mode">Enable Dark Mode</label>
                </div>
                <div className="setting-item checkbox">
                  <input 
                    id="auto-connect" 
                    type="checkbox" 
                    defaultChecked 
                  />
                  <label htmlFor="auto-connect">Auto-connect on startup</label>
                </div>
              </div>
              
              <div className="settings-section">
                <h3>About</h3>
                <div className="about-info">
                  <p><strong>Version:</strong> 1.0.0</p>
                  <p><strong>Build Date:</strong> {new Date().toLocaleDateString()}</p>
                  <p>
                    <strong>Environment:</strong> {process.env.NODE_ENV || 'development'}
                  </p>
                </div>
              </div>
            </div>
          )}
        </Suspense>
      </main>

      <footer className="app-footer">
        <p> 2024 Quant WebWorks GO</p>
      </footer>

      <style jsx>{`
        .app-container {
          display: flex;
          flex-direction: column;
          min-height: 100vh;
          font-family: 'Inter', system-ui, -apple-system, sans-serif;
        }
        
        .app-header {
          background-color: #1a1a2e;
          color: white;
          padding: 1rem 2rem;
          display: flex;
          justify-content: space-between;
          align-items: center;
        }
        
        .app-header h1 {
          margin: 0;
          font-size: 1.5rem;
        }
        
        .connection-indicator {
          display: flex;
          align-items: center;
        }
        
        .status-dot {
          width: 10px;
          height: 10px;
          border-radius: 50%;
          display: inline-block;
          margin-right: 8px;
        }
        
        .connected {
          background-color: #4caf50;
        }
        
        .disconnected {
          background-color: #f44336;
        }
        
        .app-nav {
          background-color: #f5f5f5;
          padding: 0.5rem 2rem;
          display: flex;
          border-bottom: 1px solid #ddd;
        }
        
        .app-nav button {
          background: none;
          border: none;
          padding: 0.75rem 1rem;
          margin-right: 0.5rem;
          cursor: pointer;
          font-weight: 500;
          color: #555;
          border-radius: 4px;
          transition: all 0.2s;
        }
        
        .app-nav button:hover {
          background-color: #e9e9e9;
        }
        
        .app-nav button.active {
          background-color: #0066cc;
          color: white;
        }
        
        .app-content {
          flex: 1;
          padding: 2rem;
          background-color: #f9f9f9;
        }
        
        .messages-panel, .settings-panel {
          background-color: white;
          border-radius: 8px;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
          padding: 1.5rem;
          max-width: 800px;
          margin: 0 auto;
        }
        
        .messages-panel h2, .settings-panel h2 {
          margin-top: 0;
          color: #333;
          border-bottom: 1px solid #eee;
          padding-bottom: 0.75rem;
        }
        
        .no-messages {
          color: #666;
          font-style: italic;
        }
        
        .message-list {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }
        
        .message-item {
          border: 1px solid #eee;
          border-radius: 6px;
          padding: 1rem;
          background-color: #f9f9f9;
        }
        
        .message-header {
          display: flex;
          justify-content: space-between;
          margin-bottom: 0.5rem;
          font-size: 0.875rem;
        }
        
        .message-id {
          color: #0066cc;
          font-family: monospace;
        }
        
        .message-type {
          background-color: #e6f7ff;
          color: #0066cc;
          padding: 2px 6px;
          border-radius: 4px;
          font-size: 0.75rem;
        }
        
        .message-time {
          color: #777;
        }
        
        .message-content {
          margin-bottom: 0.75rem;
          padding: 0.5rem;
          background-color: white;
          border-radius: 4px;
          border: 1px solid #eee;
        }
        
        details {
          margin-top: 0.5rem;
        }
        
        summary {
          cursor: pointer;
          color: #555;
          font-size: 0.875rem;
        }
        
        pre {
          background-color: #f5f5f5;
          padding: 0.75rem;
          border-radius: 4px;
          font-size: 0.875rem;
          overflow: auto;
          max-height: 300px;
        }
        
        .settings-section {
          margin-bottom: 2rem;
        }
        
        .settings-section h3 {
          font-size: 1.1rem;
          color: #333;
          margin-bottom: 1rem;
        }
        
        .setting-item {
          margin-bottom: 1rem;
        }
        
        .setting-item label {
          display: block;
          margin-bottom: 0.5rem;
          font-weight: 500;
        }
        
        .setting-item input[type="text"] {
          width: 100%;
          padding: 0.5rem;
          border: 1px solid #ddd;
          border-radius: 4px;
          font-size: 0.875rem;
        }
        
        .setting-item small {
          display: block;
          margin-top: 0.25rem;
          color: #777;
          font-size: 0.75rem;
        }
        
        .setting-item.checkbox {
          display: flex;
          align-items: center;
        }
        
        .setting-item.checkbox input {
          margin-right: 0.5rem;
        }
        
        .setting-item.checkbox label {
          margin-bottom: 0;
        }
        
        .about-info {
          background-color: #f5f5f5;
          padding: 1rem;
          border-radius: 4px;
        }
        
        .about-info p {
          margin: 0.5rem 0;
        }
        
        .app-footer {
          background-color: #f5f5f5;
          padding: 1rem 2rem;
          border-top: 1px solid #ddd;
          text-align: center;
          color: #666;
          font-size: 0.875rem;
        }
      `}</style>
    </div>
  );
};

export default App;
