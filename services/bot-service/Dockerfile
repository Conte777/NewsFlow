# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Install dependencies
RUN apk add --no-cache git

# Copy pkg/proto (local dependency)
COPY pkg/proto ./pkg/proto

# Copy service
COPY services/bot-service ./services/bot-service

# Download dependencies with workspace disabled
WORKDIR /workspace/services/bot-service
ENV GOWORK=off
RUN go mod edit -replace github.com/Conte777/NewsFlow/pkg/proto=../../pkg/proto && \
    go mod tidy && \
    go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot-service ./cmd/app

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /workspace/services/bot-service/bot-service .

EXPOSE 8081

CMD ["./bot-service"]
