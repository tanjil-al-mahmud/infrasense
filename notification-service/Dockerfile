# Multi-stage build for notification service

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o notification-service ./cmd/main.go

# Runtime stage
FROM alpine:3.19

# Install ca-certificates and tzdata
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 notifier && \
    adduser -D -u 1000 -G notifier notifier

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/notification-service .

# Copy config example (can be overridden with volume mount)
COPY config.example.yml config.yml

# Change ownership
RUN chown -R notifier:notifier /app

# Switch to non-root user
USER notifier

# Expose webhook port
EXPOSE 8080

# Run the service
CMD ["./notification-service"]
