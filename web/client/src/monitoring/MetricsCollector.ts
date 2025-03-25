/**
 * Frontend Metrics Collector
 * 
 * Collects and exposes metrics from the React frontend for Prometheus
 * Uses the Prometheus client format to expose metrics at /metrics endpoint
 */

// Singleton class to collect and expose metrics
class MetricsCollector {
  private static instance: MetricsCollector;
  private metrics: Record<string, any> = {};
  private browserPerformanceSupported: boolean;
  private metricsEndpointPath: string = '/metrics';
  private initialized: boolean = false;

  // Connection metrics
  private connectionAttempts: number = 0;
  private connectionSuccesses: number = 0;
  private connectionFailures: number = 0;
  private lastConnectionTime: number = 0;
  private messagesSent: number = 0;
  private messagesReceived: number = 0;
  private messageErrors: number = 0;
  private messageTypes: Record<string, number> = {};
  private processingTimes: number[] = [];

  // Performance metrics
  private renderTimes: Record<string, number[]> = {};
  private lastRenderTimestamp: number = 0;
  private memoryUsage: number[] = [];
  private jsHeapSizeLimit: number = 0;
  private totalJSHeapSize: number = 0;
  private usedJSHeapSize: number = 0;

  // UI interaction metrics
  private userInteractions: number = 0;
  private userErrors: number = 0;

  private constructor() {
    // Check if browser performance API is supported
    this.browserPerformanceSupported = typeof window !== 'undefined' && 
      window.performance !== undefined && 
      typeof window.performance.now === 'function';
    
    if (this.browserPerformanceSupported) {
      // Initialize memory metrics if supported
      if (performance && (performance as any).memory) {
        const memory = (performance as any).memory;
        this.jsHeapSizeLimit = memory.jsHeapSizeLimit;
        this.totalJSHeapSize = memory.totalJSHeapSize;
        this.usedJSHeapSize = memory.usedJSHeapSize;
      }
    }
  }

  public static getInstance(): MetricsCollector {
    if (!MetricsCollector.instance) {
      MetricsCollector.instance = new MetricsCollector();
    }
    return MetricsCollector.instance;
  }

  /**
   * Initialize the metrics collector
   * @param options Configuration options
   */
  public initialize(options: { metricsEndpointPath?: string } = {}): void {
    if (this.initialized) return;

    if (options.metricsEndpointPath) {
      this.metricsEndpointPath = options.metricsEndpointPath;
    }

    // Create metrics endpoint
    this.setupMetricsEndpoint();

    // Start periodic collection of metrics
    this.startPeriodicCollection();

    this.initialized = true;
    console.log('[CSM-INFO] MetricsCollector initialized');
  }

  /**
   * Setup metrics endpoint to expose metrics to Prometheus
   */
  private setupMetricsEndpoint(): void {
    if (typeof window === 'undefined') return;

    // Create a fake endpoint that Prometheus can scrape
    const originalFetch = window.fetch;
    window.fetch = async (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
      
      if (url.endsWith(this.metricsEndpointPath)) {
        return new Response(this.formatMetricsForPrometheus(), {
          status: 200,
          headers: {
            'Content-Type': 'text/plain; charset=utf-8',
          },
        });
      }
      
      return originalFetch(input, init);
    };
  }

  /**
   * Start periodic collection of performance metrics
   */
  private startPeriodicCollection(): void {
    setInterval(() => {
      this.collectPerformanceMetrics();
    }, 5000); // Collect every 5 seconds
  }

