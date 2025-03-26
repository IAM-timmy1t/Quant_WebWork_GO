/**
 * useBridge.ts
 * 
 * @module hooks
 * @description Custom hook for accessing bridge functionality and managing connections
 * @version 1.0.0
 */

import { useState, useEffect, useCallback } from 'react';
import { BridgeClient } from '../bridge/BridgeClient';

export interface BridgeConnection {
  id: string;
  name: string;
  protocol: string;
  host: string;
  port: number;
  status: 'connected' | 'disconnected' | 'error';
  lastConnected?: Date;
  metrics?: {
    requestsTotal: number;
    errorsTotal: number;
    bytesTransferred: number;
    averageResponseTime: number;
  };
}

export interface BridgeMessage {
  id: string;
  connectionId: string;
  type: 'request' | 'response' | 'notification' | 'error';
  timestamp: Date;
  payload: any;
  size: number;
}

interface UseBridgeResult {
  // Bridge client instance
  client: BridgeClient | null;
  
  // List of active connections
  connections: BridgeConnection[];
  
  // Current status of the bridge client
  status: 'connected' | 'disconnected' | 'connecting' | 'error';
  
  // Error message if status is 'error'
  error: Error | null;
  
  // Recent messages (limited to mostRecentCount)
  recentMessages: BridgeMessage[];
  
  // Actions
  connect: (config?: any) => Promise<void>;
  disconnect: () => void;
  createConnection: (config: Omit<BridgeConnection, 'id' | 'status' | 'metrics'>) => Promise<string>;
  deleteConnection: (id: string) => Promise<boolean>;
  sendMessage: (connectionId: string, payload: any) => Promise<BridgeMessage>;
}

/**
 * Hook for interacting with the Bridge system
 */
export function useBridge(config?: any): UseBridgeResult {
  const [client, setClient] = useState<BridgeClient | null>(null);
  const [status, setStatus] = useState<'connected' | 'disconnected' | 'connecting' | 'error'>('disconnected');
  const [error, setError] = useState<Error | null>(null);
  const [connections, setConnections] = useState<BridgeConnection[]>([]);
  const [recentMessages, setRecentMessages] = useState<BridgeMessage[]>([]);
  
  // Initialize bridge client
  useEffect(() => {
    // Skip if config is not provided or client already exists
    if (!config || client) return;
    
    try {
      const newClient = new BridgeClient(config);
      setClient(newClient);
      
      // Set up event listeners
      newClient.on('connect', () => {
        setStatus('connected');
        setError(null);
      });
      
      newClient.on('disconnect', () => {
        setStatus('disconnected');
      });
      
      newClient.on('error', (err) => {
        setStatus('error');
        setError(err);
      });
      
      newClient.on('message', (message) => {
        setRecentMessages((prevMessages) => {
          // Keep only the most recent 50 messages
          const updatedMessages = [message, ...prevMessages];
          return updatedMessages.slice(0, 50);
        });
      });
      
      // Clean up on unmount
      return () => {
        newClient.disconnect();
      };
    } catch (err) {
      setStatus('error');
      setError(err as Error);
    }
  }, [config]);
  
  // Fetch connections on client change or status change
  useEffect(() => {
    if (!client || status !== 'connected') return;
    
    const fetchConnections = async () => {
      try {
        const connectionsList = await client.getConnections();
        setConnections(connectionsList);
      } catch (err) {
        setError(err as Error);
      }
    };
    
    fetchConnections();
    
    // Set up an interval to refresh connections
    const interval = setInterval(fetchConnections, 5000);
    
    return () => {
      clearInterval(interval);
    };
  }, [client, status]);
  
  /**
   * Connect to the bridge server
   */
  const connect = useCallback(async (connectConfig?: any) => {
    if (!client) {
      setError(new Error('Bridge client not initialized'));
      return;
    }
    
    try {
      setStatus('connecting');
      await client.connect(connectConfig);
    } catch (err) {
      setStatus('error');
      setError(err as Error);
    }
  }, [client]);
  
  /**
   * Disconnect from the bridge server
   */
  const disconnect = useCallback(() => {
    if (!client) return;
    
    client.disconnect();
    setStatus('disconnected');
  }, [client]);
  
  /**
   * Create a new bridge connection
   */
  const createConnection = useCallback(async (connectionConfig: Omit<BridgeConnection, 'id' | 'status' | 'metrics'>) => {
    if (!client) {
      throw new Error('Bridge client not initialized');
    }
    
    const newConnectionId = await client.createConnection(connectionConfig);
    
    // Refresh connections list
    const connectionsList = await client.getConnections();
    setConnections(connectionsList);
    
    return newConnectionId;
  }, [client]);
  
  /**
   * Delete a bridge connection
   */
  const deleteConnection = useCallback(async (id: string) => {
    if (!client) {
      throw new Error('Bridge client not initialized');
    }
    
    const success = await client.deleteConnection(id);
    
    if (success) {
      // Remove connection from state
      setConnections((prevConnections) => 
        prevConnections.filter(conn => conn.id !== id)
      );
    }
    
    return success;
  }, [client]);
  
  /**
   * Send a message through a bridge connection
   */
  const sendMessage = useCallback(async (connectionId: string, payload: any) => {
    if (!client) {
      throw new Error('Bridge client not initialized');
    }
    
    const message = await client.sendMessage(connectionId, payload);
    
    // Add message to recent messages
    setRecentMessages((prevMessages) => {
      const updatedMessages = [message, ...prevMessages];
      return updatedMessages.slice(0, 50);
    });
    
    return message;
  }, [client]);
  
  return {
    client,
    status,
    error,
    connections,
    recentMessages,
    connect,
    disconnect,
    createConnection,
    deleteConnection,
    sendMessage,
  };
} 