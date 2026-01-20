# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies for confluent-kafka-go
RUN apk add --no-cache gcc musl-dev librdkafka-dev pkgconf git

# Disable go workspace mode
ENV GOWORK=off

# Copy shared packages first
COPY pkg/ ./pkg/

# Copy service module
COPY services/gateway/ ./services/gateway/

# Download dependencies and tidy
WORKDIR /app/services/gateway
RUN go mod tidy && go mod download

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -tags musl -o /gateway ./cmd/gateway

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates librdkafka

WORKDIR /app

COPY --from=builder /gateway .

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8082 9092

ENTRYPOINT ["./gateway"]
