# Multi-stage Dockerfile for Quant WebWorks GO
# Stage 1: Build the React frontend
FROM node:18-alpine AS frontend-builder
WORKDIR /app/client

# Copy package.json and install dependencies
COPY client/package.json client/package-lock.json* ./
RUN npm ci

# Copy the rest of the client code and build
COPY client/ ./
RUN npm run build

# Stage 2: Build the Go backend
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy and download Go module dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY tests/ ./tests/

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o bridge-server ./cmd/bridge/main.go

# Stage 3: Final lightweight image
FROM alpine:3.18
WORKDIR /app

# Install necessary runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Copy the backend binary
COPY --from=backend-builder --chown=appuser:appgroup /app/bridge-server .

# Copy the React frontend build
COPY --from=frontend-builder --chown=appuser:appgroup /app/client/build ./client/build

# Create necessary directories with proper permissions
RUN mkdir -p /app/data /app/logs

# Environment variables
ENV PORT=8080
ENV GIN_MODE=release
ENV ENABLE_WEB=true
ENV CORS_ORIGINS=http://localhost:3000,https://app.example.com
ENV LOG_LEVEL=info

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:$PORT/health || exit 1

# Run the application
CMD ["./bridge-server"]
