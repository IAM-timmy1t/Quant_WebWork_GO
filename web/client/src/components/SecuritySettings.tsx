/**
 * SecuritySettings.tsx
 * 
 * @module components
 * @description Security configuration interface for managing security settings
 * @version 1.0.0
 */

import React, { useState, useEffect } from 'react';
import { Card, Form, Switch, Input, Select, Button, Alert, Tabs, Table, Tag } from './ui';
import { useConfig } from '../hooks/useConfig';
import { useSecurityAudit } from '../hooks/useSecurityAudit';

// Component to display and edit security settings
export const SecuritySettings: React.FC = () => {
  const { config, updateConfig, loading, error } = useConfig();
  const { auditLogs, loading: auditLoading } = useSecurityAudit();
  const [securityScore, setSecurityScore] = useState(0);
  const [recommendations, setRecommendations] = useState<string[]>([]);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [saveError, setSaveError] = useState<Error | null>(null);
  
  // Calculate security score based on current configuration
  useEffect(() => {
    if (!config || loading) return;
    
    let score = 0;
    const recs: string[] = [];
    
    // Authentication checks
    if (config.security?.authRequired) {
      score += 20;
    } else {
      recs.push("Enable authentication for improved security");
    }
    
    // TLS checks
    if (config.security?.tlsEnabled) {
      score += 20;
    } else {
      recs.push("Enable TLS encryption for secure communications");
    }
    
    // IP Masking checks
    if (config.security?.ipMasking?.enabled) {
      score += 15;
      
      // Additional points for rotation
      if (config.security?.ipMasking?.rotationEnabled) {
        score += 5;
      } else {
        recs.push("Enable IP address rotation for enhanced privacy");
      }
    } else {
      recs.push("Enable IP masking to protect user privacy");
    }
    
    // Firewall checks
    if (config.security?.firewall?.enabled) {
      score += 15;
    } else {
      recs.push("Enable firewall protection");
    }
    
    // Rate limiting checks
    if (config.security?.rateLimiting?.enabled) {
      score += 10;
      
      // Check if using advanced rate limiter
      if (config.security?.rateLimiting?.advanced) {
        score += 5;
      } else {
        recs.push("Enable advanced rate limiting for high-traffic scenarios");
      }
    } else {
      recs.push("Enable rate limiting to prevent abuse");
    }
    
    // Audit logging checks
    if (config.security?.auditLogging?.enabled) {
      score += 10;
    } else {
      recs.push("Enable audit logging to track security events");
    }
    
    setSecurityScore(score);
    setRecommendations(recs);
  }, [config, loading]);
  
  // Handle saving security configuration
  const handleSave = async (values: any) => {
    try {
      await updateConfig({
        security: values
      });
      setSaveSuccess(true);
      setSaveError(null);
      
      // Reset success message after 3 seconds
      setTimeout(() => {
        setSaveSuccess(false);
      }, 3000);
    } catch (err) {
      setSaveError(err as Error);
      setSaveSuccess(false);
    }
  };
  
  if (loading) {
    return <div>Loading security settings...</div>;
  }
  
  if (error) {
    return (
      <Alert
        type="error"
        message="Error loading security settings"
        description={error.toString()}
      />
    );
  }
  
  // Get security level based on score
  const getSecurityLevel = (score: number) => {
    if (score >= 90) return { level: 'Excellent', color: '#52c41a' };
    if (score >= 70) return { level: 'Good', color: '#1890ff' };
    if (score >= 50) return { level: 'Moderate', color: '#faad14' };
    return { level: 'Weak', color: '#f5222d' };
  };
  
  const securityLevel = getSecurityLevel(securityScore);
  
  // Setup audit log columns
  const auditColumns = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'timestamp',
      render: (text: string) => new Date(text).toLocaleString()
    },
    {
      title: 'User',
      dataIndex: 'userID',
      key: 'userID',
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
    },
    {
      title: 'Category',
      dataIndex: 'category',
      key: 'category',
      render: (text: string) => (
        <Tag color={
          text === 'AUTHENTICATION' ? 'blue' :
          text === 'AUTHORIZATION' ? 'purple' :
          text === 'CONFIGURATION' ? 'orange' :
          text === 'SECURITY' ? 'red' : 'default'
        }>
          {text}
        </Tag>
      )
    },
    {
      title: 'Result',
      dataIndex: 'result',
      key: 'result',
      render: (text: string) => (
        <Tag color={text === 'success' ? 'green' : 'red'}>
          {text}
        </Tag>
      )
    },
    {
      title: 'IP Address',
      dataIndex: 'ipAddress',
      key: 'ipAddress',
    }
  ];
  
  return (
    <div className="security-settings">
      <div className="security-settings-header">
        <h1>Security Settings</h1>
        <div className="security-score">
          <div className="score-circle" style={{ borderColor: securityLevel.color }}>
            <span className="score-value">{securityScore}</span>
            <span className="score-max">/100</span>
          </div>
          <div className="score-details">
            <h3 style={{ color: securityLevel.color }}>{securityLevel.level} Security</h3>
            <p>Your system's security configuration score</p>
          </div>
        </div>
      </div>
      
      {/* Success/Error messages */}
      {saveSuccess && (
        <Alert
          message="Security settings saved successfully"
          type="success"
          showIcon
          closable
          style={{ marginBottom: '16px' }}
        />
      )}
      
      {saveError && (
        <Alert
          message="Failed to save security settings"
          description={saveError.message}
          type="error"
          showIcon
          closable
          style={{ marginBottom: '16px' }}
        />
      )}
      
      {/* Recommendations */}
      {recommendations.length > 0 && (
        <Card className="security-recommendations" style={{ marginBottom: '24px' }}>
          <h3>Security Recommendations</h3>
          <ul>
            {recommendations.map((rec, index) => (
              <li key={index}>{rec}</li>
            ))}
          </ul>
        </Card>
      )}
      
      <Tabs defaultActiveKey="general">
        <Tabs.TabPane tab="General Security" key="general">
          <Card>
            <Form
              layout="vertical"
              initialValues={config?.security || {}}
              onFinish={handleSave}
            >
              <h3>Authentication</h3>
              <Form.Item
                name={['authRequired']}
                label="Require Authentication"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['sessionTimeout']}
                label="Session Timeout (minutes)"
              >
                <Input type="number" min={5} max={1440} />
              </Form.Item>
              
              <h3>Encryption</h3>
              <Form.Item
                name={['tlsEnabled']}
                label="Enable TLS"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['tlsMinVersion']}
                label="Minimum TLS Version"
              >
                <Select>
                  <Select.Option value="1.0">TLS 1.0 (Not Recommended)</Select.Option>
                  <Select.Option value="1.1">TLS 1.1</Select.Option>
                  <Select.Option value="1.2">TLS 1.2</Select.Option>
                  <Select.Option value="1.3">TLS 1.3 (Recommended)</Select.Option>
                </Select>
              </Form.Item>
              
              <h3>Privacy</h3>
              <Form.Item
                name={['ipMasking', 'enabled']}
                label="Enable IP Masking"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['ipMasking', 'rotationEnabled']}
                label="Enable IP Address Rotation"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['ipMasking', 'rotationInterval']}
                label="Rotation Interval (minutes)"
              >
                <Input type="number" min={10} max={1440} />
              </Form.Item>
              
              <Form.Item
                name={['dnsPrivacy']}
                label="Enable DNS Privacy Protection"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <div className="form-actions">
                <Button type="primary" htmlType="submit">
                  Save Settings
                </Button>
              </div>
            </Form>
          </Card>
        </Tabs.TabPane>
        
        <Tabs.TabPane tab="Firewall & Rate Limiting" key="firewall">
          <Card>
            <Form
              layout="vertical"
              initialValues={config?.security || {}}
              onFinish={handleSave}
            >
              <h3>Firewall</h3>
              <Form.Item
                name={['firewall', 'enabled']}
                label="Enable Firewall"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['firewall', 'strictMode']}
                label="Strict Mode"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['firewall', 'allowedIPs']}
                label="Allowed IP Addresses (comma-separated)"
              >
                <Input placeholder="e.g. 192.168.1.1, 10.0.0.0/24" />
              </Form.Item>
              
              <h3>Rate Limiting</h3>
              <Form.Item
                name={['rateLimiting', 'enabled']}
                label="Enable Rate Limiting"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['rateLimiting', 'advanced']}
                label="Use Advanced Rate Limiter (for high traffic)"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['rateLimiting', 'requestsPerMinute']}
                label="Requests per Minute"
              >
                <Input type="number" min={10} max={10000} />
              </Form.Item>
              
              <Form.Item
                name={['rateLimiting', 'burstSize']}
                label="Burst Size"
              >
                <Input type="number" min={1} max={1000} />
              </Form.Item>
              
              <div className="form-actions">
                <Button type="primary" htmlType="submit">
                  Save Settings
                </Button>
              </div>
            </Form>
          </Card>
        </Tabs.TabPane>
        
        <Tabs.TabPane tab="Audit Logging" key="audit">
          <Card>
            <Form
              layout="vertical"
              initialValues={config?.security?.auditLogging || {}}
              onFinish={handleSave}
            >
              <h3>Audit Logging Configuration</h3>
              <Form.Item
                name={['auditLogging', 'enabled']}
                label="Enable Audit Logging"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name={['auditLogging', 'level']}
                label="Logging Level"
              >
                <Select>
                  <Select.Option value="basic">Basic (Essential events only)</Select.Option>
                  <Select.Option value="detailed">Detailed (Most events)</Select.Option>
                  <Select.Option value="verbose">Verbose (All events)</Select.Option>
                </Select>
              </Form.Item>
              
              <Form.Item
                name={['auditLogging', 'retentionDays']}
                label="Log Retention (days)"
              >
                <Input type="number" min={1} max={365} />
              </Form.Item>
              
              <div className="form-actions">
                <Button type="primary" htmlType="submit">
                  Save Settings
                </Button>
              </div>
            </Form>
          </Card>
          
          <Card title="Recent Audit Logs" style={{ marginTop: '24px' }}>
            <Table
              dataSource={auditLogs || []}
              columns={auditColumns}
              loading={auditLoading}
              rowKey="id"
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
}; 