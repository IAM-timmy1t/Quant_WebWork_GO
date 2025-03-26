# QUANT_WebWork_GO Initial Setup Guide

## Prerequisites

Before setting up QUANT_WebWork_GO, ensure you have the following prerequisites installed:

- **Go** (version 1.21 or higher)
- **Docker** (version 20.10 or higher)
- **Docker Compose** (version 2.0 or higher)
- **Node.js** (version 18 or higher, for frontend development)
- **Git** (for version control)

## Clone the Repository

```bash
git clone https://github.com/IAM-timmy1t/Quant_WebWork_GO.git
cd Quant_WebWork_GO
```

## Project Structure Overview

The QUANT_WebWork_GO project follows a standard Go project layout with clear separation of concerns:

```markdown
Quant_WebWork_GO/
├── cmd/                  # Application entry points
│   └── server/           # Main server executable
├── internal/             # Private application code
│   ├── api/              # API handlers and routes
│   │   ├── rest/         # REST API implementation
│   │   └── graphql/      # GraphQL implementation
│   ├── bridge/           # Bridge system implementation
│   │   └── adapters/     # Protocol adapters
│   ├── core/             # Core components
│   │   ├── config/       # Configuration management
│   │   ├── discovery/    # Service discovery
│   │   ├── metrics/      # Metrics collection
│   │   └── security/     # Security components
│   ├── middleware/       # HTTP middleware components
│   └── security/         # Security implementations
│       ├── firewall/     # Firewall components
│       └── ipmasking/    # IP masking implementation
├── pkg/                  # Public libraries for external applications
├── web/                  # Frontend React application
│   └── client/           # React client code
├── scripts/              # Build and deployment scripts
├── tests/                # Test files
├── deployments/          # Deployment manifests
│   ├── monitoring/       # Prometheus and Grafana configuration
│   └── k8s/              # Kubernetes manifests
├── docs/                 # Documentation
├── .github/              # GitHub workflow configurations
├── Dockerfile            # Multi-stage production Dockerfile
├── Dockerfile.dev        # Development Dockerfile with debugging tools
├── docker-compose.yml    # Production Docker Compose configuration
├── docker-compose.dev.yml # Development Docker Compose configuration
├── .air.toml             # Air configuration for hot reloading
├── go.mod                # Go module definition
└── go.sum                # Go module checksum
```

## Important Note on Module Path

This project uses the module path `github.com/IAM-timmy1t/Quant_WebWork_GO`. When importing internal packages, always use the full import path:

```go
// Correct import path - follows the module declaration in go.mod
import "github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"

// Incorrect import path - will cause "no matching versions" errors
import "github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/internal/core/config"
```

If you encounter import errors, check the following:

1. Ensure the directory structure matches the imports
2. Verify there are no duplicate directory segments in your import paths
3. Confirm all imported packages exist in the filesystem
4. Always use the module path from go.mod as the root of your imports

## Development Setup

### Option 1: Using Docker (Recommended)

The project includes Docker configurations for both production and development environments.

#### Development Environment

```bash
# Build and start all services in development mode with hot reloading
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose -f docker-compose.dev.yml logs -f

# Access the services
# - Bridge Server: http://localhost:8080
# - Frontend: http://localhost:3000
# - Grafana: http://localhost:3001 (admin/admin)
# - Prometheus: http://localhost:9091
```

The development environment includes:

- Hot reloading with Air for Go code changes
- React development server with hot reloading
- Development databases (PostgreSQL and Redis)
- Prometheus and Grafana for monitoring
- Mock server for testing external dependencies

#### Production-like Environment

```bash
# Build and start all services in production-like mode
docker-compose up -d

# View logs
docker-compose logs -f
```

### Option 2: Local Development

For local development without Docker:

1. Install Go dependencies:

   ```bash
   go mod download
   ```

2. Install Air for hot reloading (optional):

   ```bash
   go install github.com/cosmtrek/air@latest
   ```

3. Install frontend dependencies:

   ```bash
   cd web/client
   npm install
   ```

4. Start the backend server:

   ```bash
   # Without hot reloading
   go run cmd/server/main.go

   # With hot reloading using Air
   air -c .air.toml
   ```

5. Start the frontend development server:

   ```bash
   cd web/client
   npm start
   ```

## Configuration

### Environment Variables

The application supports configuration through environment variables:

```bash
# Server configuration
PORT=8080
ENABLE_WEB=true
CORS_ORIGINS=http://localhost:3000
LOG_LEVEL=debug  # debug, info, warn, error

# Security
ENABLE_IP_MASKING=true
IP_MASK_ROTATION_INTERVAL=3600  # Rotation interval in seconds
TOKEN_VALIDATION_LEVEL=strict
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60

# Metrics
METRICS_ENABLED=true
PROMETHEUS_ENDPOINT=/metrics
METRICS_COLLECTION_INTERVAL=15  # Collection interval in seconds
```

### Configuration Files

Configuration can also be provided through YAML, JSON, or TOML files:

```bash
# Specify a config file path
CONFIG_PATH=./config/dev
```

Example configuration file (config/dev/config.yaml):

```yml
server:
  port: 8080
  enable_web: true
  cors_origins:
    - http://localhost:3000
  log_level: debug

security:
  enable_ip_masking: true
  ip_mask_rotation_interval: 3600
  token_validation_level: strict
  rate_limit:
    enabled: true
    requests: 100
    window: 60

metrics:
  enabled: true
  prometheus_endpoint: /metrics
  collection_namespace: quant
  collection_subsystem: webwork
  collection_interval: 15
```

## Testing

### Running Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge
```

### Running Integration Tests

```bash
# Run the integration test suite
go test ./tests/integration/...

# Run a specific integration test
go test ./tests/integration/bridge_test.go
```

### Running End-to-End Tests

```bash
# From the web/client directory
cd web/client
npm run cypress:open  # Interactive mode
npm run cypress:run   # Headless mode
```

## Common Tasks

### Adding a New API Endpoint

1. Define the handler in the appropriate file in `internal/api/rest/`
2. Register the route in `internal/api/rest/router.go`
3. Add authentication middleware if required
4. Update API documentation in `docs/api_specifications.md`

### Creating a New Bridge Adapter

1. Create a new adapter file in `internal/bridge/adapters/`
2. Implement the adapter interface defined in `internal/bridge/adapters/adapter.go`
3. Register the adapter in the bridge manager
4. Add tests for the new adapter

### Configuring IP Masking

1. Edit the security settings in your configuration file
2. Set the rotation interval appropriate for your security requirements
3. Enable or disable masking based on your needs

## Troubleshooting

### Import Path Issues

If you encounter "no matching versions" errors when importing packages:

```plaintext
go: finding module for package github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/internal/...
go: found github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/internal/... in github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO v0.0.0-00010101000000-000000000000: go.mod has non-...
```

Fix by correcting import paths to use the proper module path:

```go
// Change this:
import "github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/internal/core/config"

// To this:
import "github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"
```

### Docker Networking Issues

If services can't communicate within Docker:

1. Check that service names match the hostnames used in your code
2. Verify that ports are correctly exposed and mapped
3. Ensure all services are on the same Docker network

### Authentication Problems

If you're getting authentication errors:

1. Verify that your JWT secret is correctly configured
2. Check that tokens haven't expired
3. Ensure you're including the token in the correct format in your requests

## Further Resources

- [Go Modules Documentation](https://golang.org/ref/mod)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [React Documentation](https://reactjs.org/docs/getting-started.html)
- [Prometheus Documentation](https://prometheus.io/docs/introduction/overview/)
- [Grafana Documentation](https://grafana.com/docs/)
