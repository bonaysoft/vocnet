FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Install certificates required during go build (e.g. for fetching modules)
RUN apk --no-cache add ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build linux binary inside the container to ensure compatibility
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /workspace/bin/vocnet .

FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appuser && \
    adduser -S -D -H -u 1001 -h /app -s /sbin/nologin -G appuser -g appuser appuser

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /workspace/bin/vocnet ./vocnet

# Change ownership to non-root user
RUN chown appuser:appuser /app/vocnet

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./vocnet", "serve"]
