# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o account-service ./cmd/app

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/account-service .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Create sessions directory
RUN mkdir -p sessions

# Copy .env file if needed
COPY .env* ./

EXPOSE 8084

CMD ["./account-service"]
