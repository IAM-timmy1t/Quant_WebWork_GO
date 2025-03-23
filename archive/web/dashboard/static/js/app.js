// Dashboard Application Logic
class DashboardApp {
    constructor() {
        this.state = {
            currentSection: 'overview',
            notifications: [],
            toasts: [],
            darkMode: localStorage.getItem('darkMode') === 'true',
            settings: null,
            websocket: null,
            charts: {},
            refreshInterval: 30000, // 30 seconds
        };

        this.initializeApp();
    }

    async initializeApp() {
        try {
            await this.loadSettings();
            this.initializeTheme();
            this.initializeWebSocket();
            this.setupEventListeners();
            this.setupKeyboardNavigation();
            this.setupRefreshTimer();
            this.refreshData();
        } catch (error) {
            this.handleError('Failed to initialize application', error);
        }
    }

    async loadSettings() {
        try {
            const response = await fetch('/api/v1/dashboard/settings');
            this.state.settings = await response.json();
            this.applySettings();
        } catch (error) {
            this.handleError('Failed to load settings', error);
        }
    }

    applySettings() {
        if (this.state.settings) {
            this.state.refreshInterval = this.state.settings.refreshInterval || 30000;
            document.documentElement.style.setProperty('--primary-color', this.state.settings.theme.primaryColor);
            document.documentElement.style.setProperty('--secondary-color', this.state.settings.theme.secondaryColor);
        }
    }

