# Multi-stage build for minimal image size
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/trakt-sync ./cmd/trakt-sync

# Runtime image
FROM alpine:latest

# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/trakt-sync .

# Copy example config
COPY config.example.yaml /app/config.example.yaml

# Create non-root user
RUN adduser -D -u 1000 traktsync && \
    chown -R traktsync:traktsync /app

USER traktsync

# Default command: run daemon with 6h interval
CMD ["./trakt-sync", "daemon", "--interval", "6h"]
