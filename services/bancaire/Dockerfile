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
COPY services/bancaire/ ./services/bancaire/

# Download dependencies and tidy
WORKDIR /app/services/bancaire
RUN go mod tidy && go mod download

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -tags musl -o /bancaire ./cmd/bancaire

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates librdkafka

WORKDIR /app

COPY --from=builder /bancaire .

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8081 9091

ENTRYPOINT ["./bancaire"]
