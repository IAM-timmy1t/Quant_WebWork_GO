/**
 * SecurityCheck.tsx
 * 
 * @module components/Onboarding
 * @description Component for checking and configuring security settings during onboarding.
 * @version 1.0.0
 */

import React, { useState, useEffect } from 'react';
import { Card, Checkbox, Select, Progress, Alert, List } from '../ui';
import { useConfig } from '../../hooks/useConfig';
import axios from 'axios';

interface SecurityCheckProps {
  onScoreChange: (score: number) => void;
}

interface SecuritySetting {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  critical: boolean;
  weight: number;
  configKey: string;
  dependsOn?: string[];
}

export const SecurityCheck: React.FC<SecurityCheckProps> = ({ onScoreChange }) => {
  const { config, updateConfig } = useConfig();
  const [securityScore, setSecurityScore] = useState(0);
  const [securitySettings, setSecuritySettings] = useState<SecuritySetting[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Define security settings
  useEffect(() => {
    // In a real app, you might want to fetch these from an API
    const defaultSettings: SecuritySetting[] = [
      {
        id: 'auth',
        name: 'Authentication',
        description: 'Require users to authenticate before accessing the system',
        enabled: config?.security?.authentication?.enabled || false,
        critical: true,
        weight: 20,
        configKey: 'security.authentication.enabled'
      },
      {
        id: 'tls',
        name: 'TLS/HTTPS',
        description: 'Secure all communications with TLS encryption',
        enabled: config?.security?.tls?.enabled || false,
        critical: true,
        weight: 20,
        configKey: 'security.tls.enabled'
      },
      {
        id: 'ipMasking',
        name: 'IP Masking',
        description: 'Hide client IPs from destination services',
        enabled: config?.security?.ipMasking?.enabled || false,
        critical: false,
        weight: 15,
        configKey: 'security.ipMasking.enabled'
      },
      {
        id: 'rateLimiting',
        name: 'Rate Limiting',
        description: 'Limit request rates to prevent abuse',
        enabled: config?.security?.rateLimiting?.enabled || false,
        critical: false,
        weight: 15,
        configKey: 'security.rateLimiting.enabled'
      },
      {
        id: 'firewall',
        name: 'Firewall Rules',
        description: 'Restrict access based on IP and request patterns',
        enabled: config?.security?.firewall?.enabled || false,
        critical: false,
        weight: 15,
        configKey: 'security.firewall.enabled'
      },
      {
        id: 'auditLogging',
        name: 'Audit Logging',
        description: 'Log all sensitive operations for security review',
        enabled: config?.security?.auditLogging?.enabled || false,
        critical: false,
        weight: 10,
        configKey: 'security.auditLogging.enabled'
      },
      {
        id: 'dnsPrivacy',
        name: 'DNS Privacy',
        description: 'Prevent DNS leaks that could reveal service identity',
        enabled: config?.security?.ipMasking?.dnsPrivacyEnabled || false,
        critical: false,
        weight: 5,
        configKey: 'security.ipMasking.dnsPrivacyEnabled',
        dependsOn: ['ipMasking']
      }
    ];

    setSecuritySettings(defaultSettings);
    setLoading(false);

    // Calculate initial score
    calculateScore(defaultSettings);
  }, [config]);

  // Calculate security score based on enabled settings
  const calculateScore = (settings: SecuritySetting[]) => {
    const totalWeight = settings.reduce((sum, setting) => sum + setting.weight, 0);
    const enabledWeight = settings.reduce((sum, setting) => 
      setting.enabled ? sum + setting.weight : sum, 0);
    
    const score = Math.round((enabledWeight / totalWeight) * 100);
    setSecurityScore(score);
    onScoreChange(score);
    
    return score;
  };

  // Toggle a security setting
  const toggleSetting = async (settingId: string, enabled: boolean) => {
    try {
      // Find the setting
      const setting = securitySettings.find(s => s.id === settingId);
      if (!setting) return;

      // Update in UI first for responsiveness
      const updatedSettings = securitySettings.map(s => 
        s.id === settingId ? { ...s, enabled } : s
      );
      setSecuritySettings(updatedSettings);
      
      // Calculate and update score
      calculateScore(updatedSettings);

      // Update the config
      if (setting.configKey) {
        await updateConfig(setting.configKey, enabled);
      }

      // If this setting was disabled and other settings depend on it,
      // also disable those dependent settings
      if (!enabled) {
        const dependentSettings = securitySettings.filter(s => 
          s.dependsOn && s.dependsOn.includes(settingId)
        );
        
        for (const depSetting of dependentSettings) {
          await toggleSetting(depSetting.id, false);
        }
      }
    } catch (err) {
      setError('Failed to update security setting');
      console.error('Error updating security setting:', err);
    }
  };

  // Get security level description
  const getSecurityLevel = (score: number) => {
    if (score >= 90) return { level: 'Excellent', color: 'success' };
    if (score >= 70) return { level: 'Good', color: 'success' };
    if (score >= 50) return { level: 'Moderate', color: 'warning' };
    return { level: 'Poor', color: 'error' };
  };

  const securityLevel = getSecurityLevel(securityScore);

  // Check if any critical settings are disabled
  const criticalDisabled = securitySettings.some(s => s.critical && !s.enabled);

  return (
    <div className="security-check">
      {loading ? (
        <div className="loading">Loading security settings...</div>
      ) : error ? (
        <Alert type="error">{error}</Alert>
      ) : (
        <>
          <div className="security-summary">
            <h4>Security Score: {securityScore}%</h4>
            <Progress 
              percent={securityScore} 
              strokeColor={securityLevel.color} 
            />
            <Alert 
              type={criticalDisabled ? "error" : (securityLevel.color as any)} 
              showIcon
            >
              {criticalDisabled 
                ? "Critical security features are disabled! This is not recommended for production."
                : `Your security level is ${securityLevel.level}`
              }
            </Alert>
          </div>

          <Card title="Security Settings" className="settings-card">
            <List
              itemLayout="horizontal"
              dataSource={securitySettings}
              renderItem={item => (
                <List.Item>
                  <List.Item.Meta
                    title={
                      <div className="setting-title">
                        {item.name} 
                        {item.critical && <span className="critical-tag">Critical</span>}
                      </div>
                    }
                    description={item.description}
                  />
                  <Checkbox 
                    checked={item.enabled}
                    onChange={e => toggleSetting(item.id, e.target.checked)}
                    disabled={item.dependsOn?.some(dep => 
                      !securitySettings.find(s => s.id === dep)?.enabled
                    )}
                  />
                </List.Item>
              )}
            />
          </Card>

          <div className="additional-info">
            <h4>Security Recommendations</h4>
            <p>
              For production environments, we strongly recommend enabling all critical
              security features. Authentication and TLS are essential for protecting
              your services from unauthorized access.
            </p>
          </div>
        </>
      )}
    </div>
  );
};
