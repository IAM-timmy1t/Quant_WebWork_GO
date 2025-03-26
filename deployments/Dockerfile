# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always) -X main.buildTime=$(date +%s)" \
    -o /go/bin/app ./cmd/server

# Client build stage
FROM node:20-alpine AS client-builder
WORKDIR /client
COPY client/package*.json ./
RUN npm ci
COPY client/ ./
RUN npm run build

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -H -u 1000 appuser

# Copy the binary from builder
COPY --from=builder /go/bin/app /app/
COPY --from=client-builder /client/build /app/client/build

# Copy config files
COPY configs/ /app/configs/
COPY deployments/monitoring/ /app/monitoring/

# Use non-root user
USER appuser

# Expose application port
EXPOSE 8080

# Set healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/app"]
