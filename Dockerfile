# Multi-stage build for production
FROM golang:1.21-alpine AS builder

# Install required packages
RUN apk add --no-cache git make protoc

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate code and build
RUN make generate
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/rockd-server ./cmd/server

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appuser && \
    adduser -S -D -H -u 1001 -h /app -s /sbin/nologin -G appuser -g appuser appuser

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/rockd-server .

# Change ownership to non-root user
RUN chown appuser:appuser /app/rockd-server

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./rockd-server"]
