FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kvirtos2 .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates && \
    adduser -D -s /bin/sh kvirtos2

WORKDIR /home/kvirtos2/

# Copy the binary from builder stage
COPY --from=builder /app/kvirtos2 .

# Change ownership to non-root user
RUN chown kvirtos2:kvirtos2 kvirtos2

# Switch to non-root user
USER kvirtos2

# Expose port
EXPOSE 8080

# Command to run
CMD ["./kvirtos2"]