function dashboardApp() {
    return {
        // State
        currentView: 'overview',
        showAddUserModal: false,
        securityScore: 95,
        activeServices: 12,
        wsConnection: null,
        charts: {},
        
        // Data
        services: [
            { id: 1, name: 'Web Server', status: 'online', health: 98 },
            { id: 2, name: 'Database', status: 'online', health: 95 },
            { id: 3, name: 'Cache', status: 'online', health: 100 },
            { id: 4, name: 'Message Queue', status: 'offline', health: 0 }
        ],
        
        securityEvents: [
            { 
                id: 1, 
                time: '2025-02-17 03:45:12',
                description: 'Failed login attempt',
                severity: 'medium',
                status: 'resolved'
            },
            {
                id: 2,
                time: '2025-02-17 03:30:00',
                description: 'Unusual network activity detected',
                severity: 'high',
                status: 'investigating'
            }
        ],
        
        users: [
            { 
                id: 1, 
                username: 'admin',
                role: 'Administrator',
                lastActive: '2025-02-17 03:55:00'
            },
            {
                id: 2,
                username: 'john.doe',
                role: 'User',
                lastActive: '2025-02-17 02:30:00'
            }
        ],
        
        newUser: {
            username: '',
            role: 'user'
        },

        // Lifecycle
        init() {
            this.initializeWebSocket();
            this.initializeCharts();
            this.startMetricUpdates();
        },

        // WebSocket
        initializeWebSocket() {
            this.wsConnection = new WebSocket('ws://' + window.location.host + '/ws/dashboard');
            
            this.wsConnection.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            };

            this.wsConnection.onclose = () => {
                console.log('WebSocket connection closed. Attempting to reconnect...');
                setTimeout(() => this.initializeWebSocket(), 5000);
            };
        },

        handleWebSocketMessage(data) {
            switch (data.type) {
                case 'metrics':
                    this.updateMetrics(data.payload);
                    break;
                case 'security_event':
                    this.addSecurityEvent(data.payload);
                    break;
                case 'service_status':
                    this.updateServiceStatus(data.payload);
                    break;
            }
        },

        // Charts
        initializeCharts() {
            // Resource Usage Chart
            const resourceCtx = document.getElementById('resourceChart').getContext('2d');
            this.charts.resource = new Chart(resourceCtx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'CPU Usage',
                        data: [],
                        borderColor: 'rgb(75, 192, 192)',
                        tension: 0.1
                    }, {
                        label: 'Memory Usage',
                        data: [],
                        borderColor: 'rgb(255, 99, 132)',
                        tension: 0.1
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true,
                            max: 100
                        }
                    }
                }
            });

            // Network Traffic Chart
            const networkCtx = document.getElementById('networkChart').getContext('2d');
            this.charts.network = new Chart(networkCtx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Incoming Traffic',
                        data: [],
                        borderColor: 'rgb(54, 162, 235)',
                        tension: 0.1
                    }, {
                        label: 'Outgoing Traffic',
                        data: [],
                        borderColor: 'rgb(255, 159, 64)',
                        tension: 0.1
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        },

        // Metrics Updates
        startMetricUpdates() {
            setInterval(() => {
                if (this.wsConnection && this.wsConnection.readyState === WebSocket.OPEN) {
                    this.wsConnection.send(JSON.stringify({
                        type: 'metrics_request',
                        payload: {
                            types: ['cpu', 'memory', 'network']
                        }
                    }));
                }
            }, 5000);
        },

        updateMetrics(metrics) {
            // Update charts with new data
            const timestamp = new Date().toLocaleTimeString();
            
            // Update Resource Chart
            this.charts.resource.data.labels.push(timestamp);
            this.charts.resource.data.datasets[0].data.push(metrics.cpu);
            this.charts.resource.data.datasets[1].data.push(metrics.memory);
            
            // Keep only last 10 data points
            if (this.charts.resource.data.labels.length > 10) {
                this.charts.resource.data.labels.shift();
                this.charts.resource.data.datasets.forEach(dataset => dataset.data.shift());
            }
            
            this.charts.resource.update();

            // Update Network Chart
            this.charts.network.data.labels.push(timestamp);
            this.charts.network.data.datasets[0].data.push(metrics.network.incoming);
            this.charts.network.data.datasets[1].data.push(metrics.network.outgoing);
            
            if (this.charts.network.data.labels.length > 10) {
                this.charts.network.data.labels.shift();
                this.charts.network.data.datasets.forEach(dataset => dataset.data.shift());
            }
            
            this.charts.network.update();
        },

        // User Management
        addUser() {
            // Validate input
            if (!this.newUser.username || !this.newUser.role) {
                alert('Please fill in all fields');
                return;
            }

            // Send request to backend
            fetch('/api/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(this.newUser)
            })
            .then(response => response.json())
            .then(data => {
                this.users.push({
                    id: data.id,
                    username: this.newUser.username,
                    role: this.newUser.role,
                    lastActive: 'Never'
                });
                this.showAddUserModal = false;
                this.newUser = { username: '', role: 'user' };
            })
            .catch(error => {
                console.error('Error adding user:', error);
                alert('Failed to add user');
            });
        },

        editUser(user) {
            // Implement edit user functionality
            console.log('Edit user:', user);
        },

        deleteUser(user) {
            if (!confirm(`Are you sure you want to delete user ${user.username}?`)) {
                return;
            }

            fetch(`/api/users/${user.id}`, {
                method: 'DELETE'
            })
            .then(response => {
                if (response.ok) {
                    this.users = this.users.filter(u => u.id !== user.id);
                } else {
                    throw new Error('Failed to delete user');
                }
            })
            .catch(error => {
                console.error('Error deleting user:', error);
                alert('Failed to delete user');
            });
        },

        // Service Management
        updateServiceStatus(serviceUpdate) {
            const service = this.services.find(s => s.id === serviceUpdate.id);
            if (service) {
                service.status = serviceUpdate.status;
                service.health = serviceUpdate.health;
            }
        },

        // Security Events
        addSecurityEvent(event) {
            this.securityEvents.unshift(event);
            if (this.securityEvents.length > 100) {
                this.securityEvents.pop();
            }
        }
    };
}
