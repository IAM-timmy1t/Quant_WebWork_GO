#!/bin/bash

# Quant WebWorks GO - Monitoring Setup Script
# This script sets up the monitoring infrastructure for the Bridge Module integration

set -e

echo "=== Quant WebWorks GO - Monitoring Setup ==="
echo "This script will set up monitoring for the Bridge Module integration"
echo

# Create required directories
echo "Creating directory structure..."
mkdir -p monitoring/prometheus
mkdir -p monitoring/grafana/provisioning/dashboards
mkdir -p monitoring/grafana/provisioning/datasources

# Verify if configuration files exist
if [ ! -f "monitoring/prometheus/prometheus.yml" ]; then
  echo "Error: Missing Prometheus configuration file."
  echo "Please ensure monitoring/prometheus/prometheus.yml exists."
  exit 1
fi

if [ ! -f "monitoring/grafana/provisioning/dashboards/bridge-dashboard.json" ]; then
  echo "Error: Missing Grafana dashboard configuration."
  echo "Please ensure monitoring/grafana/provisioning/dashboards/bridge-dashboard.json exists."
  exit 1
fi

if [ ! -f "monitoring/grafana/provisioning/datasources/prometheus.yml" ]; then
  echo "Error: Missing Grafana datasource configuration."
  echo "Please ensure monitoring/grafana/provisioning/datasources/prometheus.yml exists."
  exit 1
fi

# Create dashboard provisioning file
echo "Configuring Grafana dashboard provisioning..."
cat > monitoring/grafana/provisioning/dashboards/dashboard.yml << EOF
apiVersion: 1

providers:
- name: 'default'
  orgId: 1
  folder: ''
  type: file
  disableDeletion: false
  editable: true
  options:
    path: /etc/grafana/provisioning/dashboards
EOF

echo "Setting appropriate permissions..."
chmod -R 755 monitoring/

echo "Checking Docker and Docker Compose installation..."
if ! command -v docker &> /dev/null; then
  echo "Docker is not installed. Please install Docker and try again."
  exit 1
fi

if ! command -v docker-compose &> /dev/null; then
  echo "Docker Compose is not installed. Please install Docker Compose and try again."
  exit 1
fi

echo
echo "=== Setup Complete ==="
echo "You can now start the monitoring infrastructure with:"
echo "docker-compose up -d prometheus grafana node-exporter"
echo
echo "Prometheus will be available at: http://localhost:9091"
echo "Grafana will be available at: http://localhost:3001"
echo "Default Grafana credentials: admin / admin"
echo
echo "To start the complete application with monitoring:"
echo "docker-compose up -d"
echo
