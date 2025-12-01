# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NewsFlow is a microservices-based Telegram news aggregation system. Users subscribe to channels via a Telegram bot, and the system forwards news from those channels to users. The architecture uses Go microservices communicating asynchronously through Apache Kafka.

## Architecture

### Microservices Communication Pattern

All services communicate **asynchronously** via Kafka. There are no synchronous HTTP calls between services.

**Flow Example - User subscribes to a channel:**
1. Bot Service receives `/subscribe` command from user
2. Bot Service publishes to `subscriptions.created` Kafka topic
3. Subscription Service consumes event, stores in PostgreSQL
4. Subscription Service notifies Account Service via Kafka
5. Account Service joins the channel using MTProto
6. Account Service monitors channel for new messages
7. On new message, Account Service publishes to `news.received` topic
8. News Service consumes, stores, and publishes to `news.deliver` topic
9. Bot Service consumes `news.deliver` and sends to user via Bot API

### Kafka Topics

- `subscriptions.created` - New subscription created (userID, channelID, channelName)
- `subscriptions.deleted` - Subscription removed (userID, channelID)
- `news.received` - New message received from channel (channelID, messageID, content, mediaURLs)
- `news.deliver` - Deliver news to specific user (userID, newsContent)

### Services

**Bot Service** (port 8081)
- Handles user commands via Telegram Bot API
- Produces subscription events to Kafka
- Consumes news delivery events from Kafka
- No database dependency

**Subscription Service** (port 8082)
- Manages user subscriptions in PostgreSQL (`subscriptions_db:5432`)
- Tracks which users are subscribed to which channels
- Enforces unique constraint: one user cannot subscribe to same channel twice
- Provides list of active channels for Account Service

**News Service** (port 8083)
- Stores all received news in PostgreSQL (`news_db:5433`)
- Tracks delivery status per user in `delivered_news` table
- Prevents duplicate news delivery with unique constraint (news_id, user_id)
- Queries Subscription Service data via Kafka to determine delivery targets

**Account Service** (port 8084)
- Manages multiple Telegram user accounts via MTProto (gotd/td library)
- Joins/leaves channels based on subscription events
- Polls channels for new messages every 30s (configurable via `NEWS_POLL_INTERVAL`)
- Uses in-memory repository for active channels (no database)
- Session storage in `./sessions` directory

## Development Commands

### Infrastructure Setup

Start all infrastructure dependencies (Kafka, PostgreSQL, Kafka UI):
```bash
docker-compose -f deployments/docker-compose.yml up -d
```

Access Kafka UI at http://localhost:8080 to monitor topics and messages.

PostgreSQL instances:
- Subscriptions DB: `localhost:5432` (subscriptions_user/subscriptions_pass/subscriptions_db)
- News DB: `localhost:5433` (news_user/news_pass/news_db)

### Running Services

Each service runs independently. In separate terminals:

```bash
# Bot Service
cd services/bot-service && go run cmd/app/main.go

# Subscription Service
cd services/subscription-service && go run cmd/app/main.go

# News Service
cd services/news-service && go run cmd/app/main.go

# Account Service
cd services/account-service && go run cmd/app/main.go
```

### Database Migrations

Migrations are SQL files in `services/*/migrations/`. Apply manually:

```bash
# Example for subscription-service
psql postgresql://subscriptions_user:subscriptions_pass@localhost:5432/subscriptions_db \
  < services/subscription-service/migrations/000001_init.up.sql
```

### Testing

Standard Go testing:

```bash
# Test specific service
cd services/bot-service && go test ./...

# Test with coverage
go test -cover ./...

# Test specific package
go test ./internal/usecase/...
```

### Dependencies

```bash
# Add dependency to specific service
cd services/bot-service && go get github.com/package/name

# Update all dependencies
go get -u ./...

# Tidy modules
go mod tidy
```

## Code Structure (Clean Architecture)

Each service follows this structure:

