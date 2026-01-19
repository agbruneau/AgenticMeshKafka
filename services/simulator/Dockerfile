# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Disable go.work for Docker builds
ENV GOWORK=off

# Copy module files first for better caching
COPY pkg/config/go.mod pkg/config/go.sum ./pkg/config/
COPY pkg/kafka/go.mod pkg/kafka/go.sum ./pkg/kafka/
COPY pkg/database/go.mod pkg/database/go.sum ./pkg/database/
COPY pkg/events/go.mod pkg/events/go.sum ./pkg/events/
COPY pkg/observability/go.mod pkg/observability/go.sum ./pkg/observability/
COPY services/simulator/go.mod services/simulator/go.sum ./services/simulator/

# Download dependencies
RUN cd services/simulator && go mod download

# Copy source code
COPY pkg/ ./pkg/
COPY services/simulator/ ./services/simulator/

# Build the binary (pure Go, no CGO)
WORKDIR /app/services/simulator
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /simulator ./cmd/simulator

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /simulator .

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Run the binary
ENTRYPOINT ["/app/simulator"]
