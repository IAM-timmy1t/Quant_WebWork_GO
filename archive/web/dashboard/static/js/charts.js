// Dashboard Charts Management
class DashboardCharts {
    constructor() {
        this.charts = {};
        this.initializeCharts();
    }

    // Initialize all dashboard charts
    initializeCharts() {
        this.initializeHealthChart();
        this.initializeServiceChart();
        this.initializeResponseChart();
        this.initializeRequestChart();
    }

    // Initialize system health chart
    initializeHealthChart() {
        const ctx = document.getElementById('healthChart').getContext('2d');
        this.charts.health = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'System Health',
                    data: [],
                    borderColor: '#10b981',
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100,
                        title: {
                            display: true,
                            text: 'Health %'
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    }
                }
            }
        });
    }

    // Initialize service status chart
    initializeServiceChart() {
        const ctx = document.getElementById('serviceChart').getContext('2d');
        this.charts.service = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Healthy', 'Warning', 'Error'],
                datasets: [{
                    data: [0, 0, 0],
                    backgroundColor: [
                        '#10b981', // Success
                        '#f59e0b', // Warning
                        '#ef4444'  // Error
                    ]
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'right'
                    }
                }
            }
        });
    }

    // Initialize response time chart
    initializeResponseChart() {
        const ctx = document.getElementById('responseChart').getContext('2d');
        this.charts.response = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Average Response Time',
                    data: [],
                    borderColor: '#6366f1',
                    backgroundColor: 'rgba(99, 102, 241, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Response Time (ms)'
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    }
                }
            }
        });
    }

    // Initialize request volume chart
    initializeRequestChart() {
        const ctx = document.getElementById('requestChart').getContext('2d');
        this.charts.request = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Successful Requests',
                    data: [],
                    backgroundColor: '#10b981'
                }, {
                    label: 'Failed Requests',
                    data: [],
                    backgroundColor: '#ef4444'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Request Count'
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    }
                },
                plugins: {
                    legend: {
                        position: 'top'
                    }
                }
            }
        });
    }

    // Update health chart data
    updateHealthChart(data) {
        const chart = this.charts.health;
        chart.data.labels = data.map(item => moment(item.timestamp).format('HH:mm'));
        chart.data.datasets[0].data = data.map(item => item.health);
        chart.update();
    }

    // Update service status chart data
    updateServiceChart(data) {
        const chart = this.charts.service;
        chart.data.datasets[0].data = [
            data.healthy,
            data.warning,
            data.error
        ];
        chart.update();
    }

    // Update response time chart data
    updateResponseChart(data) {
        const chart = this.charts.response;
        chart.data.labels = data.map(item => moment(item.timestamp).format('HH:mm'));
        chart.data.datasets[0].data = data.map(item => item.avgResponse);
        chart.update();
    }

    // Update request volume chart data
    updateRequestChart(data) {
        const chart = this.charts.request;
        chart.data.labels = data.map(item => moment(item.timestamp).format('HH:mm'));
        chart.data.datasets[0].data = data.map(item => item.successful);
        chart.data.datasets[1].data = data.map(item => item.failed);
        chart.update();
    }

    // Update all charts
    updateAllCharts(data) {
        if (data.healthHistory) {
            this.updateHealthChart(data.healthHistory);
        }
        if (data.serviceStatus) {
            this.updateServiceChart(data.serviceStatus);
        }
        if (data.responseTimes) {
            this.updateResponseChart(data.responseTimes);
        }
        if (data.requestVolumes) {
            this.updateRequestChart(data.requestVolumes);
        }
    }
}

// Initialize charts when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.dashboardCharts = new DashboardCharts();
});
