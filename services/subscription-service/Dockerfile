# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Install dependencies
RUN apk add --no-cache git

# Copy pkg/proto (local dependency)
COPY pkg/proto ./pkg/proto

# Copy service
COPY services/subscription-service ./services/subscription-service

# Download dependencies with workspace disabled
WORKDIR /workspace/services/subscription-service
ENV GOWORK=off
RUN go mod edit -replace github.com/Conte777/NewsFlow/pkg/proto=../../pkg/proto && \
    go mod tidy && \
    go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subscription-service ./cmd/app

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /workspace/services/subscription-service/subscription-service .

# Copy migrations
COPY --from=builder /workspace/services/subscription-service/migrations ./migrations

EXPOSE 8082

CMD ["./subscription-service"]
