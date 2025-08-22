# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy package files first for better caching
COPY web/package*.json ./

# Install dependencies
RUN npm ci --silent

# Copy source code
COPY web/ ./

# Build frontend with version argument
ARG VERSION=1.1.0
RUN VITE_VERSION=${VERSION} npm run build

# Stage 2: Build backend
FROM golang:1.23-alpine AS backend-builder

ARG VERSION=1.1.0
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Copy built frontend
COPY --from=frontend-builder /build/dist ./web/dist

# Build the Go application with optimizations
ARG VERSION=1.1.0
RUN go build \
    -ldflags "-s -w -X gpt-load/internal/version.Version=${VERSION}" \
    -trimpath \
    -o gpt-load .

# Stage 3: Final runtime image
FROM alpine:latest

LABEL maintainer="charles0568" \
      version="${VERSION}" \
      description="GPT-Load - AI API transparent proxy service"

WORKDIR /app

# Install runtime dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache \
        ca-certificates \
        tzdata \
        wget \
        curl && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

# Create app user for security
RUN adduser -D -s /bin/sh -u 1001 appuser

# Create necessary directories
RUN mkdir -p /app/data /app/logs && \
    chown -R appuser:appuser /app

# Copy binary from builder
COPY --from=backend-builder --chown=appuser:appuser /build/gpt-load /app/

# Switch to app user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

# Expose port
EXPOSE 3001

# Set environment variables
ENV GIN_MODE=release \
    LOG_LEVEL=info

# Volume for data persistence
VOLUME ["/app/data"]

# Start the application
ENTRYPOINT ["/app/gpt-load"]
