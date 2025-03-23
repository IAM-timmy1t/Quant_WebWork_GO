// Metrics visualization and management
class MetricsManager {
    constructor() {
        this.charts = {};
        this.lastUpdate = new Date();
        this.ws = null;
        this.retryInterval = 5000;
        this.filters = ['system_metrics', 'service_metrics', 'route_metrics', 'logs'];
        
        // Initialize charts
        this.initializeCharts();
        
        // Connect WebSocket
        this.connectWebSocket();
        
        // Initialize event listeners
        this.initializeEventListeners();
    }

    initializeCharts() {
        // System metrics chart
        this.charts.system = new Chart(document.getElementById('systemMetrics'), {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'CPU Usage',
                    borderColor: '#4CAF50',
                    data: []
                }, {
                    label: 'Memory Usage',
                    borderColor: '#2196F3',
                    data: []
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            unit: 'minute'
                        }
                    },
                    y: {
                        beginAtZero: true,
                        max: 100
                    }
                }
            }
        });

        // Service health chart
        this.charts.services = new Chart(document.getElementById('serviceHealth'), {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Health Score',
                    backgroundColor: '#4CAF50',
                    data: []
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100
                    }
                }
            }
        });

        // Request rate chart
        this.charts.requests = new Chart(document.getElementById('requestRate'), {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Requests/sec',
                    borderColor: '#FF9800',
                    data: []
                }, {
                    label: 'Errors/sec',
                    borderColor: '#F44336',
                    data: []
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            unit: 'minute'
                        }
                    }
                }
            }
        });
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/dashboard/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.setFilters(this.filters);
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected, retrying...');
            setTimeout(() => this.connectWebSocket(), this.retryInterval);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    setFilters(filters) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                action: 'set_filters',
                filters: filters
            }));
        }
    }

    handleMessage(message) {
        switch (message.type) {
            case 'system_metrics':
                this.updateSystemMetrics(message.payload);
                break;
            case 'service_metrics':
                this.updateServiceMetrics(message.payload);
                break;
            case 'route_metrics':
                this.updateRouteMetrics(message.payload);
                break;
            case 'logs':
                this.updateLogs(message.payload);
                break;
        }
    }

    updateSystemMetrics(metrics) {
        const chart = this.charts.system;
        const timestamp = new Date(metrics.timestamp);

        // Update CPU and Memory datasets
        chart.data.labels.push(timestamp);
        chart.data.datasets[0].data.push(metrics.cpuUsage);
        chart.data.datasets[1].data.push(metrics.memoryUsage);

        // Keep last 60 data points
        if (chart.data.labels.length > 60) {
            chart.data.labels.shift();
            chart.data.datasets.forEach(dataset => dataset.data.shift());
        }

        chart.update();

        // Update summary statistics
        document.getElementById('cpuUsage').textContent = `${metrics.cpuUsage.toFixed(1)}%`;
        document.getElementById('memoryUsage').textContent = `${metrics.memoryUsage.toFixed(1)}%`;
        document.getElementById('totalRequests').textContent = metrics.totalRequests;
        document.getElementById('errorRate').textContent = `${(metrics.errorRate * 100).toFixed(2)}%`;
    }

    updateServiceMetrics(metrics) {
        const chart = this.charts.services;
        
        // Update service health chart
        const serviceIndex = chart.data.labels.indexOf(metrics.serviceName);
        if (serviceIndex === -1) {
            chart.data.labels.push(metrics.serviceName);
            chart.data.datasets[0].data.push(this.calculateHealthScore(metrics));
        } else {
            chart.data.datasets[0].data[serviceIndex] = this.calculateHealthScore(metrics);
        }
        
        chart.update();

        // Update service status table
        this.updateServiceTable(metrics);
    }

    calculateHealthScore(metrics) {
        const availability = metrics.availableTime / (metrics.availableTime + metrics.downTime);
        const errorRate = metrics.requestCount > 0 ? metrics.errorCount / metrics.requestCount : 0;
        const responseTime = metrics.responseTimes.length > 0 
            ? metrics.responseTimes.reduce((a, b) => a + b) / metrics.responseTimes.length 
            : 0;

        // Weight factors for health score calculation
        const weights = {
            availability: 0.5,
            errorRate: 0.3,
            responseTime: 0.2
        };

        return (
            (availability * weights.availability) +
            ((1 - errorRate) * weights.errorRate) +
            (Math.max(0, 1 - (responseTime / 1000)) * weights.responseTime)
        ) * 100;
    }

    updateServiceTable(metrics) {
        const tableBody = document.getElementById('serviceTableBody');
        let row = tableBody.querySelector(`tr[data-service-id="${metrics.serviceId}"]`);
        
        if (!row) {
            row = document.createElement('tr');
            row.setAttribute('data-service-id', metrics.serviceId);
            tableBody.appendChild(row);
        }

        row.innerHTML = `
            <td>${metrics.serviceName}</td>
            <td class="status-${metrics.status.toLowerCase()}">${metrics.status}</td>
            <td>${metrics.requestCount}</td>
            <td>${metrics.errorCount}</td>
            <td>${this.formatDuration(metrics.availableTime)}</td>
            <td>${metrics.lastErrorMsg || 'N/A'}</td>
        `;
    }

    updateRouteMetrics(metrics) {
        const chart = this.charts.requests;
        const timestamp = new Date(metrics.lastAccessed);

        // Update request rate chart
        chart.data.labels.push(timestamp);
        chart.data.datasets[0].data.push(metrics.requestCount);
        chart.data.datasets[1].data.push(metrics.errorCount);

        // Keep last 60 data points
        if (chart.data.labels.length > 60) {
            chart.data.labels.shift();
            chart.data.datasets.forEach(dataset => dataset.data.shift());
        }

        chart.update();
    }

    updateLogs(logs) {
        const logContainer = document.getElementById('logContainer');
        
        logs.forEach(log => {
            const logEntry = document.createElement('div');
            logEntry.className = `log-entry log-${log.level.toLowerCase()}`;
            logEntry.innerHTML = `
                <span class="log-timestamp">${new Date(log.timestamp).toLocaleTimeString()}</span>
                <span class="log-level">[${log.level}]</span>
                <span class="log-source">${log.source}:</span>
                <span class="log-message">${log.message}</span>
            `;
            logContainer.appendChild(logEntry);
        });

        // Keep only last 100 log entries
        while (logContainer.children.length > 100) {
            logContainer.removeChild(logContainer.firstChild);
        }

        // Auto-scroll to bottom
        logContainer.scrollTop = logContainer.scrollHeight;
    }

    formatDuration(ms) {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);

        if (days > 0) return `${days}d ${hours % 24}h`;
        if (hours > 0) return `${hours}h ${minutes % 60}m`;
        if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
        return `${seconds}s`;
    }

    initializeEventListeners() {
        // Add event listeners for UI controls
        document.getElementById('refreshRate').addEventListener('change', (e) => {
            const rate = parseInt(e.target.value);
            // Update chart refresh rates
            Object.values(this.charts).forEach(chart => {
                chart.options.animation.duration = rate;
            });
        });

        // Add filter controls
        document.getElementById('metricFilters').addEventListener('change', (e) => {
            const checkbox = e.target;
            if (checkbox.type === 'checkbox') {
                const filter = checkbox.value;
                if (checkbox.checked) {
                    this.filters.push(filter);
                } else {
                    const index = this.filters.indexOf(filter);
                    if (index > -1) this.filters.splice(index, 1);
                }
                this.setFilters(this.filters);
            }
        });
    }
}
