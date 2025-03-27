/**
 * BridgeSetup.tsx
 * 
 * @module components/Onboarding
 * @description Component for setting up bridge connections during onboarding.
 * @version 1.2.0
 */

import React, { useState, useEffect } from 'react';
import { Form, Input, Select, Button, Alert, Card, Tooltip, Badge } from '../ui';
import { InfoCircleOutlined, LockOutlined, ApiOutlined } from '@ant-design/icons';
import axios from 'axios';

interface BridgeSetupProps {
  serviceName: string;
  serviceHost: string;
  servicePort: string;
  serviceProtocol: string;
  onServiceNameChange: (value: string) => void;
  onServiceHostChange: (value: string) => void;
  onServicePortChange: (value: string) => void;
  onServiceProtocolChange: (value: string) => void;
}

export const BridgeSetup: React.FC<BridgeSetupProps> = ({
  serviceName,
  serviceHost,
  servicePort,
  serviceProtocol,
  onServiceNameChange,
  onServiceHostChange,
  onServicePortChange,
  onServiceProtocolChange
}) => {
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string; metrics?: any } | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [securityLevel, setSecurityLevel] = useState('inherit');
  const [healthCheckPath, setHealthCheckPath] = useState('/health');
  const [timeout, setTimeout] = useState('30');
  const [maxConnections, setMaxConnections] = useState('100');
  const [isValidName, setIsValidName] = useState(true);
  
  // Protocol options for the dropdown - aligned with supported backend protocols
  const protocolOptions = [
    { value: 'http', label: 'HTTP/REST' },
    { value: 'https', label: 'HTTPS/REST (Secure)' },
    { value: 'grpc', label: 'gRPC' },
    { value: 'ws', label: 'WebSocket' },
    { value: 'wss', label: 'WebSocket (Secure)' },
    { value: 'graphql', label: 'GraphQL' }
  ];
  
  // Validate service name
  useEffect(() => {
    const nameRegex = /^[a-z0-9][-a-z0-9]*[a-z0-9]$/;
    setIsValidName(serviceName === '' || nameRegex.test(serviceName));
  }, [serviceName]);
  
  // Test the connection to the service
  const testConnection = async () => {
    setTesting(true);
    setTestResult(null);
    
    try {
      // Call the Go backend API to test the connection
      const response = await axios.post('/api/v1/bridge/connections/test', {
        connection: {
          name: serviceName,
          host: serviceHost,
          port: parseInt(servicePort, 10),
          protocol: serviceProtocol,
          settings: {
            healthCheckPath,
            timeout: parseInt(timeout, 10),
            maxConnections: parseInt(maxConnections, 10),
            securityLevel
          }
        }
      });
      
      setTestResult({
        success: response.data.status === 'success',
        message: response.data.message || 'Connection successful!',
        metrics: response.data.metrics || null
      });
    } catch (error: any) {
      console.error('Error testing connection:', error);
      setTestResult({
        success: false,
        message: error.response?.data?.error || 
                'Failed to test connection. Please check your settings and try again.'
      });
    } finally {
      setTesting(false);
    }
  };
  
  return (
    <div className="bridge-setup">
      <Form layout="vertical">
        <Form.Item 
          label="Service Name" 
          required 
          tooltip="A unique name to identify this service (lowercase letters, numbers, and hyphens only)"
          validateStatus={isValidName ? undefined : 'error'}
          help={isValidName ? undefined : 'Service name must consist of lowercase letters, numbers, and hyphens only'}
        >
          <Input
            value={serviceName}
            onChange={(e) => onServiceNameChange(e.target.value)}
            placeholder="e.g., my-api"
            suffix={
              <Tooltip title="Name must be unique and will be used in the bridge configuration">
                <InfoCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} />
              </Tooltip>
            }
          />
        </Form.Item>
        
        <Form.Item 
          label="Service Protocol" 
          required
          tooltip="The communication protocol used by your service"
        >
          <Select
            value={serviceProtocol}
            onChange={onServiceProtocolChange}
            options={protocolOptions}
          />
        </Form.Item>
        
        <Form.Item 
          label="Host" 
          required
          tooltip="The hostname or IP address of your service"
        >
          <Input
            value={serviceHost}
            onChange={(e) => onServiceHostChange(e.target.value)}
            placeholder="e.g., localhost or 192.168.1.100"
            suffix={
              <Tooltip title="For development, use localhost. For production, use a resolvable hostname or IP.">
                <ApiOutlined style={{ color: 'rgba(0,0,0,.45)' }} />
              </Tooltip>
            }
          />
        </Form.Item>
        
        <Form.Item 
          label="Port" 
          required
          tooltip="The port your service is listening on"
        >
          <Input
            value={servicePort}
            onChange={(e) => onServicePortChange(e.target.value)}
            placeholder="e.g., 8080"
            type="number"
            min="1"
            max="65535"
          />
        </Form.Item>
        
        <Button 
          type="link" 
          onClick={() => setShowAdvanced(!showAdvanced)}
        >
          {showAdvanced ? 'Hide Advanced Options' : 'Show Advanced Options'}
        </Button>
        
        {showAdvanced && (
          <Card className="advanced-options">
            <Form.Item 
              label="Health Check Path" 
              tooltip="A URL path that returns a 200 OK status when the service is healthy"
            >
              <Input
                placeholder="e.g., /health"
                value={healthCheckPath}
                onChange={(e) => setHealthCheckPath(e.target.value)}
              />
            </Form.Item>
            
            <Form.Item 
              label="Connection Timeout (seconds)" 
              tooltip="Maximum time to wait for a connection"
            >
              <Input
                type="number"
                placeholder="30"
                value={timeout}
                onChange={(e) => setTimeout(e.target.value)}
                min="1"
                max="300"
              />
            </Form.Item>
            
            <Form.Item 
              label="Max Connections" 
              tooltip="Maximum number of concurrent connections"
            >
              <Input
                type="number"
                placeholder="100"
                value={maxConnections}
                onChange={(e) => setMaxConnections(e.target.value)}
                min="1"
                max="10000"
              />
            </Form.Item>
            
            <Form.Item 
              label="Security Level" 
              tooltip="Security level for this connection"
            >
              <Select
                value={securityLevel}
                onChange={(value) => setSecurityLevel(value)}
                options={[
                  { value: 'inherit', label: 'Inherit from Global Settings' },
                  { value: 'high', label: 'High Security (TLS 1.3, IP Masking)' },
                  { value: 'medium', label: 'Medium Security (TLS 1.2)' },
                  { value: 'low', label: 'Low Security (Development Only)' }
                ]}
                suffixIcon={<LockOutlined />}
              />
            </Form.Item>
          </Card>
        )}
        
        <div className="test-section">
          <Button 
            onClick={testConnection}
            loading={testing}
            disabled={!serviceName || !serviceHost || !servicePort || !isValidName}
            type="primary"
          >
            Test Connection
          </Button>
          
          {testResult && (
            <Alert
              message={testResult.success ? "Success" : "Connection Failed"}
              description={
                <div>
                  <p>{testResult.message}</p>
                  {testResult.metrics && (
                    <div className="metrics-preview">
                      <h4>Connection Metrics</h4>
                      <ul>
                        <li>Latency: {testResult.metrics.latency}ms</li>
                        <li>Security Level: <Badge status={testResult.metrics.security === 'high' ? 'success' : 'warning'} text={testResult.metrics.security} /></li>
                      </ul>
                    </div>
                  )}
                </div>
              }
              type={testResult.success ? "success" : "error"}
              showIcon
              style={{ marginTop: 16 }}
            />
          )}
        </div>
      </Form>
      
      <div className="connection-preview">
        <h4>Connection Preview</h4>
        <p>Your service will be accessible through the QUANT Bridge at:</p>
        <code>
          {`${window.location.protocol}//${window.location.host}/bridge/${serviceName}`}
        </code>
        <p className="note">
          Note: The actual endpoint URL may differ based on your deployment configuration and security settings.
        </p>
      </div>
    </div>
  );
};
