# NewsFlow

Проект для создания персонализированной ленты новостей в Telegram. Пользователи подписываются на каналы через бота, и система автоматически пересылает новости из этих каналов.

## Архитектура

Проект построен на основе микросервисной архитектуры с использованием Clean Architecture принципов. Основное взаимодействие между сервисами происходит асинхронно через Apache Kafka, дополняется gRPC вызовами.

### Микросервисы

1. **Bot Service** - Управление Telegram ботом
   - Обработка команд пользователей
   - Взаимодействие с пользователями
   - Отправка новостей пользователям

2. **Subscription Service** - Управление подписками
   - Сага подписка/отписка через Kafka (`subscription.*`, `unsubscription.*`)
   - gRPC для получения подписок/подписчиков
   - База данных: PostgreSQL

3. **News Service** - Управление новостями
   - Хранение полученных новостей, история доставки
   - Маршрутизация новостей в bot-service
   - Обработка событий редактирования/удаления
   - База данных: PostgreSQL

4. **Account Service** - Управление Telegram аккаунтами
   - Управление несколькими Telegram аккаунтами (MTProto)
   - Подписка/отписка от каналов по саге
   - Сбор новостей и публикация в Kafka

### Схема взаимодействия

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│     Bot     │  Kafka  │ Subscription │  Kafka  │   Account    │
│   Service   │────────→│   Service    │────────→│   Service    │
└──────┬──────┘         └──────┬───────┘         └──────┬───────┘
       │                       │                        │
       │ gRPC                  │ gRPC                   │ Kafka
       │                       ↓                        │ news.received
       │                ┌──────────────┐                │
       └───────────────→│    News      │←───────────────┘
            Kafka       │   Service    │
         news.deliver   └──────────────┘
```

**Коммуникация:**
- **Kafka (подписки, сага):**
  - `subscription.requested` → `subscription.pending` → `subscription.activated|failed` → `subscription.confirmed|rejected`
  - `unsubscription.requested` → `unsubscription.pending` → `unsubscription.completed|failed` → `unsubscription.confirmed|rejected`
- **Kafka (новости):**
  - `news.received` (account-service → news-service)
  - `news.deliver` (news-service → bot-service)
  - `news.delivered` (bot-service → news-service)
  - `news.edited`/`news.deleted` (account-service → news-service) → `news.edit`/`news.delete` (news-service → bot-service)
- **gRPC:** GetUserSubscriptions, GetChannelSubscribers

### Kafka Topics (основные)

- `subscription.requested`, `subscription.pending`, `subscription.activated`, `subscription.failed`, `subscription.confirmed`, `subscription.rejected`
- `unsubscription.requested`, `unsubscription.pending`, `unsubscription.completed`, `unsubscription.failed`, `unsubscription.confirmed`, `unsubscription.rejected`
- `news.received`, `news.deliver`, `news.delivered`, `news.edited`, `news.deleted`, `news.edit`, `news.delete`

## Технологический стек

- **Язык**: Go 1.24+
- **DI Framework**: Uber FX
- **Telegram Bot API**: github.com/go-telegram/bot
- **Telegram MTProto**: github.com/gotd/td
- **Message Broker**: Apache Kafka
- **Inter-service**: gRPC + Protobuf
- **Базы данных**: PostgreSQL (GORM)
- **Логирование**: zerolog
- **Конфигурация**: Environment variables

## Структура проекта

```
.
├── services/
│   ├── bot-service/           # Сервис управления ботом
│   ├── subscription-service/  # Сервис подписок
│   ├── news-service/          # Сервис новостей
│   └── account-service/       # Сервис аккаунтов
├── pkg/                       # Общие библиотеки
├── deployments/
│   └── docker-compose.yml     # Инфраструктура для разработки
└── README.md
```

### Структура микросервиса

Каждый микросервис следует принципам Clean Architecture:

```
service-name/
├── cmd/app/              # Точка входа
├── internal/
│   ├── domain/           # Бизнес-сущности и интерфейсы
│   ├── usecase/          # Бизнес-логика
│   ├── delivery/         # Handlers (Telegram, Kafka)
│   ├── repository/       # Работа с данными
│   └── infrastructure/   # Внешние зависимости
├── config/               # Конфигурация
├── migrations/           # SQL миграции (если есть БД)
├── .env.example
├── Dockerfile
└── go.mod
```

## Установка и запуск

### Требования

- Go 1.24+
- Docker и Docker Compose
- PostgreSQL 15+
- Apache Kafka

### Локальная разработка

1. Клонировать репозиторий:
```bash
git clone <repository-url>
cd NewsFlow
```

2. Запустить инфраструктуру:
```bash
docker-compose -f deployments/docker-compose.yml up -d
```

3. Настроить переменные окружения для каждого сервиса:
```bash
# Пример для bot-service
cd services/bot-service
cp .env.example .env
# Отредактировать .env
```

4. Запустить сервисы:
```bash
# В отдельных терминалах
cd services/bot-service && go run ./cmd/app
cd services/subscription-service && go run ./cmd/app
cd services/news-service && go run ./cmd/app
cd services/account-service && go run ./cmd/app
```

## Конфигурация

Каждый сервис настраивается через переменные окружения. Примеры находятся в файлах `.env.example`.

### Основные переменные

- `TELEGRAM_BOT_TOKEN` - Токен Telegram бота (Bot Service)
- `TELEGRAM_API_ID` - API ID для MTProto (Account Service)
- `TELEGRAM_API_HASH` - API Hash для MTProto (Account Service)
- `DATABASE_URL` - URL подключения к PostgreSQL
- `KAFKA_BROKERS` - Список Kafka брокеров
- `LOG_LEVEL` - Уровень логирования (debug, info, warn, error)

## Разработка

### Принципы

1. **Clean Architecture** - разделение на слои (domain, usecase, delivery, repository)
2. **Dependency Injection** - внедрение зависимостей через конструкторы
3. **Interface-based design** - использование интерфейсов для абстракций
4. **Single Responsibility** - каждый модуль отвечает за одну задачу
5. **SOLID принципы**
