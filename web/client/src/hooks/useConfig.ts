/**
 * useConfig.ts
 * 
 * @module hooks
 * @description Custom hook for managing system configuration
 * @version 1.0.0
 */

import { useState, useEffect, useCallback } from 'react';

// Define configuration types
export interface SecurityConfig {
  authRequired: boolean;
  sessionTimeout: number;
  tlsEnabled: boolean;
  tlsMinVersion: string;
  ipMasking: {
    enabled: boolean;
    rotationEnabled: boolean;
    rotationInterval: number;
    dnsPrivacyEnabled: boolean;
  };
  firewall: {
    enabled: boolean;
    strictMode: boolean;
    allowedIPs: string;
  };
  rateLimiting: {
    enabled: boolean;
    advanced: boolean;
    requestsPerMinute: number;
    burstSize: number;
  };
  auditLogging: {
    enabled: boolean;
    level: string;
    retentionDays: number;
  };
  dnsPrivacy: boolean;
}

export interface SystemConfig {
  environment: string;
  version: string;
  serverName: string;
  serverPort: number;
  serverHost: string;
  apiVersion: string;
  bridgePriority: string;
  maxConnections: number;
}

export interface Config {
  system: SystemConfig;
  security: SecurityConfig;
}

export interface UseConfigResult {
  // Current configuration
  config: Config | null;
  
  // Loading state
  loading: boolean;
  
  // Error state
  error: Error | null;
  
  // Update configuration
  updateConfig: (newConfig: Partial<Config>) => Promise<void>;
  
  // Reset configuration to defaults
  resetConfig: () => Promise<void>;
}

/**
 * Hook for managing system configuration
 */
export function useConfig(): UseConfigResult {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);
  
  // Fetch configuration
  const fetchConfig = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Fetch configuration from API
      const response = await fetch('/api/v1/config');
      
      if (!response.ok) {
        throw new Error(`Failed to fetch configuration: ${response.statusText}`);
      }
      
      const data = await response.json();
      setConfig(data);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  }, []);
  
  // Update configuration
  const updateConfig = useCallback(async (newConfig: Partial<Config>) => {
    try {
      setLoading(true);
      setError(null);
      
      // Send update to API
      const response = await fetch('/api/v1/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newConfig),
      });
      
      if (!response.ok) {
        throw new Error(`Failed to update configuration: ${response.statusText}`);
      }
      
      // Get updated configuration
      const data = await response.json();
      setConfig(data);
      setError(null);
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);
  
  // Reset configuration to defaults
  const resetConfig = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Send reset request to API
      const response = await fetch('/api/v1/config/reset', {
        method: 'POST',
      });
      
      if (!response.ok) {
        throw new Error(`Failed to reset configuration: ${response.statusText}`);
      }
      
      // Get updated configuration
      const data = await response.json();
      setConfig(data);
      setError(null);
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);
  
  // Fetch configuration on mount
  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);
  
  return {
    config,
    loading,
    error,
    updateConfig,
    resetConfig,
  };
} 