# Account Service

Микросервис для управления Telegram аккаунтами и сбора новостей из каналов.

## Возможности

- Управляет пулом Telegram аккаунтов через MTProto (gotd/td)
- Синхронизирует активные аккаунты из БД по расписанию
- Подписывается/отписывается по саге через Kafka
- Периодически собирает новости из каналов (по умолчанию каждые 30 сек)
- Публикует собранные новости и события редактирования/удаления в Kafka

## Быстрый старт

**Требования:**
- Go 1.24+
- PostgreSQL
- Kafka
- Telegram API credentials ([получить на my.telegram.org](https://my.telegram.org))

**Запуск:**
```powershell
cp .env.example .env
# Добавить обязательные переменные в .env (см. ниже)
go run ./cmd/app
```

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `TELEGRAM_API_ID` | да | - | Telegram API ID (my.telegram.org) |
| `TELEGRAM_API_HASH` | да | - | Telegram API Hash |
| `TELEGRAM_ACCOUNT_SYNC_INTERVAL` | нет | 1m | Период проверки новых аккаунтов в БД |
| `TELEGRAM_MIN_REQUIRED_ACCOUNTS` | нет | - | Минимум аккаунтов для старта (если не задано, не проверяется) |
| `DATABASE_HOST` | да | localhost | Хост PostgreSQL |
| `DATABASE_PORT` | да | 5434 | Порт PostgreSQL |
| `DATABASE_USER` | да | accounts_user | Пользователь БД |
| `DATABASE_PASSWORD` | да | accounts_pass | Пароль БД |
| `DATABASE_NAME` | да | accounts_db | Название БД |
| `DATABASE_SSLMODE` | нет | disable | SSL режим |
| `KAFKA_BROKERS` | да | localhost:9093 | Kafka брокеры |
| `KAFKA_GROUP_ID` | да | account-service-group | Consumer group |
| `KAFKA_TOPIC_NEWS_RECEIVED` | да | news.received | Топик публикации новостей |
| `KAFKA_TOPIC_SUBSCRIPTION_PENDING` | нет | subscription.pending | Входящие запросы на подписку (consumer) |
| `KAFKA_TOPIC_SUBSCRIPTION_ACTIVATED` | нет | subscription.activated | Результат подписки (producer) |
| `KAFKA_TOPIC_SUBSCRIPTION_FAILED` | нет | subscription.failed | Ошибка подписки (producer) |
| `KAFKA_TOPIC_UNSUBSCRIPTION_PENDING` | нет | unsubscription.pending | Входящие запросы на отписку (consumer) |
| `KAFKA_TOPIC_UNSUBSCRIPTION_COMPLETED` | нет | unsubscription.completed | Результат отписки (producer) |
| `KAFKA_TOPIC_UNSUBSCRIPTION_FAILED` | нет | unsubscription.failed | Ошибка отписки (producer) |
| `NEWS_POLL_INTERVAL` | нет | 30s | Интервал опроса новостей |
| `NEWS_FALLBACK_POLL_INTERVAL` | нет | 5m | Интервал фоллбека при работающих realtime-обновлениях |
| `NEWS_COLLECTION_TIMEOUT` | нет | 5m | Таймаут одного цикла сбора |
| `SERVICE_PORT` | нет | 8084 | Порт HTTP сервера |
| `SERVICE_SHUTDOWN_TIMEOUT` | нет | 30s | Таймаут graceful shutdown |
| `S3_ENDPOINT` | нет | localhost:9000 | Хранилище медиа |
| `S3_ACCESS_KEY` | нет | minioadmin | Ключ доступа |
| `S3_SECRET_KEY` | нет | minioadmin | Секретный ключ |
| `S3_BUCKET` | нет | newsflow-media | Bucket для медиа |
| `S3_USE_SSL` | нет | false | Использовать SSL для S3 |
| `S3_PUBLIC_URL` | нет | http://localhost:9000 | Публичный URL для раздачи медиа |
| `LOG_LEVEL` | нет | info | Уровень логирования |

## Взаимодействие с сервисами

```
Subscription Service ──(Kafka)──► Account Service ──(Kafka)──► News Service
          ▲                              │                     ▲
          │                              │                     │
          └───────────────(Kafka)────────┘                     │
                                         │ MTProto             │
                                         ▼                     │
                                   Telegram API                │
                                                               │
                        Bot Service ──(Kafka: news.deliver/news.delivered)─┘
```

### Kafka Topics

| Topic | Направление | Описание |
|-------|-------------|----------|
| `subscription.pending` | Consumer | Запрос на подписку от subscription-service |
| `subscription.activated` | Producer | Успешная подписка (в ответ subscription-service) |
| `subscription.failed` | Producer | Ошибка подписки (в ответ subscription-service) |
| `unsubscription.pending` | Consumer | Запрос на отписку от subscription-service |
| `unsubscription.completed` | Producer | Успешная отписка (в ответ subscription-service) |
| `unsubscription.failed` | Producer | Ошибка отписки (в ответ subscription-service) |
| `news.received` | Producer | Собранные новости |
| `news.edited` | Producer | Событие редактирования новости |
| `news.deleted` | Producer | Событие удаления новости |

## API

| Endpoint | Описание |
|----------|----------|
| `GET /health` | Проверка состояния сервиса |
| `GET /metrics` | Prometheus метрики |

## Структура проекта

```
cmd/app/main.go           - Точка входа (fx.New)
config/                   - Конфигурация
internal/
├── app/                  - FX bootstrap
├── domain/
│   ├── account/          - HTTP handlers
│   ├── channel/          - Kafka handler, usecase
│   └── news/             - Worker, usecase
└── infrastructure/
    ├── database/         - PostgreSQL, миграции
    ├── http/             - FastHTTP server
    ├── kafka/            - Sarama producer
    ├── logger/           - zerolog
    ├── metrics/          - Prometheus
    └── telegram/         - MTProto client, AccountManager
```

## Сборка и тесты

```powershell
# Сборка
go build -o account-service.exe ./cmd/app

# Docker
docker build -t account-service .

# Тесты
go test ./...
```
