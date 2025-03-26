/**
 * BridgeSetup.tsx
 * 
 * @module components/Onboarding
 * @description Component for setting up bridge connections during onboarding.
 * @version 1.0.0
 */

import React, { useState } from 'react';
import { Form, Input, Select, Button, Alert, Card, Tooltip } from '../ui';
import { InfoCircleOutlined } from '@ant-design/icons';
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
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  
  // Protocol options for the dropdown
  const protocolOptions = [
    { value: 'http', label: 'HTTP' },
    { value: 'https', label: 'HTTPS' },
    { value: 'tcp', label: 'TCP' },
    { value: 'udp', label: 'UDP' },
    { value: 'ws', label: 'WebSocket' },
    { value: 'grpc', label: 'gRPC' }
  ];
  
  // Test the connection to the service
  const testConnection = async () => {
    setTesting(true);
    setTestResult(null);
    
    try {
      // In a real implementation, this would call your API to test the connection
      const response = await axios.post('/api/v1/bridge/test-connection', {
        name: serviceName,
        host: serviceHost,
        port: parseInt(servicePort, 10),
        protocol: serviceProtocol
      });
      
      setTestResult({
        success: response.data.success,
        message: response.data.message || 'Connection successful!'
      });
    } catch (error) {
      console.error('Error testing connection:', error);
      setTestResult({
        success: false,
        message: 'Failed to test connection. Please check your settings and try again.'
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
          tooltip="A unique name to identify this service"
        >
          <Input
            value={serviceName}
            onChange={(e) => onServiceNameChange(e.target.value)}
            placeholder="e.g., my-api"
            suffix={
              <Tooltip title="Name must be unique and will be used in the URL path">
                <InfoCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} />
              </Tooltip>
            }
          />
        </Form.Item>
        
        <Form.Item 
          label="Service Protocol" 
          required
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
                defaultValue="/health"
              />
            </Form.Item>
            
            <Form.Item 
              label="Connection Timeout (seconds)" 
              tooltip="Maximum time to wait for a connection"
            >
              <Input
                type="number"
                placeholder="30"
                defaultValue="30"
              />
            </Form.Item>
            
            <Form.Item 
              label="Max Connections" 
              tooltip="Maximum number of concurrent connections"
            >
              <Input
                type="number"
                placeholder="100"
                defaultValue="100"
              />
            </Form.Item>
            
            <Form.Item 
              label="Security Level" 
              tooltip="Security level for this connection"
            >
              <Select
                defaultValue="inherit"
                options={[
                  { value: 'inherit', label: 'Inherit from Global Settings' },
                  { value: 'high', label: 'High Security' },
                  { value: 'medium', label: 'Medium Security' },
                  { value: 'low', label: 'Low Security (Development Only)' }
                ]}
              />
            </Form.Item>
          </Card>
        )}
        
        <div className="test-section">
          <Button 
            onClick={testConnection}
            loading={testing}
            disabled={!serviceName || !serviceHost || !servicePort}
          >
            Test Connection
          </Button>
          
          {testResult && (
            <Alert
              message={testResult.success ? "Success" : "Connection Failed"}
              description={testResult.message}
              type={testResult.success ? "success" : "error"}
              showIcon
              style={{ marginTop: 16 }}
            />
          )}
        </div>
      </Form>
      
      <div className="connection-preview">
        <h4>Connection Preview</h4>
        <p>Your service will be accessible at:</p>
        <code>
          {`${window.location.protocol}//${window.location.host}/bridge/${serviceName}`}
        </code>
        <p className="note">
          Note: The actual URL may differ based on your deployment configuration.
        </p>
      </div>
    </div>
  );
};