    initializeTheme() {
        if (this.state.darkMode) {
            document.documentElement.classList.add('dark');
        }
        
        // Watch for system theme changes
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
            this.toggleDarkMode(e.matches);
        });
    }

    toggleDarkMode(enabled) {
        this.state.darkMode = enabled;
        localStorage.setItem('darkMode', enabled);
        document.documentElement.classList.toggle('dark', enabled);
        this.refreshCharts(); // Update chart themes
    }

    setupEventListeners() {
        // Theme toggle
        document.getElementById('themeToggle')?.addEventListener('click', () => {
            this.toggleDarkMode(!this.state.darkMode);
        });

        // Refresh button with loading state
        document.getElementById('refreshBtn')?.addEventListener('click', async (e) => {
            const button = e.currentTarget;
            button.disabled = true;
            button.classList.add('loading');
            await this.refreshData();
            button.disabled = false;
            button.classList.remove('loading');
        });

        // Service management
        document.getElementById('addServiceBtn')?.addEventListener('click', () => {
            this.showServiceModal();
        });

        // Form submissions with validation
        document.querySelectorAll('form').forEach(form => {
            form.addEventListener('submit', async (e) => {
                e.preventDefault();
                if (this.validateForm(form)) {
                    await this.handleFormSubmit(form);
                }
            });
        });

        // Real-time search
        document.querySelectorAll('.search-input').forEach(input => {
            input.addEventListener('input', this.debounce((e) => {
                this.handleSearch(e.target.value, e.target.dataset.target);
            }, 300));
        });
    }

    setupKeyboardNavigation() {
        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + / for help modal
            if ((e.ctrlKey || e.metaKey) && e.key === '/') {
                e.preventDefault();
                this.showHelpModal();
            }
            
            // Ctrl/Cmd + D for dark mode toggle
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                this.toggleDarkMode(!this.state.darkMode);
            }
        });

        // Focus trap for modals
        document.querySelectorAll('.modal').forEach(modal => {
            this.setupFocusTrap(modal);
        });
    }

    setupFocusTrap(modal) {
        const focusableElements = modal.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        
        const firstFocusable = focusableElements[0];
        const lastFocusable = focusableElements[focusableElements.length - 1];

        modal.addEventListener('keydown', (e) => {
            if (e.key === 'Tab') {
                if (e.shiftKey && document.activeElement === firstFocusable) {
                    e.preventDefault();
                    lastFocusable.focus();
                } else if (!e.shiftKey && document.activeElement === lastFocusable) {
                    e.preventDefault();
                    firstFocusable.focus();
                }
            }
        });
    }

    async handleFormSubmit(form) {
        try {
            const formData = new FormData(form);
            const data = Object.fromEntries(formData.entries());
            
            const response = await fetch(form.action, {
                method: form.method || 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            this.showToast('Success', 'Changes saved successfully', 'success');
            this.refreshData();
        } catch (error) {
            this.handleError('Form submission failed', error);
        }
    }

    validateForm(form) {
        let isValid = true;
        const inputs = form.querySelectorAll('input, select, textarea');
        
        inputs.forEach(input => {
            if (input.hasAttribute('required') && !input.value) {
                this.showInputError(input, 'This field is required');
                isValid = false;
            } else if (input.type === 'email' && input.value && !this.isValidEmail(input.value)) {
                this.showInputError(input, 'Please enter a valid email address');
                isValid = false;
            }
        });

        return isValid;
    }

    showInputError(input, message) {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error-message text-red-500 text-sm mt-1';
        errorDiv.textContent = message;
        
        const existingError = input.parentNode.querySelector('.error-message');
        if (existingError) {
            existingError.remove();
        }
        
        input.parentNode.appendChild(errorDiv);
        input.classList.add('error');
        
        input.addEventListener('input', () => {
            errorDiv.remove();
            input.classList.remove('error');
        }, { once: true });
    }

    handleSearch(query, target) {
        const items = document.querySelectorAll(`#${target} .searchable-item`);
        const normalizedQuery = query.toLowerCase();

        items.forEach(item => {
            const text = item.textContent.toLowerCase();
            item.classList.toggle('hidden', !text.includes(normalizedQuery));
        });
    }

    showToast(title, message, type = 'info') {
        const toast = {
            id: Date.now(),
            title,
            message,
            type,
            visible: true
        };

        this.state.toasts.push(toast);
        this.updateToasts();

        setTimeout(() => {
            toast.visible = false;
            this.updateToasts();
        }, 5000);
    }

    updateToasts() {
        const container = document.getElementById('toasts');
        if (!container) return;

        container.innerHTML = this.state.toasts
            .filter(t => t.visible)
            .map(toast => `
                <div class="toast toast-${toast.type}" role="alert" aria-live="polite">
                    <div class="toast-header">
                        <strong>${toast.title}</strong>
                        <button onclick="app.dismissToast(${toast.id})" aria-label="Close">×</button>
                    </div>
                    <div class="toast-body">${toast.message}</div>
                </div>
            `).join('');
    }

    dismissToast(id) {
        const toast = this.state.toasts.find(t => t.id === id);
        if (toast) {
            toast.visible = false;
            this.updateToasts();
        }
    }

    handleError(context, error) {
        console.error(`${context}:`, error);
        this.showToast('Error', error.message || context, 'error');
    }

    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    isValidEmail(email) {
        return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    }

    // Switch between dashboard sections
    switchSection(section) {
        // Update navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
            if (item.dataset.section === section) {
                item.classList.add('active');
            }
        });

        // Update section visibility
        document.querySelectorAll('.section').forEach(sec => {
            sec.classList.add('hidden');
        });
        document.getElementById(section)?.classList.remove('hidden');

        // Update section title
        document.getElementById('section-title').textContent = 
            section.charAt(0).toUpperCase() + section.slice(1);

        this.state.currentSection = section;
        this.refreshData();
    }

    // Refresh dashboard data
    async refreshData() {
        switch (this.state.currentSection) {
            case 'overview':
                await this.refreshOverview();
                break;
            case 'services':
                await this.refreshServices();
                break;
            case 'proxy':
                await this.refreshProxyRoutes();
                break;
            case 'metrics':
                await this.refreshMetrics();
                break;
            case 'logs':
                await this.refreshLogs();
                break;
            case 'config':
                await this.refreshConfig();
                break;
        }
    }

    // Refresh overview section
    async refreshOverview() {
        try {
            const response = await fetch('/api/v1/dashboard/overview');
            const data = await response.json();

            // Update statistics
            document.getElementById('total-services').textContent = data.totalServices;
            document.getElementById('health-status').textContent = `${data.healthStatus}%`;
            document.getElementById('active-routes').textContent = data.activeRoutes;
            document.getElementById('avg-response').textContent = `${data.avgResponseTime}ms`;

            // Update charts
            this.updateHealthChart(data.healthHistory);
            this.updateServiceChart(data.serviceStatus);
        } catch (error) {
            this.handleError('Error refreshing overview data', error);
        }
    }

    // Refresh services section
    async refreshServices() {
        try {
            const response = await fetch('/api/v1/dashboard/services');
            const services = await response.json();

            const tbody = document.querySelector('#services-table tbody');
            tbody.innerHTML = '';

            services.forEach(service => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${service.name}</td>
                    <td><span class="status-${service.status.toLowerCase()}">${service.status}</span></td>
                    <td>${service.endpoints.join(', ')}</td>
                    <td>${moment(service.lastUpdated).fromNow()}</td>
                    <td>
                        <button class="btn-secondary btn-sm" onclick="app.editService('${service.id}')">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn-danger btn-sm" onclick="app.deleteService('${service.id}')">
                            <i class="fas fa-trash"></i>
                        </button>
                    </td>
                `;
                tbody.appendChild(row);
            });
        } catch (error) {
            this.handleError('Error refreshing services data', error);
        }
    }

    // Refresh proxy routes section
    async refreshProxyRoutes() {
        try {
            const response = await fetch('/api/v1/dashboard/routes');
            const routes = await response.json();

            const tbody = document.querySelector('#routes-table tbody');
            tbody.innerHTML = '';

            routes.forEach(route => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${route.path}</td>
                    <td>${route.targets.join(', ')}</td>
                    <td>${route.loadBalancing ? 'Enabled' : 'Disabled'}</td>
                    <td><span class="status-${route.status.toLowerCase()}">${route.status}</span></td>
                    <td>
                        <button class="btn-secondary btn-sm" onclick="app.editRoute('${route.id}')">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn-danger btn-sm" onclick="app.deleteRoute('${route.id}')">
                            <i class="fas fa-trash"></i>
                        </button>
                    </td>
                `;
                tbody.appendChild(row);
            });
        } catch (error) {
            this.handleError('Error refreshing proxy routes data', error);
        }
    }

    // Refresh metrics section
    async refreshMetrics() {
        try {
            const response = await fetch('/api/v1/dashboard/metrics');
            const data = await response.json();

            // Update charts
            this.updateResponseTimeChart(data.responseTimes);
            this.updateRequestVolumeChart(data.requestVolumes);

            // Update metrics table
            const tbody = document.querySelector('#metrics-table tbody');
            tbody.innerHTML = '';

            data.serviceMetrics.forEach(metric => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${metric.service}</td>
                    <td>${metric.requests}</td>
                    <td>${metric.errors}</td>
                    <td>${metric.avgResponse}ms</td>
                    <td>${metric.uptime}%</td>
                `;
                tbody.appendChild(row);
            });
        } catch (error) {
            this.handleError('Error refreshing metrics data', error);
        }
    }

    // Refresh logs section
    async refreshLogs() {
        try {
            const response = await fetch('/api/v1/dashboard/logs');
            const logs = await response.json();

            const container = document.getElementById('log-container');
            container.innerHTML = '';

            logs.forEach(log => {
                const entry = document.createElement('div');
                entry.className = `log-entry log-${log.level.toLowerCase()}`;
                entry.textContent = `[${log.timestamp}] ${log.message}`;
                container.appendChild(entry);
            });
        } catch (error) {
            this.handleError('Error refreshing logs data', error);
        }
    }

    // Refresh configuration section
    async refreshConfig() {
        try {
            const response = await fetch('/api/v1/dashboard/config');
            const config = await response.json();

            // Update proxy config form
            const proxyForm = document.getElementById('proxyConfigForm');
            proxyForm.querySelector('[name="loadBalancing"]').value = config.proxy.loadBalancing;
            proxyForm.querySelector('[name="healthCheckInterval"]').value = config.proxy.healthCheckInterval;

            // Update metrics config form
            const metricsForm = document.getElementById('metricsConfigForm');
            metricsForm.querySelector('[name="retentionPeriod"]').value = config.metrics.retentionPeriod;
            metricsForm.querySelector('[name="sampleInterval"]').value = config.metrics.sampleInterval;
        } catch (error) {
            this.handleError('Error refreshing configuration data', error);
        }
    }

    // Setup automatic refresh timer
    setupRefreshTimer() {
        setInterval(() => {
            this.refreshData();
        }, this.state.refreshInterval); // Refresh every 30 seconds
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new DashboardApp();
});
