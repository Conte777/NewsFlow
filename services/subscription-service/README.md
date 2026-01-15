# Subscription Service

Микросервис управления подписками пользователей на Telegram-каналы.

## Возможности

- Хранит подписки пользователей на каналы (PostgreSQL)
- Предоставляет gRPC API для получения подписок
- Обрабатывает события подписки/отписки через Kafka
- Уведомляет account-service об изменениях подписок

## Быстрый старт

**Требования:**
- Go 1.24+
- PostgreSQL с базой `subscriptions_db`
- Kafka

**Запуск:**
```powershell
cp .env.example .env
go run ./cmd/app
```

**Порты:**
- `8082` — HTTP (health checks)
- `50051` — gRPC API

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `DATABASE_HOST` | да | localhost | Хост PostgreSQL |
| `DATABASE_PORT` | да | 5432 | Порт PostgreSQL |
| `DATABASE_USER` | да | subscriptions_user | Пользователь БД |
| `DATABASE_PASSWORD` | да | subscriptions_pass | Пароль БД |
| `DATABASE_NAME` | да | subscriptions_db | Название БД |
| `DATABASE_SSLMODE` | нет | disable | SSL режим |
| `KAFKA_BROKERS` | да | localhost:9093 | Kafka брокеры |
| `KAFKA_GROUP_ID` | нет | subscription-service-group | Consumer Group ID |
| `GRPC_PORT` | нет | 50051 | Порт gRPC сервера |
| `LOG_LEVEL` | нет | info | Уровень логирования |
| `SERVICE_PORT` | нет | 8082 | HTTP порт |

## Взаимодействие с сервисами

```
Bot Service ──(Kafka)──► Subscription Service ──(Kafka)──► Account Service
                               │
     ┌─────────────────────────┼─────────────────────────┐
     │                         │                         │
     ▼ gRPC                    ▼ gRPC                    ▼ gRPC
Bot Service              News Service              Account Service
```

### Kafka Topics

| Topic | Направление | Описание |
|-------|-------------|----------|
| `subscription.created` | Consumer/Producer | Новая подписка |
| `subscription.cancelled` | Consumer/Producer | Отписка |

### gRPC API

| Метод | Описание |
|-------|----------|
| `GetUserSubscriptions(user_id)` | Список подписок пользователя |
| `GetChannelSubscribers(channel_id)` | Список подписчиков канала |

## Структура проекта

```
cmd/app/main.go           - Точка входа (fx.New)
config/                   - Конфигурация
internal/
├── app/                  - FX bootstrap
├── domain/subscription/
│   ├── entities/         - Subscription
│   ├── deps/             - Интерфейсы
│   ├── consts/           - Kafka topics
│   ├── delivery/kafka/   - Kafka consumer handlers
│   ├── usecase/          - Бизнес-логика
│   └── repository/       - PostgreSQL репозиторий
├── delivery/grpc/        - gRPC server
└── infrastructure/
    ├── database/         - PostgreSQL (GORM)
    ├── kafka/            - Producer/Consumer
    └── logger/           - zerolog
migrations/               - SQL миграции
```

## Сборка и тесты

```powershell
# Сборка
go build -o subscription-service.exe ./cmd/app

# Docker
docker build -t subscription-service .

# Тесты
go test ./...

# Валидация FX графа
go test ./internal/app -run TestCreateApp
```
