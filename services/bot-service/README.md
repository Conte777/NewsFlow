# Bot Service

Telegram бот для управления подписками на новостные каналы в системе NewsFlow.

## Возможности

- Подписка и отписка от Telegram каналов
- Просмотр текущих подписок
- Автоматическая доставка новостей из подписанных каналов

## Быстрый старт

**Требования:**
- Go 1.24+
- Запущенный Kafka
- Запущенный subscription-service (для команды /list)

**Запуск:**
```powershell
cp .env.example .env
# Отредактировать .env: TELEGRAM_BOT_TOKEN, KAFKA_BROKERS
go run ./cmd/app
```

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `TELEGRAM_BOT_TOKEN` | да | - | Токен Telegram бота |
| `KAFKA_BROKERS` | да | localhost:9093 | Kafka брокеры (через запятую) |
| `KAFKA_GROUP_ID` | нет | bot-service-group | Consumer group ID |
| `SUBSCRIPTION_SERVICE_GRPC_ADDR` | нет | localhost:50051 | Адрес subscription-service gRPC |
| `LOG_LEVEL` | нет | info | Уровень логов (debug/info/warn/error) |
| `SERVICE_NAME` | нет | bot-service | Имя сервиса |
| `SERVICE_PORT` | нет | 8081 | Порт сервиса |

## Взаимодействие с сервисами

```
                    ┌──────────────────────┐
                    │   Telegram Users     │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │     Bot Service      │
                    │     (порт 8081)      │
                    └──┬───────────────┬───┘
                       │               │
         ┌─────────────▼───┐    ┌──────▼──────────────┐
         │      Kafka      │    │ Subscription Service│
         │                 │    │   (gRPC :50051)     │
         │ subscription.   │    │                     │
         │   created  ──►  │    │ GetUserSubscriptions│
         │   cancelled ──► │    └─────────────────────┘
         │                 │
         │ news.deliver ◄──│
         └─────────────────┘
```

### Kafka Topics

| Topic | Направление | Описание |
|-------|-------------|----------|
| `subscription.created` | Producer | Событие новой подписки |
| `subscription.cancelled` | Producer | Событие отписки |
| `news.deliver` | Consumer | Получение новостей для доставки |

### gRPC

| Метод | Сервис | Описание |
|-------|--------|----------|
| `GetUserSubscriptions(user_id)` | Subscription Service | Список подписок пользователя для /list |

## Команды бота

| Команда | Описание |
|---------|----------|
| `/start` | Начать работу с ботом |
| `/help` | Справка по командам |
| `/subscribe @channel1 @channel2` | Подписаться на каналы |
| `/unsubscribe @channel1 @channel2` | Отписаться от каналов |
| `/list` | Список текущих подписок |

## Структура проекта

```
cmd/app/main.go           - Точка входа (fx.New)
config/                   - Конфигурация
internal/
├── app/                  - FX bootstrap
├── domain/bot/
│   ├── delivery/         - Telegram handlers, Kafka handlers
│   ├── usecase/          - Бизнес-логика
│   ├── repository/       - Kafka producer, gRPC clients
│   └── workers/          - News consumer worker
└── infrastructure/       - Logger, Telegram client
```

## Сборка и тесты

```powershell
# Сборка
go build -o bot-service.exe ./cmd/app

# Docker
docker build -t bot-service .

# Тесты
go test ./...
```
