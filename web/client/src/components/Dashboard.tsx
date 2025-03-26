/**
 * Dashboard.tsx
 * 
 * @module components
 * @description Main dashboard view providing system overview and navigation
 * @version 1.0.0
 */

import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Card, Grid, Statistic, Button, Tabs, Progress, Alert } from './ui';
import { useMetrics } from '../hooks/useMetrics';
import { useConfig } from '../hooks/useConfig';
import { useBridge } from '../hooks/useBridge';
import { MetricsChart } from './charts/MetricsChart';

// Helper function to format bytes to a human-readable string
function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 Bytes';
  
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

interface SystemHealth {
  status: 'healthy' | 'warning' | 'critical';
  issues: Array<{ 
    severity: 'info' | 'warning' | 'error'; 
    message: string;
    details?: string;
  }>;
}

// Dashboard component
export const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const { metrics, loading: metricsLoading, error: metricsError } = useMetrics();
  const { config, loading: configLoading } = useConfig();
  const { connections, status: bridgeStatus } = useBridge();
  
  const [systemHealth, setSystemHealth] = useState<SystemHealth>({ 
    status: 'healthy', 
    issues: [] 
  });
  
  // Update system health based on metrics and configuration
  useEffect(() => {
    if (metricsLoading || configLoading) return;
    
    const issues = [];
    let status = 'healthy';
    
    // Check for high CPU usage
    if (metrics?.system?.cpu?.usage > 80) {
      issues.push({
        severity: 'warning',
        message: 'High CPU usage detected',
        details: `Current usage: ${metrics.system.cpu.usage}%`
      });
      status = 'warning';
    }
    
    // Check for high memory usage
    if (metrics?.system?.memory?.usagePercent > 90) {
      issues.push({
        severity: 'warning',
        message: 'High memory usage detected',
        details: `Current usage: ${metrics.system.memory.usagePercent}%`
      });
      status = 'warning';
    }
    
    // Check for security issues
    if (config?.environment === 'production' && !config?.security?.tlsEnabled) {
      issues.push({
        severity: 'error',
        message: 'TLS is not enabled in production',
        details: 'This is a security risk, enable TLS in security settings'
      });
      status = 'critical';
    }
    
    // Check bridge status
    if (bridgeStatus === 'disconnected') {
      issues.push({
        severity: 'warning',
        message: 'Bridge system is disconnected',
        details: 'Check network connectivity or bridge configuration'
      });
      status = 'warning';
    }
    
    setSystemHealth({ status, issues });
  }, [metrics, config, bridgeStatus, metricsLoading, configLoading]);
  
  // Handle quick actions
  const handleAddBridge = () => {
    navigate('/bridges/new');
  };
  
  const handleSecuritySettings = () => {
    navigate('/security');
  };
  
  const handleRefreshMetrics = () => {
    // This would trigger a refresh of metrics
    // assuming useMetrics has a refresh function
    if ('refresh' in useMetrics) {
      (useMetrics as any).refresh();
    }
  };
  
  // Render content based on loading state
  if (metricsLoading || configLoading) {
    return (
      <div className="dashboard dashboard-loading">
        <h2>Loading dashboard...</h2>
        <Progress percent={50} status="active" />
      </div>
    );
  }
  
  if (metricsError) {
    return (
      <div className="dashboard dashboard-error">
        <Alert 
          type="error" 
          message="Error loading dashboard" 
          description={metricsError.toString()}
        />
        <Button onClick={handleRefreshMetrics} type="primary">
          Retry
        </Button>
      </div>
    );
  }

  // Helper to get alert status
  const getAlertStatus = (health: SystemHealth) => {
    switch (health.status) {
      case 'critical': return 'error';
      case 'warning': return 'warning';
      default: return 'success';
    }
  };
  
  return (
    <div className="dashboard">
      <div className="dashboard-header">
        <h1>System Dashboard</h1>
        <div className="dashboard-actions">
          <Button onClick={handleAddBridge} type="primary">Add Bridge</Button>
          <Button onClick={handleSecuritySettings}>Security Settings</Button>
          <Button onClick={handleRefreshMetrics} icon="refresh">Refresh</Button>
        </div>
      </div>
      
      {/* System health summary */}
      <Card className="health-card">
        <div className="health-status">
          <h3>System Health</h3>
          <Alert
            message={`System Status: ${systemHealth.status.toUpperCase()}`}
            type={getAlertStatus(systemHealth)}
            showIcon
          />
        </div>
        
        {systemHealth.issues.length > 0 && (
          <div className="health-issues">
            <h4>Issues Detected</h4>
            {systemHealth.issues.map((issue, index) => (
              <Alert
                key={index}
                message={issue.message}
                description={issue.details}
                type={issue.severity}
                showIcon
                style={{ marginBottom: '8px' }}
              />
            ))}
          </div>
        )}
      </Card>
      
      {/* Key metrics */}
      <Grid className="metrics-grid">
        <Card>
          <Statistic
            title="Active Connections"
            value={connections?.length || 0}
            suffix="/ 100"
            valueStyle={{ color: connections?.length > 80 ? '#ff4d4f' : '#3f8600' }}
          />
        </Card>
        <Card>
          <Statistic
            title="CPU Usage"
            value={metrics?.system?.cpu?.usage || 0}
            suffix="%"
            precision={1}
            valueStyle={{ 
              color: (metrics?.system?.cpu?.usage || 0) > 80 ? '#ff4d4f' : '#3f8600' 
            }}
          />
        </Card>
        <Card>
          <Statistic
            title="Memory Usage"
            value={metrics?.system?.memory?.usagePercent || 0}
            suffix="%"
            precision={1}
            valueStyle={{ 
              color: (metrics?.system?.memory?.usagePercent || 0) > 90 ? '#ff4d4f' : '#3f8600' 
            }}
          />
        </Card>
        <Card>
          <Statistic
            title="Network Traffic"
            value={formatBytes(metrics?.system?.network?.bytesPerSecond || 0)}
            suffix="/s"
          />
        </Card>
      </Grid>
      
      {/* Detailed metrics */}
      <Card className="detailed-metrics">
        <Tabs defaultActiveKey="system">
          <Tabs.TabPane tab="System Metrics" key="system">
            <div className="chart-container">
              <h3>CPU & Memory Usage (24h)</h3>
              <MetricsChart 
                data={metrics?.history?.system || []}
                metrics={['cpu.usage', 'memory.usagePercent']}
                labels={['CPU', 'Memory']}
                colors={['#1890ff', '#52c41a']}
              />
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane tab="Network Metrics" key="network">
            <div className="chart-container">
              <h3>Network Traffic (24h)</h3>
              <MetricsChart 
                data={metrics?.history?.network || []}
                metrics={['bytesIn', 'bytesOut']}
                labels={['Incoming', 'Outgoing']}
                colors={['#722ed1', '#13c2c2']}
                formatY={formatBytes}
              />
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane tab="Bridge Metrics" key="bridge">
            <div className="chart-container">
              <h3>Bridge Performance (24h)</h3>
              <MetricsChart 
                data={metrics?.history?.bridge || []}
                metrics={['requestsPerSecond', 'errorRate']}
                labels={['Requests/s', 'Error Rate %']}
                colors={['#fa8c16', '#f5222d']}
              />
            </div>
          </Tabs.TabPane>
        </Tabs>
      </Card>
      
      {/* Quick links */}
      <div className="dashboard-links">
        <h3>Quick Links</h3>
        <div className="links-grid">
          <Link to="/bridges" className="dashboard-link">
            <Card hoverable>
              <h4>Bridge Management</h4>
              <p>Manage connection bridges and monitor status</p>
            </Card>
          </Link>
          <Link to="/security" className="dashboard-link">
            <Card hoverable>
              <h4>Security Settings</h4>
              <p>Configure security options and review audit logs</p>
            </Card>
          </Link>
          <Link to="/monitoring" className="dashboard-link">
            <Card hoverable>
              <h4>Detailed Monitoring</h4>
              <p>View comprehensive system metrics and logs</p>
            </Card>
          </Link>
          <Link to="/configuration" className="dashboard-link">
            <Card hoverable>
              <h4>System Configuration</h4>
              <p>Manage general system settings and parameters</p>
            </Card>
          </Link>
        </div>
      </div>
    </div>
  );
}; 