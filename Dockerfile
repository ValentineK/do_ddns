# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go files
COPY main.go .

# Initialize go module and build
RUN go mod init ddns-updater && \
    go build -o ddns-updater main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S ddns && \
    adduser -u 1001 -S ddns -G ddns

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/ddns-updater .

# Change ownership
RUN chown ddns:ddns /app/ddns-updater

# Switch to non-root user
USER ddns

# Run the updater
ENTRYPOINT ["./ddns-updater"]
