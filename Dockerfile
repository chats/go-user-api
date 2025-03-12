FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install required packages
RUN apk add --no-cache git gcc libc-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o go-user-api ./cmd/server

# Use a small alpine image for the final stage
FROM alpine:3.16

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/go-user-api /app/go-user-api

# Copy migrations
COPY --from=builder /app/internal/database/migrations /app/internal/database/migrations

# Install CA certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata && \
    update-ca-certificates

# Create a non-root user and set permissions
RUN adduser -D -g '' appuser && \
    chown -R appuser:appuser /app
USER appuser

# Expose HTTP and gRPC ports
EXPOSE 8080
EXPOSE 50051

# Make sure the binary is executable
RUN chmod +x /app/go-user-api

# Run the application
CMD ["/app/go-user-api"]