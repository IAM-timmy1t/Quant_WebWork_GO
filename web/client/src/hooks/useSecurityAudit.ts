/**
 * useSecurityAudit.ts
 * 
 * @module hooks
 * @description Custom hook for fetching and managing security audit logs
 * @version 1.0.0
 */

import { useState, useEffect, useCallback } from 'react';

export interface AuditLogEntry {
  id: string;
  timestamp: string;
  userID: string;
  ipAddress: string;
  action: string;
  resource: string;
  result: string;
  category: string;
  severity: string;
  description: string;
  details?: any;
  sessionID?: string;
  environment: string;
  requestID?: string;
}

export interface AuditLogFilter {
  startDate?: Date;
  endDate?: Date;
  userId?: string;
  ipAddress?: string;
  action?: string;
  category?: string;
  severity?: string;
  result?: string;
  environment?: string;
}

export interface UseSecurityAuditOptions {
  // How often to refresh audit logs (in milliseconds)
  refreshInterval?: number;
  
  // Whether to automatically refresh
  autoRefresh?: boolean;
  
  // Maximum number of logs to fetch
  limit?: number;
  
  // Initial filter
  initialFilter?: AuditLogFilter;
}

export interface UseSecurityAuditResult {
  // Audit logs
  auditLogs: AuditLogEntry[] | null;
  
  // Current filter
  filter: AuditLogFilter;
  
  // Loading state
  loading: boolean;
  
  // Error state
  error: Error | null;
  
  // Functions
  refresh: () => Promise<void>;
  setFilter: (filter: AuditLogFilter) => void;
  exportLogs: (format: 'csv' | 'json') => Promise<string>;
  
  // Pagination
  totalCount: number;
  page: number;
  setPage: (page: number) => void;
  pageSize: number;
  setPageSize: (size: number) => void;
}

/**
 * Hook for fetching and managing security audit logs
 */
export function useSecurityAudit(options?: UseSecurityAuditOptions): UseSecurityAuditResult {
  const {
    refreshInterval = 60000, // Default to 1 minute
    autoRefresh = true,
    limit = 100,
    initialFilter = {},
  } = options || {};
  
  const [auditLogs, setAuditLogs] = useState<AuditLogEntry[] | null>(null);
  const [filter, setFilter] = useState<AuditLogFilter>(initialFilter);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [page, setPage] = useState<number>(1);
  const [pageSize, setPageSize] = useState<number>(20);
  
  // Fetch audit logs
  const fetchAuditLogs = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Prepare query parameters
      const params = new URLSearchParams();
      params.append('limit', pageSize.toString());
      params.append('offset', ((page - 1) * pageSize).toString());
      
      if (filter.startDate) {
        params.append('startDate', filter.startDate.toISOString());
      }
      
      if (filter.endDate) {
        params.append('endDate', filter.endDate.toISOString());
      }
      
      if (filter.userId) {
        params.append('userId', filter.userId);
      }
      
      if (filter.ipAddress) {
        params.append('ipAddress', filter.ipAddress);
      }
      
      if (filter.action) {
        params.append('action', filter.action);
      }
      
      if (filter.category) {
        params.append('category', filter.category);
      }
      
      if (filter.severity) {
        params.append('severity', filter.severity);
      }
      
      if (filter.result) {
        params.append('result', filter.result);
      }
      
      if (filter.environment) {
        params.append('environment', filter.environment);
      }
      
      // Fetch audit logs from API
      const response = await fetch(`/api/v1/security/audit?${params.toString()}`);
      
      if (!response.ok) {
        throw new Error(`Failed to fetch audit logs: ${response.statusText}`);
      }
      
      const data = await response.json();
      
      setAuditLogs(data.logs);
      setTotalCount(data.total);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  }, [filter, page, pageSize]);
  
  // Refresh audit logs manually
  const refresh = useCallback(async () => {
    await fetchAuditLogs();
  }, [fetchAuditLogs]);
  
  // Update filter
  const updateFilter = useCallback((newFilter: AuditLogFilter) => {
    setFilter(newFilter);
    setPage(1); // Reset to first page when filter changes
  }, []);
  
  // Export logs to CSV or JSON
  const exportLogs = useCallback(async (format: 'csv' | 'json'): Promise<string> => {
    try {
      // Prepare query parameters
      const params = new URLSearchParams();
      params.append('format', format);
      params.append('limit', limit.toString());
      
      if (filter.startDate) {
        params.append('startDate', filter.startDate.toISOString());
      }
      
      if (filter.endDate) {
        params.append('endDate', filter.endDate.toISOString());
      }
      
      if (filter.userId) {
        params.append('userId', filter.userId);
      }
      
      if (filter.ipAddress) {
        params.append('ipAddress', filter.ipAddress);
      }
      
      if (filter.action) {
        params.append('action', filter.action);
      }
      
      if (filter.category) {
        params.append('category', filter.category);
      }
      
      if (filter.severity) {
        params.append('severity', filter.severity);
      }
      
      if (filter.result) {
        params.append('result', filter.result);
      }
      
      if (filter.environment) {
        params.append('environment', filter.environment);
      }
      
      // Fetch export from API
      const response = await fetch(`/api/v1/security/audit/export?${params.toString()}`);
      
      if (!response.ok) {
        throw new Error(`Failed to export audit logs: ${response.statusText}`);
      }
      
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      
      return url;
    } catch (err) {
      setError(err as Error);
      throw err;
    }
  }, [filter, limit]);
  
  // Set up auto-refresh
  useEffect(() => {
    // Initial fetch
    fetchAuditLogs();
    
    // Set up interval if auto-refresh is enabled
    if (autoRefresh && refreshInterval > 0) {
      const intervalId = setInterval(fetchAuditLogs, refreshInterval);
      
      // Clean up on unmount
      return () => {
        clearInterval(intervalId);
      };
    }
  }, [fetchAuditLogs, autoRefresh, refreshInterval]);
  
  return {
    auditLogs,
    filter,
    loading,
    error,
    refresh,
    setFilter: updateFilter,
    exportLogs,
    totalCount,
    page,
    setPage,
    pageSize,
    setPageSize,
  };
} 