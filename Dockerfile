# WebStack CLI - Dockerfile for building and distribution

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -a -installsuffix cgo \
    -o webstack .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl wget bash

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/webstack /usr/local/bin/webstack
COPY --from=builder /app/templates /etc/webstack/templates

# Make binary executable
RUN chmod +x /usr/local/bin/webstack

# Create necessary directories
RUN mkdir -p /etc/webstack /var/www /var/log/webstack

ENTRYPOINT ["webstack"]
CMD ["--help"]