FROM node:20-alpine AS builder

ARG VERSION=1.1.0
WORKDIR /build
COPY ./web .
RUN npm ci --only=production
RUN VITE_VERSION=${VERSION} npm run build


FROM golang:1.23-alpine AS builder2

ARG VERSION=1.1.0
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /build

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .
# Copy built frontend
COPY --from=builder /build/dist ./web/dist

# Build the Go application
RUN go build -ldflags "-s -w -X gpt-load/internal/version.Version=${VERSION}" -o gpt-load .


FROM alpine:latest

WORKDIR /app

# Install required packages
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata \
    && update-ca-certificates

# Create app user for security
RUN adduser -D -s /bin/sh appuser

# Copy binary
COPY --from=builder2 /build/gpt-load .

# Change ownership
RUN chown appuser:appuser /app/gpt-load

# Switch to app user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

EXPOSE 3001

ENTRYPOINT ["/app/gpt-load"]
