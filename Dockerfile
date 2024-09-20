# Step 1: Build the Go binary for Schednex
FROM golang:1.23-alpine AS builder

# Install git for dependency management
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum to cache dependencies
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o schednex .

# Step 2: Create a smaller image with the binary
FROM alpine:latest

# Install certificates for Kubernetes API communication
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the built Go binary from the builder stage
COPY --from=builder /app/ .

# Expose port if necessary (e.g., for metrics or health checks)
# EXPOSE 8080

# Run the Schednex scheduler
CMD ["/app/schednex"]
