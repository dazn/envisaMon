# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o envisaMon .
FROM alpine:latest

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /build/envisaMon .
# Create logs directory
RUN mkdir -p logs

# Volume for persistent logs
VOLUME ["/app/logs"]
ENTRYPOINT ["./envisaMon"]

# Default command shows usage (will fail without required arguments)
CMD []
