#!/bin/bash
# Development environment setup script for QUANT_WebWork_GO

# Check prerequisites
command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Aborting."; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed. Aborting."; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "Docker Compose is required but not installed. Aborting."; exit 1; }
command -v node >/dev/null 2>&1 || { echo "Node.js is required but not installed. Aborting."; exit 1; }

# Initialize Go module if not already initialized
if [ ! -f "go.mod" ]; then
  echo "Initializing Go module..."
  go mod init github.com/IAM-timmy1t/Quant_WebWork_GO
fi

# Install Go dependencies
echo "Installing Go dependencies..."
go mod tidy

# Install frontend dependencies
echo "Installing frontend dependencies..."
if [ -d "web/client" ]; then
  cd web/client
  npm install
  cd ../..
else
  echo "Frontend directory not found, skipping frontend dependencies."
fi

# Create config directory if it doesn't exist
mkdir -p config

# Generate default configuration if it doesn't exist
if [ ! -f "config/default.yaml" ]; then
  echo "Creating default configuration..."
  cat > config/default.yaml << EOF
server:
  host: 0.0.0.0
  port: 8080
  timeout: 30s

security:
  tls_enabled: false
  cert_file: certs/server.crt
  key_file: certs/server.key
  jwt_secret: \${QUANT_JWT_SECRET}
  token_expiry_min: 60
  rate_limiting:
    enabled: true
    default_limit: 100
    interval: 1m
  ip_masking:
    enabled: true
    rotation_interval: 1h

bridge:
  enabled: true
  default_protocol: rest
  max_concurrent_requests: 100
  request_timeout: 15s
  protocols:
    rest:
      host: localhost
      port: 8081
      timeout: 10s
      max_retries: 3
    grpc:
      host: localhost
      port: 50051
      timeout: 5s
      max_retries: 2
    websocket:
      host: localhost
      port: 8082
      timeout: 30s
      max_retries: 1

monitoring:
  metrics:
    enabled: true
    path: /metrics
    collection_interval: 15s
    batch_size: 100
  tracing:
    enabled: false
    provider: jaeger
    sample_rate: 0.1
  logging:
    level: info
    format: json
    output_path: stdout
  profiling:
    enabled: false
    port: 6060
    path: /debug/pprof
EOF
fi

echo "Development environment setup complete!"