  /**
   * Collect current performance metrics
   */
  private collectPerformanceMetrics(): void {
    if (!this.browserPerformanceSupported) return;

    // Collect memory metrics if available
    if ((performance as any).memory) {
      const memory = (performance as any).memory;
      this.totalJSHeapSize = memory.totalJSHeapSize;
      this.usedJSHeapSize = memory.usedJSHeapSize;
      
      // Store memory usage as a percentage of heap limit
      const memoryPercentage = (this.usedJSHeapSize / this.jsHeapSizeLimit) * 100;
      this.memoryUsage.push(memoryPercentage);
      
      // Keep only last 100 measurements
      if (this.memoryUsage.length > 100) {
        this.memoryUsage.shift();
      }
    }

    // Collect navigation timing metrics if available
    if (performance && performance.getEntriesByType) {
      const navigationEntries = performance.getEntriesByType('navigation');
      if (navigationEntries && navigationEntries.length > 0) {
        const navigationTiming = navigationEntries[0] as PerformanceNavigationTiming;
        this.metrics.domInteractive = navigationTiming.domInteractive;
        this.metrics.domComplete = navigationTiming.domComplete;
        this.metrics.loadEventEnd = navigationTiming.loadEventEnd;
      }
    }
  }

  /**
   * Record connection attempt
   */
  public recordConnectionAttempt(): void {
    this.connectionAttempts++;
  }

  /**
   * Record successful connection
   */
  public recordConnectionSuccess(): void {
    this.connectionSuccesses++;
    this.lastConnectionTime = Date.now();
  }

  /**
   * Record connection failure
   * @param error Error object or message
   */
  public recordConnectionFailure(error: Error | string): void {
    this.connectionFailures++;
    this.recordError('connection', error instanceof Error ? error.message : error);
  }

  /**
   * Record a message being sent
   * @param type Message type
   */
  public recordMessageSent(type: string): void {
    this.messagesSent++;
    this.messageTypes[type] = (this.messageTypes[type] || 0) + 1;
  }

  /**
   * Record a message being received
   * @param type Message type
   */
  public recordMessageReceived(type: string): void {
    this.messagesReceived++;
    this.messageTypes[type] = (this.messageTypes[type] || 0) + 1;
  }

  /**
   * Record processing time for a message
   * @param processingTime Time in milliseconds
   */
  public recordProcessingTime(processingTime: number): void {
    this.processingTimes.push(processingTime);
    
    // Keep only last 100 measurements
    if (this.processingTimes.length > 100) {
      this.processingTimes.shift();
    }
  }

  /**
   * Record an error
   * @param category Error category
   * @param message Error message
   */
  public recordError(category: string, message: string): void {
    const errorKey = `error_${category}`;
    this.metrics[errorKey] = this.metrics[errorKey] || {};
    this.metrics[errorKey][message] = (this.metrics[errorKey][message] || 0) + 1;
    this.messageErrors++;
  }

  /**
   * Record component render time
   * @param componentName Name of the component
   * @param renderTime Time in milliseconds
   */
  public recordRenderTime(componentName: string, renderTime: number): void {
    this.renderTimes[componentName] = this.renderTimes[componentName] || [];
    this.renderTimes[componentName].push(renderTime);
    
    // Keep only last 20 measurements per component
    if (this.renderTimes[componentName].length > 20) {
      this.renderTimes[componentName].shift();
    }
    
    this.lastRenderTimestamp = Date.now();
  }

  /**
   * Record user interaction
   * @param interactionType Type of interaction (e.g., 'click', 'input')
   */
  public recordUserInteraction(interactionType: string): void {
    this.userInteractions++;
    const key = `interaction_${interactionType}`;
    this.metrics[key] = (this.metrics[key] || 0) + 1;
  }

  /**
   * Record user error (e.g., validation error)
   * @param errorType Type of error
   */
  public recordUserError(errorType: string): void {
    this.userErrors++;
    const key = `user_error_${errorType}`;
    this.metrics[key] = (this.metrics[key] || 0) + 1;
  }

  /**
   * Calculate average processing time
   */
  private getAverageProcessingTime(): number {
    if (this.processingTimes.length === 0) return 0;
    const sum = this.processingTimes.reduce((acc, val) => acc + val, 0);
    return sum / this.processingTimes.length;
  }

