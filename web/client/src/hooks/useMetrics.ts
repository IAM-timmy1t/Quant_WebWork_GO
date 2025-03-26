/**
 * useMetrics.ts
 * 
 * @module hooks
 * @description Custom hook for fetching and managing system metrics
 * @version 1.0.0
 */

import { useState, useEffect, useCallback } from 'react';
import { MetricsCollector } from '../monitoring/MetricsCollector';

// Define metric types
export interface SystemMetrics {
  system: {
    cpu: {
      usage: number;
      cores: number;
      loadAverage: number[];
    };
    memory: {
      total: number;
      used: number;
      free: number;
      usagePercent: number;
    };
    disk: {
      total: number;
      used: number;
      free: number;
      usagePercent: number;
    };
    network: {
      bytesPerSecond: number;
      bytesIn: number;
      bytesOut: number;
      connectionsActive: number;
    };
  };
  bridges: {
    active: number;
    connected: number;
    disconnected: number;
    errors: number;
  };
  requests: {
    total: number;
    success: number;
    failed: number;
    avgResponseTime: number;
  };
  history: {
    system: Array<{
      timestamp: string;
      'cpu.usage': number;
      'memory.usagePercent': number;
      'disk.usagePercent': number;
    }>;
    network: Array<{
      timestamp: string;
      bytesIn: number;
      bytesOut: number;
      connectionsActive: number;
    }>;
    bridge: Array<{
      timestamp: string;
      requestsPerSecond: number;
      errorRate: number;
      avgResponseTime: number;
    }>;
  };
}

export interface UseMetricsOptions {
  // How often to refresh metrics (in milliseconds)
  refreshInterval?: number;
  
  // Whether to automatically refresh
  autoRefresh?: boolean;
  
  // Whether to fetch historical data
  includeHistory?: boolean;
  
  // Historical data time range (in hours)
  historyTimeRange?: number;
}

export interface UseMetricsResult {
  // Current metrics
  metrics: SystemMetrics | null;
  
  // Loading state
  loading: boolean;
  
  // Error state
  error: Error | null;
  
  // Manual refresh function
  refresh: () => Promise<void>;
  
  // Last updated timestamp
  lastUpdated: Date | null;
}

/**
 * Hook for fetching and managing system metrics
 */
export function useMetrics(options?: UseMetricsOptions): UseMetricsResult {
  const {
    refreshInterval = 30000, // Default to 30 seconds
    autoRefresh = true,
    includeHistory = true,
    historyTimeRange = 24, // Default to 24 hours
  } = options || {};
  
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [collector] = useState(() => new MetricsCollector());
  
  // Fetch metrics function
  const fetchMetrics = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Get current metrics
      const currentMetrics = await collector.getMetrics();
      
      // Get historical data if requested
      let historicalData = {};
      if (includeHistory) {
        const endTime = new Date();
        const startTime = new Date(endTime.getTime() - (historyTimeRange * 60 * 60 * 1000));
        
        historicalData = {
          history: await collector.getHistoricalMetrics(startTime, endTime),
        };
      }
      
      // Update state
      setMetrics({ ...currentMetrics, ...historicalData } as SystemMetrics);
      setLastUpdated(new Date());
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  }, [collector, includeHistory, historyTimeRange]);
  
  // Refresh metrics manually
  const refresh = useCallback(async () => {
    await fetchMetrics();
  }, [fetchMetrics]);
  
  // Set up auto-refresh
  useEffect(() => {
    // Initial fetch
    fetchMetrics();
    
    // Set up interval if auto-refresh is enabled
    if (autoRefresh && refreshInterval > 0) {
      const intervalId = setInterval(fetchMetrics, refreshInterval);
      
      // Clean up on unmount
      return () => {
        clearInterval(intervalId);
      };
    }
  }, [fetchMetrics, autoRefresh, refreshInterval]);
  
  return {
    metrics,
    loading,
    error,
    refresh,
    lastUpdated,
  };
} 