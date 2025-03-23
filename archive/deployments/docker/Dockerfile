# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev git python3 python3-dev py3-pip

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install Python dependencies
RUN pip3 install --no-cache-dir -r internal/bridge/pyhelper/requirements.txt

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o webworks ./cmd/webworks

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache python3 libmagic tzdata ca-certificates

# Create non-root user
RUN adduser -D -g '' webworks

# Set working directory
WORKDIR /app

# Copy binary and configs from builder
COPY --from=builder /build/webworks .
COPY --from=builder /build/internal/bridge/pyhelper /app/pyhelper
COPY --from=builder /usr/lib/python3.* /usr/lib/python3.*/
COPY --from=builder /usr/lib/libmagic.so.* /usr/lib/
COPY --from=builder /usr/share/misc/magic.mgc /usr/share/misc/

# Create necessary directories
RUN mkdir -p /app/uploads /app/config /app/logs \
    && chown -R webworks:webworks /app

# Switch to non-root user
USER webworks

# Expose ports
EXPOSE 8080

# Set environment variables
ENV PATH="/app:${PATH}" \
    PYTHONPATH="/app/pyhelper" \
    GO_ENV="production"

# Health check
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/webworks"]
