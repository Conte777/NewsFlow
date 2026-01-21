# News Service

Сервис обработки и доставки новостей из Telegram каналов подписчикам.

## Возможности

- Получает новости от account-service через Kafka (`news.received`)
- Сохраняет новости в PostgreSQL
- Запрашивает подписчиков канала у subscription-service (gRPC)
- Отправляет новости на доставку в bot-service (`news.deliver`)
- Принимает подтверждения доставки (`news.delivered`)
- Обрабатывает редактирование/удаление (`news.edited`/`news.deleted` → `news.edit`/`news.delete`)
- Отслеживает историю доставки (таблица `delivered_news`)

## Быстрый старт

**Требования:**
- Go 1.24+
- PostgreSQL с базой `news_db`
- Kafka
- Запущенный subscription-service (для gRPC)

**Запуск:**
```powershell
cp .env.example .env
# Отредактировать .env под локальное окружение
go run ./cmd/app
```

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `DATABASE_HOST` | да | localhost | Хост PostgreSQL |
| `DATABASE_PORT` | да | 5433 | Порт PostgreSQL |
| `DATABASE_USER` | да | news_user | Пользователь БД |
| `DATABASE_PASSWORD` | да | news_pass | Пароль БД |
| `DATABASE_NAME` | да | news_db | Название БД |
| `KAFKA_BROKERS` | да | localhost:9093 | Kafka брокеры |
| `KAFKA_GROUP_ID` | нет | news-service-group | Consumer Group ID (news.received) |
| `KAFKA_GROUP_ID_DELETED` | нет | news-service-deleted-group | Consumer Group ID (news.deleted) |
| `KAFKA_GROUP_ID_EDITED` | нет | news-service-edited-group | Consumer Group ID (news.edited) |
| `KAFKA_GROUP_ID_DELIVERED` | нет | news-service-delivered-group | Consumer Group ID (news.delivered) |
| `KAFKA_TOPIC_NEWS_RECEIVED` | нет | news.received | Топик входящих новостей |
| `KAFKA_TOPIC_NEWS_DELIVERED` | нет | news.delivered | Топик подтверждений доставки |
| `SUBSCRIPTION_SERVICE_GRPC_ADDR` | нет | localhost:50051 | Адрес subscription-service |
| `LOG_LEVEL` | нет | info | Уровень логирования |
| `SERVICE_PORT` | нет | 8083 | Порт сервиса |

## Взаимодействие с сервисами

```
Account Service ──(Kafka)──► News Service ──(Kafka)──► Bot Service
       │                         │  ▲                ▲
       │                         │  │                │
       └──(Kafka: news.edited/   │  │                │
               news.deleted)─────┘  │                │
                                    │ gRPC           │
                                    ▼                │
                           Subscription Service      │
                                                     │
                         Bot Service ──(Kafka: news.delivered)──► News Service
```

### Kafka Topics

| Topic | Направление | Описание |
|-------|-------------|----------|
| `news.received` | Consumer | Новости от account-service |
| `news.deliver` | Producer | Доставка в bot-service |
| `news.delivered` | Consumer | Подтверждение доставки от bot-service |
| `news.edited` | Consumer | Событие редактирования от account-service |
| `news.deleted` | Consumer | Событие удаления от account-service |
| `news.edit` | Producer | Команда на обновление новости в bot-service |
| `news.delete` | Producer | Команда на удаление новости в bot-service |

### gRPC

| Метод | Сервис | Описание |
|-------|--------|----------|
| `GetChannelSubscribers(channel_id)` | Subscription Service | Список подписчиков канала |

## Структура проекта

```
cmd/app/main.go           - Точка входа (fx.New)
config/                   - Конфигурация
internal/
├── app/                  - FX bootstrap
├── domain/news/
│   ├── entities/         - News, DeliveredNews
│   ├── deps/             - Интерфейсы
│   ├── delivery/kafka/   - Kafka consumer handlers
│   ├── usecase/          - Бизнес-логика
│   └── repository/
│       ├── postgres/     - Репозитории
│       └── grpc_clients/ - gRPC клиенты
└── infrastructure/
    ├── database/         - PostgreSQL (GORM)
    ├── kafka/            - Producer
    └── logger/           - zerolog
migrations/               - SQL миграции
```

## Сборка и тесты

```powershell
# Сборка
go build -o news-service.exe ./cmd/app

# Docker
docker build -t news-service .

# Тесты
go test ./...
```
