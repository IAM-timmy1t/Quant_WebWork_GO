/**
 * BridgeClient.ts
 * React-compatible bridge client for communication with the Go backend
 */

import { grpc } from '@improbable-eng/grpc-web';
import { BridgeServiceClient } from '../proto/bridge_grpc_web_pb';
import { 
  TokenRequest,
  TokenResponse,
  BridgeRequest,
  BridgeResponse,
  AnalysisRequest
} from '../proto/bridge_pb';
import { EventEmitter } from 'events';

// Message type definitions matching server-side
export enum MessageType {
  ANALYSIS_REQUEST = 'analysis-request',
  ANALYSIS_RESPONSE = 'analysis-response',
  RISK_REQUEST = 'risk-request',
  RISK_RESPONSE = 'risk-response',
  EVENT = 'event',
  METRICS = 'metrics',
  QUERY = 'query',
  QUERY_RESPONSE = 'query-response',
  UI_UPDATE = 'ui-update',
  ERROR = 'error',
}

// Client information
export interface ClientInfo {
  type: string;
  version: string;
  environment: string;
  platform: string;
  metadata?: Record<string, string>;
}

// Token metadata structure
export interface TokenMetadata {
  modelId: string;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  tokenizationMs: number;
  timestamp: Date;
  chunkIndex?: number;
  totalChunks?: number;
  compressionRate?: number;
  riskScore?: number;
  formatVersion?: string;
  clientInfo?: ClientInfo;
}

// Bridge message structure
export interface BridgeMessage {
  id: string;
  content: string;
  metadata: Record<string, any>;
  tokenInfo: TokenMetadata;
  timestamp: Date;
  type: MessageType;
  compression?: string;
  format?: string;
  version?: string;
}

/**
 * Bridge client options
 */
export interface BridgeClientOptions {
  serverUrl: string;
  enableMetrics?: boolean;
  timeout?: number;
  maxRetries?: number;
  clientInfo?: ClientInfo;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
}

/**
 * Bridge client for communicating with the Go backend
 */
export class BridgeClient extends EventEmitter {
  private client: BridgeServiceClient;
  private options: BridgeClientOptions;
  private connected: boolean = false;
  private messageCounter: number = 0;
  private pendingRequests: Map<string, { 
    resolve: (value: any) => void,
    reject: (reason: any) => void,
    timeout: NodeJS.Timeout
  }> = new Map();

  /**
   * Creates a new bridge client
   * @param options Client options
   */
  constructor(options: BridgeClientOptions) {
    super();
    this.options = {
      timeout: 30000,
      maxRetries: 3,
      enableMetrics: true,
      ...options,
    };

    // Create gRPC client
    this.client = new BridgeServiceClient(this.options.serverUrl, {
      transport: grpc.WebsocketTransport(),
    });

    // Default client info
    if (!this.options.clientInfo) {
      this.options.clientInfo = {
        type: 'web',
        version: '1.0.0',
        environment: process.env.NODE_ENV || 'development',
        platform: this.detectPlatform(),
        metadata: {}
      };
    }
  }

  /**
   * Connects to the bridge server
   */
  public async connect(): Promise<void> {
    if (this.connected) {
      return;
    }

    try {
      // Create connection request
      const request = new BridgeRequest();
      request.setType('connect');
      request.setClientInfo(JSON.stringify(this.options.clientInfo));

      // Call server
      await new Promise<void>((resolve, reject) => {
        this.client.connect(request, (err, response) => {
          if (err) {
            reject(err);
            return;
          }

          this.connected = true;
          if (this.options.onConnect) {
            this.options.onConnect();
          }
          this.emit('connected');
          resolve();
        });
      });
    } catch (error) {
      if (this.options.onError) {
        this.options.onError(error as Error);
      }
      throw error;
    }
  }

  /**
   * Disconnects from the bridge server
   */
  public async disconnect(): Promise<void> {
    if (!this.connected) {
      return;
    }

    try {
      // Create disconnect request
      const request = new BridgeRequest();
      request.setType('disconnect');

      // Call server
      await new Promise<void>((resolve, reject) => {
        this.client.disconnect(request, (err, response) => {
          if (err) {
            reject(err);
            return;
          }

          this.connected = false;
          if (this.options.onDisconnect) {
            this.options.onDisconnect();
          }
          this.emit('disconnected');
          resolve();
        });
      });
    } catch (error) {
      if (this.options.onError) {
        this.options.onError(error as Error);
      }
      throw error;
    } finally {
      // Clear all pending requests
      for (const [id, request] of this.pendingRequests.entries()) {
        clearTimeout(request.timeout);
        request.reject(new Error('Client disconnected'));
        this.pendingRequests.delete(id);
      }
    }
  }

