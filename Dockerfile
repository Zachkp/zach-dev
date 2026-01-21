# Use official Go image as build stage
FROM golang:1.24-alpine AS builder

# Set working directory inside container
WORKDIR /app

# Copy go mod files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source code
COPY . .

# Build the Go application
RUN go build -o main .

# Use minimal alpine image for final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the built binary from builder stage
COPY --from=builder /app/main .

# Copy static assets
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static  
COPY --from=builder /app/images ./images

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run when container starts
CMD ["./main"]