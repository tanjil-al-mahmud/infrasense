# Multi-stage build for Redfish collector

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o redfish-collector ./cmd/main.go

# Runtime stage
FROM alpine:3.19

# Install ca-certificates and tzdata
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 collector && \
    adduser -D -u 1000 -G collector collector

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/redfish-collector .

# Copy config example (can be overridden with volume mount)
COPY config.example.yml config.yml

# Change ownership
RUN chown -R collector:collector /app

# Switch to non-root user
USER collector

# Expose health check port
EXPOSE 8081

# Run the collector
CMD ["./redfish-collector"]
