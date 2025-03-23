// Dashboard WebSocket Connection
let ws = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;

// Charts
let resourcesChart = null;
let networkChart = null;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    initializeCharts();
    connectWebSocket();
});

// Initialize Chart.js charts
function initializeCharts() {
    // System Resources Chart
    const resourcesCtx = document.getElementById('resourcesChart').getContext('2d');
    resourcesChart = new Chart(resourcesCtx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'CPU Usage',
                    borderColor: 'rgb(75, 192, 192)',
                    data: [],
                    fill: false,
                },
                {
                    label: 'Memory Usage',
                    borderColor: 'rgb(255, 99, 132)',
                    data: [],
                    fill: false,
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    display: true,
                    title: {
                        display: true,
                        text: 'Time'
                    }
                },
                y: {
                    display: true,
                    title: {
                        display: true,
                        text: 'Usage %'
                    },
                    min: 0,
                    max: 100
                }
            }
        }
    });

    // Network Traffic Chart
    const networkCtx = document.getElementById('networkChart').getContext('2d');
    networkChart = new Chart(networkCtx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'Bytes Sent',
                    borderColor: 'rgb(54, 162, 235)',
                    data: [],
                    fill: false,
                },
                {
                    label: 'Bytes Received',
                    borderColor: 'rgb(255, 159, 64)',
                    data: [],
                    fill: false,
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    display: true,
                    title: {
                        display: true,
                        text: 'Time'
                    }
                },
                y: {
                    display: true,
                    title: {
                        display: true,
                        text: 'Bytes/s'
                    }
                }
            }
        }
    });
}

// Connect to WebSocket server
function connectWebSocket() {
    if (ws !== null) {
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/dashboard`;

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('Connected to dashboard WebSocket');
        reconnectAttempts = 0;
    };

    ws.onclose = () => {
        console.log('Dashboard WebSocket connection closed');
        ws = null;

        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            setTimeout(connectWebSocket, 1000 * Math.pow(2, reconnectAttempts));
        }
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };

    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
    };
}

// Handle incoming WebSocket messages
function handleWebSocketMessage(message) {
    switch (message.type) {
        case 'resource_metrics':
            updateResourceMetrics(message.payload);
            break;
        case 'security_event':
            handleSecurityEvent(message.payload);
            break;
        default:
            console.warn('Unknown message type:', message.type);
    }
}

// Update resource metrics and charts
function updateResourceMetrics(metrics) {
    // Update overview cards
    document.getElementById('cpuUsage').textContent = `${metrics.cpu.usagePercent.toFixed(1)}%`;
    document.getElementById('memoryUsage').textContent = `${metrics.memory.usagePercent.toFixed(1)}%`;

    // Calculate network I/O rate
    let totalBytesPerSec = 0;
    metrics.network.forEach(net => {
        totalBytesPerSec += (net.bytesSent + net.bytesRecv);
    });
    document.getElementById('networkIO').textContent = `${formatBytes(totalBytesPerSec)}/s`;

    // Update charts
    const timestamp = new Date(metrics.timestamp).toLocaleTimeString();

    // Update resources chart
    if (resourcesChart.data.labels.length > 20) {
        resourcesChart.data.labels.shift();
        resourcesChart.data.datasets[0].data.shift();
        resourcesChart.data.datasets[1].data.shift();
    }

    resourcesChart.data.labels.push(timestamp);
    resourcesChart.data.datasets[0].data.push(metrics.cpu.usagePercent);
    resourcesChart.data.datasets[1].data.push(metrics.memory.usagePercent);
    resourcesChart.update();

    // Update network chart
    if (networkChart.data.labels.length > 20) {
        networkChart.data.labels.shift();
        networkChart.data.datasets[0].data.shift();
        networkChart.data.datasets[1].data.shift();
    }

    let totalBytesSent = 0;
    let totalBytesRecv = 0;
    metrics.network.forEach(net => {
        totalBytesSent += net.bytesSent;
        totalBytesRecv += net.bytesRecv;
    });

    networkChart.data.labels.push(timestamp);
    networkChart.data.datasets[0].data.push(totalBytesSent);
    networkChart.data.datasets[1].data.push(totalBytesRecv);
    networkChart.update();
}

// Handle security events
function handleSecurityEvent(event) {
    const tbody = document.getElementById('securityEvents');
    const row = document.createElement('tr');

    // Add severity class
    row.className = getSeverityClass(event.severity);

    // Create cells
    const timestamp = new Date(event.timestamp).toLocaleString();
    row.innerHTML = `
        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${timestamp}</td>
        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${event.eventType}</td>
        <td class="px-6 py-4 whitespace-nowrap text-sm">${event.severity}</td>
        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${event.source}</td>
        <td class="px-6 py-4 text-sm text-gray-500">${event.description}</td>
    `;

    // Insert at the top
    tbody.insertBefore(row, tbody.firstChild);

    // Update risk score
    document.getElementById('riskScore').textContent = event.riskScore.toFixed(1);

    // Limit number of visible events
    while (tbody.children.length > 100) {
        tbody.removeChild(tbody.lastChild);
    }
}

// Helper function to format bytes
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}

// Get severity class for styling
function getSeverityClass(severity) {
    switch (severity) {
        case 'critical':
            return 'bg-red-50';
        case 'high':
            return 'bg-orange-50';
        case 'medium':
            return 'bg-yellow-50';
        case 'low':
            return 'bg-blue-50';
        default:
            return '';
    }
}
