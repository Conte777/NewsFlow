# Account Service

Микросервис для управления Telegram аккаунтами и сбора новостей из каналов.

## Возможности

- Управляет пулом Telegram аккаунтов через MTProto (gotd/td)
- Подписывается на каналы по событиям из Kafka
- Периодически собирает новости из каналов (по умолчанию каждые 30 сек)
- Публикует собранные новости в Kafka для дальнейшей обработки

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
| `TELEGRAM_ACCOUNTS` | да | - | Телефоны аккаунтов (через запятую) |
| `DATABASE_HOST` | да | localhost | Хост PostgreSQL |
| `DATABASE_PORT` | да | 5434 | Порт PostgreSQL |
| `DATABASE_USER` | да | accounts_user | Пользователь БД |
| `DATABASE_PASSWORD` | да | accounts_pass | Пароль БД |
| `DATABASE_NAME` | да | accounts_db | Название БД |
| `KAFKA_BROKERS` | да | localhost:9093 | Kafka брокеры |
| `SERVICE_PORT` | нет | 8084 | Порт HTTP сервера |
| `NEWS_POLL_INTERVAL` | нет | 30s | Интервал сбора новостей |
| `TELEGRAM_MIN_REQUIRED_ACCOUNTS` | нет | 50% от общего | Минимум аккаунтов для старта |
| `LOG_LEVEL` | нет | info | Уровень логирования |

## Взаимодействие с сервисами

```
Subscription Service ──(Kafka)──► Account Service ──(Kafka)──► News Service
                                         │
                                         │ MTProto
                                         ▼
                                   Telegram API
```

### Kafka Topics

| Topic | Направление | Описание |
|-------|-------------|----------|
| `subscription.created` | Consumer | Событие создания подписки |
| `subscription.cancelled` | Consumer | Событие удаления подписки |
| `news.received` | Producer | Собранные новости |

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
