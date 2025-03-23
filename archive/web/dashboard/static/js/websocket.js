// Dashboard WebSocket Management
class DashboardWebSocket {
    constructor() {
        this.socket = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.notifications = [];
        this.initialize();
    }

    // Initialize WebSocket connection
    initialize() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/dashboard/ws`;
        
        this.connect(wsUrl);
        this.setupNotifications();
    }

    // Connect to WebSocket server
    connect(url) {
        this.socket = new WebSocket(url);

        this.socket.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.updateConnectionStatus('connected');
        };

        this.socket.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus('disconnected');
            this.attemptReconnect();
        };

        this.socket.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.updateConnectionStatus('error');
        };

        this.socket.onmessage = (event) => {
            this.handleMessage(event.data);
        };
    }

    // Handle incoming WebSocket messages
    handleMessage(data) {
        try {
            const message = JSON.parse(data);
            
            switch (message.type) {
                case 'health_update':
                    this.handleHealthUpdate(message.data);
                    break;
                case 'service_update':
                    this.handleServiceUpdate(message.data);
                    break;
                case 'metrics_update':
                    this.handleMetricsUpdate(message.data);
                    break;
                case 'alert':
                    this.handleAlert(message.data);
                    break;
                case 'log':
                    this.handleLog(message.data);
                    break;
                default:
                    console.warn('Unknown message type:', message.type);
            }
        } catch (error) {
            console.error('Error handling WebSocket message:', error);
        }
    }

    // Handle health update messages
    handleHealthUpdate(data) {
        if (window.dashboardCharts) {
            // Update health chart
            window.dashboardCharts.updateHealthChart([{
                timestamp: new Date(),
                health: data.health
            }]);

            // Update health status display
            document.getElementById('health-status').textContent = `${data.health}%`;
        }

        // Show notification if health is below threshold
        if (data.health < 80) {
            this.showNotification('System Health Alert', 
                `System health has dropped to ${data.health}%`, 
                'warning');
        }
    }

    // Handle service update messages
    handleServiceUpdate(data) {
        // Update service status in table if visible
        const serviceRow = document.querySelector(`#services-table tr[data-service-id="${data.id}"]`);
        if (serviceRow) {
            const statusCell = serviceRow.querySelector('.service-status');
            if (statusCell) {
                statusCell.className = `service-status status-${data.status.toLowerCase()}`;
                statusCell.textContent = data.status;
            }
        }

        // Show notification for status changes
        if (data.status === 'Error') {
            this.showNotification('Service Alert', 
                `Service ${data.name} is reporting errors`, 
                'error');
        }
    }

    // Handle metrics update messages
    handleMetricsUpdate(data) {
        if (window.dashboardCharts) {
            if (data.responseTimes) {
                window.dashboardCharts.updateResponseChart(data.responseTimes);
            }
            if (data.requestVolumes) {
                window.dashboardCharts.updateRequestChart(data.requestVolumes);
            }
        }

        // Update metrics table if visible
        const metricsTable = document.getElementById('metrics-table');
        if (metricsTable && data.serviceMetrics) {
            this.updateMetricsTable(data.serviceMetrics);
        }
    }

    // Handle alert messages
    handleAlert(data) {
        this.showNotification(data.title, data.message, data.level || 'info');
    }

    // Handle log messages
    handleLog(data) {
        const logContainer = document.getElementById('log-container');
        if (logContainer) {
            const entry = document.createElement('div');
            entry.className = `log-entry log-${data.level.toLowerCase()}`;
            entry.textContent = `[${new Date().toISOString()}] ${data.message}`;
            
            logContainer.insertBefore(entry, logContainer.firstChild);
            
            // Limit number of log entries
            while (logContainer.children.length > 1000) {
                logContainer.removeChild(logContainer.lastChild);
            }
        }
    }

    // Update metrics table
    updateMetricsTable(metrics) {
        const tbody = document.querySelector('#metrics-table tbody');
        if (!tbody) return;

        metrics.forEach(metric => {
            const row = tbody.querySelector(`tr[data-service-id="${metric.serviceId}"]`);
            if (row) {
                row.innerHTML = `
                    <td>${metric.service}</td>
                    <td>${metric.requests}</td>
                    <td>${metric.errors}</td>
                    <td>${metric.avgResponse}ms</td>
                    <td>${metric.uptime}%</td>
                `;
            }
        });
    }

    // Show notification
    showNotification(title, message, level = 'info') {
        const notification = {
            id: Date.now(),
            title,
            message,
            level
        };

        this.notifications.push(notification);
        this.updateNotificationBadge();
        this.renderNotification(notification);
    }

    // Setup notifications
    setupNotifications() {
        const notificationBtn = document.querySelector('.btn-notification');
        if (notificationBtn) {
            notificationBtn.addEventListener('click', () => {
                this.showNotificationPanel();
            });
        }
    }

    // Update notification badge
    updateNotificationBadge() {
        const badge = document.querySelector('.notification-badge');
        if (badge) {
            badge.textContent = this.notifications.length;
            badge.style.display = this.notifications.length > 0 ? 'flex' : 'none';
        }
    }

    // Render notification
    renderNotification(notification) {
        // Create notification element
        const element = document.createElement('div');
        element.className = `notification notification-${notification.level}`;
        element.innerHTML = `
            <div class="notification-title">${notification.title}</div>
            <div class="notification-message">${notification.message}</div>
            <div class="notification-time">${moment().fromNow()}</div>
        `;

        // Add to notification panel if open
        const panel = document.querySelector('.notification-panel');
        if (panel) {
            panel.insertBefore(element, panel.firstChild);
        }
    }

    // Show notification panel
    showNotificationPanel() {
        const panel = document.querySelector('.notification-panel');
        if (panel) {
            panel.classList.toggle('hidden');
        }
    }

    // Update connection status
    updateConnectionStatus(status) {
        const statusIndicator = document.querySelector('.connection-status');
        if (statusIndicator) {
            statusIndicator.className = `connection-status status-${status}`;
            statusIndicator.title = `WebSocket: ${status}`;
        }
    }

    // Attempt to reconnect
    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
            
            setTimeout(() => {
                this.initialize();
            }, this.reconnectDelay * this.reconnectAttempts);
        } else {
            console.error('Max reconnection attempts reached');
            this.showNotification('Connection Error', 
                'Unable to connect to server. Please refresh the page.', 
                'error');
        }
    }

    // Send message to server
    send(type, data) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({
                type,
                data
            }));
        }
    }
}

// Initialize WebSocket when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.dashboardWS = new DashboardWebSocket();
});
