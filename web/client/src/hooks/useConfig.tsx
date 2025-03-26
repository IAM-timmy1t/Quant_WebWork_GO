/**
 * useConfig.tsx
 * 
 * @module hooks
 * @description Custom hook for reading and updating application configuration
 * @version 1.0.0
 */

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import axios from 'axios';
import { set, get } from 'lodash-es';

// Define configuration structure with TypeScript for better type safety
interface SecurityConfig {
  authentication: {
    enabled: boolean;
    provider: string;
    sessionDuration: number;
  };
  tls: {
    enabled: boolean;
    certPath: string;
    keyPath: string;
  };
  ipMasking: {
    enabled: boolean;
    dnsPrivacyEnabled: boolean;
  };
  rateLimiting: {
    enabled: boolean;
    level: string;
  };
  firewall: {
    enabled: boolean;
    rules: any[];
  };
  auditLogging: {
    enabled: boolean;
    level: string;
    retention: number;
  };
  adminConfigured: boolean;
}

interface BridgeConfig {
  enabled: boolean;
  defaultTimeout: number;
  maxConnections: number;
  connectionPoolSize: number;
}

export interface AppConfig {
  environment: string;
  version: string;
  security: SecurityConfig;
  bridge: BridgeConfig;
  [key: string]: any;
}

interface ConfigContextType {
  config: AppConfig | null;
  loading: boolean;
  error: string | null;
  reloadConfig: () => Promise<void>;
  updateConfig: (path: string, value: any) => Promise<boolean>;
  getConfigValue: (path: string, defaultValue?: any) => any;
}

// Create context with default values
const ConfigContext = createContext<ConfigContextType>({
  config: null,
  loading: false,
  error: null,
  reloadConfig: async () => {},
  updateConfig: async () => false,
  getConfigValue: () => undefined,
});

interface ConfigProviderProps {
  children: ReactNode;
}

export const ConfigProvider: React.FC<ConfigProviderProps> = ({ children }) => {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // Load configuration on mount
  useEffect(() => {
    reloadConfig();
  }, []);

  // Function to reload configuration from server
  const reloadConfig = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await axios.get('/api/v1/config');
      setConfig(response.data);
    } catch (err: any) {
      console.error('Failed to load configuration:', err);
      setError(err.message || 'Failed to load configuration');
      
      // Use default configuration for development purposes
      if (process.env.NODE_ENV === 'development') {
        setConfig(getDefaultConfig());
      }
    } finally {
      setLoading(false);
    }
  };

  // Function to update configuration
  const updateConfig = async (path: string, value: any): Promise<boolean> => {
    if (!config) return false;

    try {
      // Optimistically update the UI first
      const newConfig = { ...config };
      set(newConfig, path, value);
      setConfig(newConfig);

      // Save to backend
      await axios.patch('/api/v1/config', {
        path,
        value,
      });

      return true;
    } catch (err: any) {
      console.error('Failed to update configuration:', err);
      setError(err.message || 'Failed to update configuration');
      
      // Revert changes on error
      await reloadConfig();
      return false;
    }
  };

  // Function to get a value from the config with optional default
  const getConfigValue = (path: string, defaultValue?: any): any => {
    if (!config) return defaultValue;
    return get(config, path, defaultValue);
  };

  return (
    <ConfigContext.Provider
      value={{
        config,
        loading,
        error,
        reloadConfig,
        updateConfig,
        getConfigValue,
      }}
    >
      {children}
    </ConfigContext.Provider>
  );
};

// Hook for using the config context
export const useConfig = () => useContext(ConfigContext);

// Default configuration for development or fallback
function getDefaultConfig(): AppConfig {
  return {
    environment: 'development',
    version: '1.0.0-dev',
    security: {
      authentication: {
        enabled: false,
        provider: 'local',
        sessionDuration: 3600,
      },
      tls: {
        enabled: false,
        certPath: '',
        keyPath: '',
      },
      ipMasking: {
        enabled: false,
        dnsPrivacyEnabled: false,
      },
      rateLimiting: {
        enabled: false,
        level: 'medium',
      },
      firewall: {
        enabled: false,
        rules: [],
      },
      auditLogging: {
        enabled: false,
        level: 'basic',
        retention: 30,
      },
      adminConfigured: false,
    },
    bridge: {
      enabled: true,
      defaultTimeout: 30,
      maxConnections: 100,
      connectionPoolSize: 10,
    },
  };
}