```
services/service-name/
├── cmd/app/main.go              # Entrypoint: wires dependencies, starts service
├── config/config.go             # Environment variable loading with defaults
├── internal/
│   ├── domain/                  # Business entities and interfaces (no dependencies)
│   │   ├── models.go            # Core entities (Subscription, News, etc)
│   │   ├── interfaces.go        # Repository, UseCase, Kafka interfaces
│   │   └── errors.go            # Domain-specific errors
│   ├── usecase/                 # Business logic implementation
│   │   └── *_usecase.go         # Implements UseCase interfaces from domain
│   ├── repository/              # Data persistence
│   │   └── postgres/            # PostgreSQL implementation of Repository interface
│   ├── delivery/                # External interfaces
│   │   ├── telegram/            # Telegram bot handlers
│   │   └── kafka/               # Kafka consumers
│   └── infrastructure/          # External service clients
│       ├── logger/              # Zerolog wrapper
│       ├── database/            # GORM connection setup
│       └── kafka/               # Kafka producer/consumer setup
├── migrations/                  # SQL migration files
├── .env.example                 # Required environment variables
├── Dockerfile
└── go.mod
```

**Dependency Rules:**
- `domain` has NO external dependencies (pure Go)
- `usecase` depends only on `domain` interfaces
- `repository`, `delivery`, `infrastructure` implement `domain` interfaces
- `cmd/app/main.go` creates concrete implementations and injects them

## Key Implementation Details

### Configuration Pattern

All services use the same config pattern:
- `config/config.go` loads from environment variables
- Uses `godotenv` to load `.env` files
- `config.Validate()` ensures required values are present
- Default values provided for optional settings

### Error Handling

Domain-specific errors defined in `internal/domain/errors.go`:
```go
var (
    ErrSubscriptionExists = errors.New("subscription already exists")
    ErrSubscriptionNotFound = errors.New("subscription not found")
)
```

Use these errors in usecases and check with `errors.Is()` in handlers.

### Kafka Event Structure

Events are JSON-encoded. Example subscription created event:
```json
{
  "user_id": 123456789,
  "channel_id": "@channelname",
  "channel_name": "Channel Display Name"
}
```

Producers and consumers should handle JSON marshaling/unmarshaling in infrastructure layer.

### Telegram Bot Commands

Defined in Bot Service delivery layer:
- `/start` - Initialize user, send welcome message
- `/help` - Show available commands
- `/subscribe @channel` - Subscribe to channel
- `/unsubscribe @channel` - Unsubscribe from channel
- `/list` - Show user's subscriptions

### MTProto Account Management

Account Service manages Telegram user accounts (not bots):
- Requires `TELEGRAM_API_ID` and `TELEGRAM_API_HASH` from https://my.telegram.org
- Session files stored in `TELEGRAM_SESSION_DIR` (default: `./sessions`)
- Uses gotd/td library for MTProto implementation
- Can manage multiple accounts simultaneously

## Environment Variables

Copy `.env.example` to `.env` in each service directory before running.

**Critical Variables:**
- `TELEGRAM_BOT_TOKEN` - Get from @BotFather (Bot Service only)
- `TELEGRAM_API_ID` / `TELEGRAM_API_HASH` - From my.telegram.org (Account Service only)
- `DATABASE_*` - PostgreSQL connection details (News and Subscription Services)
- `KAFKA_BROKERS` - Comma-separated list (default: `localhost:9093`)

**Optional but Important:**
- `LOG_LEVEL` - debug/info/warn/error (default: info)
- `NEWS_POLL_INTERVAL` - How often Account Service checks channels (default: 30s)

## Common Development Tasks

### Adding a New Kafka Topic

1. Define topic name as constant in relevant service
2. Add to docker-compose.yml if auto-creation is disabled
3. Update this document's Kafka Topics section
4. Implement producer in service that publishes
5. Implement consumer in service that subscribes

### Adding a New Service

1. Copy structure from existing service
2. Create `go.mod` with appropriate module path
3. Define domain entities and interfaces
4. Implement usecases, repositories, delivery handlers
5. Update docker-compose.yml if database needed
6. Add service to this document

### Modifying Database Schema

1. Create new migration files: `000002_description.up.sql` and `000002_description.down.sql`
2. Place in service's `migrations/` directory
3. Apply manually with psql (no migration tool currently integrated)
4. Update corresponding `domain/models.go` struct
5. Update GORM repository queries if needed

### Debugging Kafka Issues

1. Check Kafka UI at http://localhost:8080
2. Verify topic exists and has messages
3. Check consumer group lag
4. Ensure `KAFKA_BROKERS` points to correct address
5. For local development, use `localhost:9093` (PLAINTEXT_HOST listener)
6. For service-to-service in Docker, use `kafka:9092` (PLAINTEXT listener)