  /**
   * Calculate average render time for a component
   * @param componentName Name of the component
   */
  private getAverageRenderTime(componentName: string): number {
    const times = this.renderTimes[componentName];
    if (!times || times.length === 0) return 0;
    const sum = times.reduce((acc, val) => acc + val, 0);
    return sum / times.length;
  }

  /**
   * Format metrics in Prometheus exposition format
   */
  private formatMetricsForPrometheus(): string {
    let output = '';

    // Connection metrics
    output += '# HELP bridge_connection_attempts_total Total number of connection attempts\n';
    output += '# TYPE bridge_connection_attempts_total counter\n';
    output += `bridge_connection_attempts_total ${this.connectionAttempts}\n`;
    
    output += '# HELP bridge_connection_successes_total Total number of successful connections\n';
    output += '# TYPE bridge_connection_successes_total counter\n';
    output += `bridge_connection_successes_total ${this.connectionSuccesses}\n`;
    
    output += '# HELP bridge_connection_failures_total Total number of connection failures\n';
    output += '# TYPE bridge_connection_failures_total counter\n';
    output += `bridge_connection_failures_total ${this.connectionFailures}\n`;
    
    output += '# HELP bridge_messages_sent_total Total number of messages sent\n';
    output += '# TYPE bridge_messages_sent_total counter\n';
    output += `bridge_messages_sent_total ${this.messagesSent}\n`;
    
    output += '# HELP bridge_messages_received_total Total number of messages received\n';
    output += '# TYPE bridge_messages_received_total counter\n';
    output += `bridge_messages_received_total ${this.messagesReceived}\n`;
    
    output += '# HELP bridge_message_errors_total Total number of message errors\n';
    output += '# TYPE bridge_message_errors_total counter\n';
    output += `bridge_message_errors_total ${this.messageErrors}\n`;
    
    // Message type metrics
    output += '# HELP bridge_message_types_total Total number of messages by type\n';
    output += '# TYPE bridge_message_types_total counter\n';
    for (const [type, count] of Object.entries(this.messageTypes)) {
      output += `bridge_message_types_total{type="${type}"} ${count}\n`;
    }
    
    // Processing time
    output += '# HELP bridge_message_processing_time_average Average message processing time in milliseconds\n';
    output += '# TYPE bridge_message_processing_time_average gauge\n';
    output += `bridge_message_processing_time_average ${this.getAverageProcessingTime()}\n`;
    
    // Memory metrics
    output += '# HELP react_memory_usage_percent JS heap usage as percentage of limit\n';
    output += '# TYPE react_memory_usage_percent gauge\n';
    output += `react_memory_usage_percent ${this.memoryUsage[this.memoryUsage.length - 1] || 0}\n`;
    
    output += '# HELP react_memory_used_bytes JS heap used size in bytes\n';
    output += '# TYPE react_memory_used_bytes gauge\n';
    output += `react_memory_used_bytes ${this.usedJSHeapSize}\n`;
    
    output += '# HELP react_memory_total_bytes JS heap total size in bytes\n';
    output += '# TYPE react_memory_total_bytes gauge\n';
    output += `react_memory_total_bytes ${this.totalJSHeapSize}\n`;
    
    // Render time metrics
    output += '# HELP react_render_time_average Average render time by component in milliseconds\n';
    output += '# TYPE react_render_time_average gauge\n';
    for (const componentName of Object.keys(this.renderTimes)) {
      const avgTime = this.getAverageRenderTime(componentName);
      output += `react_render_time_average{component="${componentName}"} ${avgTime}\n`;
    }
    
    // User interaction metrics
    output += '# HELP react_user_interactions_total Total number of user interactions\n';
    output += '# TYPE react_user_interactions_total counter\n';
    output += `react_user_interactions_total ${this.userInteractions}\n`;
    
    output += '# HELP react_user_errors_total Total number of user errors\n';
    output += '# TYPE react_user_errors_total counter\n';
    output += `react_user_errors_total ${this.userErrors}\n`;
    
    return output;
  }
}

export default MetricsCollector.getInstance();
