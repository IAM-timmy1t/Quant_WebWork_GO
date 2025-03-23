# Bridge Module Monitoring Infrastructure

This document provides information about the monitoring setup for the Bridge Module integration between the React frontend and Go backend.

## Overview

The monitoring infrastructure consists of:

1. **Prometheus** - For metrics collection and storage
2. **Grafana** - For metrics visualization and dashboarding
3. **Node Exporter** - For host-level metrics
4. **Frontend Metrics Collector** - React component for collecting frontend metrics
5. **Backend Metrics Integration** - Prometheus integration in the Go backend

## Architecture

```
┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │
│  React Frontend │    │   Go Backend    │
│  (Metrics API)  │    │  (Metrics API)  │
│                 │    │                 │
└────────┬────────┘    └────────┬────────┘
         │                      │
         │                      │
         ▼                      ▼
┌─────────────────────────────────────────┐
│                                         │
│              Prometheus                 │
│           (Metrics Storage)             │
│                                         │
└────────────────────┬────────────────────┘
                     │
                     │
                     ▼
┌─────────────────────────────────────────┐
│                                         │
│                Grafana                  │
│           (Metrics Visualization)       │
│                                         │
└─────────────────────────────────────────┘
```

## Metrics Collected

### Frontend Metrics
- Connection attempts, successes, and failures
- Messages sent and received by type
- Message processing times
- UI rendering performance
- Memory usage
- User interactions and errors

### Backend Metrics
- Active connections
- Messages processed (by type)
- Message processing times
- Error rates and types
- Token analysis performance
- Resource usage (memory, CPU)
- Server uptime

## Setup Instructions

1. Ensure all configuration files are in place:
   - `monitoring/prometheus/prometheus.yml`
   - `monitoring/grafana/provisioning/dashboards/bridge-dashboard.json`
   - `monitoring/grafana/provisioning/datasources/prometheus.yml`

2. Run the setup script:
   ```
   chmod +x setup_monitoring.sh
   ./setup_monitoring.sh
   ```

3. Start the monitoring infrastructure:
   ```
   docker-compose up -d prometheus grafana node-exporter
   ```

4. Or start the complete application with monitoring:
   ```
   docker-compose up -d
   ```

## Accessing the Monitoring UIs

- **Prometheus**: http://localhost:9091
- **Grafana**: http://localhost:3001
  - Default credentials: admin / admin

## Bridge Dashboard

The preconfigured Bridge Dashboard in Grafana includes:

1. **Connection Metrics**
   - Active connections
   - Connection success/failure rates

2. **Message Metrics**
   - Messages per second
   - Message types distribution
   - Total messages processed

3. **Performance Metrics**
   - Average processing time
   - React UI rendering times
   - Memory usage

4. **Error Tracking**
   - Error count
   - Error types distribution
   - Recent errors table

## Custom Metrics Integration

### For React Components

To add metrics tracking to a React component:

```tsx
import metricsCollector from '../monitoring/MetricsCollector';

// In your component
useEffect(() => {
  // Record component render time
  const startTime = performance.now();
  
  return () => {
    const endTime = performance.now();
    metricsCollector.recordRenderTime('YourComponent', endTime - startTime);
  };
}, [dependencies]);

// Record user interactions
const handleClick = () => {
  metricsCollector.recordUserInteraction('button_click');
  // Rest of your handler
};

// Record errors
try {
  // Your code
} catch (err) {
  metricsCollector.recordError('operation_type', err.message);
}
```

### For Go Backend

To add metrics to Go backend code:

```go
// Initialize metrics
bridgeMetrics := metrics.NewBridgeMetrics()

// Record a message received
bridgeMetrics.RecordMessageReceived("query")

// Measure processing time
startTime := time.Now()
// Process message...
bridgeMetrics.ObserveMessageProcessingTime("query", time.Since(startTime))

// Record connection events
bridgeMetrics.RecordConnectionOpened()
bridgeMetrics.RecordConnectionClosed()

// Record errors
bridgeMetrics.RecordMessageError("validation_error")

// Measure token analysis with automatic completion
done := bridgeMetrics.StartTokenAnalysis()
// Analyze token...
done(err) // Automatically records completion time and errors
```

## Troubleshooting

### Common Issues

1. **Metrics not appearing in Grafana**
   - Check Prometheus is running: `docker-compose ps prometheus`
   - Verify targets are up in Prometheus UI: http://localhost:9091/targets
   - Check Prometheus datasource is configured in Grafana

2. **Dashboard not loading**
   - Verify dashboard provisioning file exists
   - Check Grafana logs: `docker-compose logs grafana`

3. **Frontend metrics not being collected**
   - Ensure the MetricsCollector is initialized in your React app
   - Check browser console for any errors

## Extending the Monitoring

### Adding New Metrics

1. **Frontend**: Add new metrics in `client/src/monitoring/MetricsCollector.ts`
2. **Backend**: Add new metrics in `internal/core/metrics/bridge_metrics.go`

### Creating Custom Dashboards

1. Export a dashboard from Grafana UI
2. Save it to `monitoring/grafana/provisioning/dashboards/`
3. Update the dashboard provisioning configuration

## Security Considerations

- Grafana is configured with default credentials (admin/admin) - change these in production
- Prometheus and Grafana endpoints should not be publicly exposed in production
- Consider implementing authentication for metrics endpoints in production