  /**
   * Sends a message to the bridge server
   * @param type Message type
   * @param content Message content
   * @param metadata Message metadata
   */
  public async sendMessage(
    type: MessageType,
    content: string,
    metadata: Record<string, any> = {}
  ): Promise<BridgeMessage> {
    if (!this.connected) {
      await this.connect();
    }

    // Create message ID
    const id = this.generateMessageId();

    // Create token info
    const tokenInfo: TokenMetadata = {
      modelId: metadata.modelId || 'default',
      inputTokens: 0,
      outputTokens: 0,
      totalTokens: 0,
      tokenizationMs: 0,
      timestamp: new Date(),
      clientInfo: this.options.clientInfo
    };

    // Create bridge message
    const bridgeMessage: BridgeMessage = {
      id,
      content,
      metadata,
      tokenInfo,
      timestamp: new Date(),
      type,
      version: '1.0.0'
    };

    // Create request
    const request = new TokenRequest();
    request.setId(id);
    request.setType(type);
    request.setContent(content);
    request.setMetadata(JSON.stringify(metadata));
    request.setTimestamp(Date.now());

    // Send request and wait for response
    return new Promise<BridgeMessage>((resolve, reject) => {
      // Set timeout
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(id);
        reject(new Error(`Request timeout after ${this.options.timeout}ms`));
      }, this.options.timeout as number);

      // Store pending request
      this.pendingRequests.set(id, { resolve, reject, timeout });

      // Call server
      this.client.processMessage(request, (err, response) => {
        // Clear timeout and remove from pending requests
        clearTimeout(timeout);
        this.pendingRequests.delete(id);

        if (err) {
          reject(err);
          return;
        }

        try {
          // Parse response metadata
          const metadata = JSON.parse(response.getMetadata() || '{}');
          const responseTokenInfo = metadata.token_info || {};

          // Create response message
          const responseMessage: BridgeMessage = {
            id: response.getId(),
            content: response.getContent(),
            metadata,
            tokenInfo: {
              modelId: responseTokenInfo.model_id || tokenInfo.modelId,
              inputTokens: responseTokenInfo.input_tokens || 0,
              outputTokens: responseTokenInfo.output_tokens || 0,
              totalTokens: responseTokenInfo.total_tokens || 0,
              tokenizationMs: responseTokenInfo.tokenization_ms || 0,
              timestamp: new Date(response.getTimestamp()),
              riskScore: responseTokenInfo.risk_score
            },
            timestamp: new Date(response.getTimestamp()),
            type: response.getType() as MessageType,
            version: metadata.version || '1.0.0'
          };

          // Check if it's an error
          if (responseMessage.type === MessageType.ERROR) {
            const error = new Error(metadata.error || 'Unknown error');
            this.emit('error', error, responseMessage);
            
            if (this.options.onError) {
              this.options.onError(error);
            }
            
            // Still resolve with the error message
            resolve(responseMessage);
          } else {
            resolve(responseMessage);
          }

          // Emit message event
          this.emit('message', responseMessage);
        } catch (parseError) {
          reject(new Error(`Failed to parse response: ${parseError}`));
        }
      });
    });
  }

  /**
   * Performs token analysis
   * @param tokenAddress Token address to analyze
   */
  public async analyzeToken(tokenAddress: string): Promise<any> {
    const response = await this.sendMessage(
      MessageType.ANALYSIS_REQUEST,
      '',
      { token_address: tokenAddress }
    );

    if (response.type === MessageType.ERROR) {
      throw new Error(response.metadata.error || 'Analysis failed');
    }

    return response.metadata.analysis_result;
  }

  /**
   * Performs risk analysis on multiple tokens
   * @param tokenAddresses Token addresses to analyze
   */
  public async analyzeRisk(tokenAddresses: string[]): Promise<Record<string, any>> {
    const response = await this.sendMessage(
      MessageType.RISK_REQUEST,
      '',
      { token_addresses: tokenAddresses }
    );

    if (response.type === MessageType.ERROR) {
      throw new Error(response.metadata.error || 'Risk analysis failed');
    }

    return response.metadata.risk_results;
  }

  /**
   * Gets the bridge metrics
   */
  public async getMetrics(): Promise<any> {
    const response = await this.sendMessage(MessageType.METRICS, '');

    if (response.type === MessageType.ERROR) {
      throw new Error(response.metadata.error || 'Failed to get metrics');
    }

    return response.metadata.metrics;
  }

  /**
   * Detects the current platform
   */
  private detectPlatform(): string {
    if (typeof window !== 'undefined') {
      const userAgent = window.navigator.userAgent;
      
      if (/Android/i.test(userAgent)) {
        return 'android';
      }
      
      if (/iPhone|iPad|iPod/i.test(userAgent)) {
        return 'ios';
      }
      
      if (/Windows/i.test(userAgent)) {
        return 'windows';
      }
      
      if (/Mac/i.test(userAgent)) {
        return 'macos';
      }
      
      if (/Linux/i.test(userAgent)) {
        return 'linux';
      }
    }
    
    return 'unknown';
  }

  /**
   * Generates a unique message ID
   */
  private generateMessageId(): string {
    const timestamp = Date.now();
    const counter = this.messageCounter++;
    const random = Math.floor(Math.random() * 1000);
    return `msg_${timestamp}_${counter}_${random}`;
  }
}

/**
 * Creates a new bridge client
 * @param options Client options
 */
export function createBridgeClient(options: BridgeClientOptions): BridgeClient {
  return new BridgeClient(options);
}
