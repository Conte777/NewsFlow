# Bot Service

Telegram бот для управления подписками на новостные каналы в системе NewsFlow.

## Возможности

- Подписка и отписка от Telegram каналов (сага через Kafka)
- Просмотр текущих подписок
- Автоматическая доставка новостей из подписанных каналов
- Подтверждение доставки новостей в news-service

## Быстрый старт

**Требования:**
- Go 1.24+
- Запущенный Kafka
- Запущенный subscription-service (для команды /list)
- PostgreSQL для хранения истории доставленных сообщений (по умолчанию `bot_db`)

**Запуск:**
```powershell
cp .env.example .env
# Обязательное: TELEGRAM_BOT_TOKEN
# При необходимости: DATABASE_*, KAFKA_TOPIC_* overrides
go run ./cmd/app
```

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `TELEGRAM_BOT_TOKEN` | да | - | Токен Telegram бота |
| `KAFKA_BROKERS` | да | localhost:9093 | Kafka брокеры (через запятую) |
| `KAFKA_GROUP_ID` | нет | bot-service-group | Consumer group ID для новостей |
| `KAFKA_TOPIC_SUBSCRIPTION_REQUESTED` | нет | subscription.requested | Запрос на подписку (producer) |
| `KAFKA_TOPIC_UNSUBSCRIPTION_REQUESTED` | нет | unsubscription.requested | Запрос на отписку (producer) |
| `KAFKA_TOPIC_SUBSCRIPTION_CONFIRMED` | нет | subscription.confirmed | Подтверждение подписки (consumer) |
| `KAFKA_TOPIC_UNSUBSCRIPTION_CONFIRMED` | нет | unsubscription.confirmed | Подтверждение отписки (consumer) |
| `KAFKA_TOPIC_SUBSCRIPTION_REJECTED` | нет | subscription.rejected | Отклонение подписки (consumer) |
| `KAFKA_TOPIC_UNSUBSCRIPTION_REJECTED` | нет | unsubscription.rejected | Отклонение отписки (consumer) |
| `KAFKA_TOPIC_NEWS_DELIVERED` | нет | news.delivered | Подтверждение доставки (producer) |
| `SUBSCRIPTION_SERVICE_GRPC_ADDR` | нет | localhost:50051 | Адрес subscription-service gRPC |
| `DATABASE_HOST` | нет | localhost | Хост PostgreSQL |
| `DATABASE_PORT` | нет | 5432 | Порт PostgreSQL |
| `DATABASE_USER` | нет | postgres | Пользователь БД |
| `DATABASE_PASSWORD` | нет | postgres | Пароль БД |
| `DATABASE_NAME` | нет | bot_db | Название БД |
| `DATABASE_SSLMODE` | нет | disable | SSL режим |
| `LOG_LEVEL` | нет | info | Уровень логов |
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
         │ subscription.   │    │ GetUserSubscriptions│
         │ requested   ──► │    └─────────────────────┘
         │ unsubscription. │
         │ requested   ──► │
         │ subscription.   │
         │ confirmed  ◄─── │
         │ subscription.   │
         │ rejected   ◄─── │
         │ unsubscription. │
         │ confirmed  ◄─── │
         │ unsubscription. │
         │ rejected   ◄─── │
         │                 │
         │ news.deliver ◄──│
         │ news.edit   ◄── │
         │ news.delete ◄── │
         │ news.delivered ─►
         └─────────────────┘
```

### Kafka Topics

| Topic | Направление | Описание |
|-------|-------------|----------|
| `subscription.requested` | Producer | Запрос на подписку |
| `unsubscription.requested` | Producer | Запрос на отписку |
| `subscription.confirmed` | Consumer | Подтверждение успешной подписки |
| `subscription.rejected` | Consumer | Отклонение подписки |
| `unsubscription.confirmed` | Consumer | Подтверждение успешной отписки |
| `unsubscription.rejected` | Consumer | Отклонение отписки |
| `news.deliver` | Consumer | Получение новостей для доставки |
| `news.edit` | Consumer | Обновление доставленной новости |
| `news.delete` | Consumer | Удаление доставленной новости |
| `news.delivered` | Producer | Подтверждение доставки в news-service |

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
