
## 1. Основной README.md

**`services/bot-service/README.md`**

```markdown
# Bot Service

Сервис Telegram бота для управления подписками на новостные каналы.

## Описание сервиса

Bot Service предоставляет функциональность Telegram бота, который позволяет пользователям:
- Подписываться на новостные каналы
- Отписываться от каналов
- Просматривать текущие подписки
- Получать новости из подписанных каналов

Сервис интегрируется с Apache Kafka для обработки событий подписок и доставки новостей.

## Архитектура и компоненты

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Telegram API  │◄──►│   Bot Service    │◄──►│   Apache Kafka  │
│                 │    │                  │    │                 │
│ - Получение     │    │ - Обработка      │    │ - События       │
│   сообщений     │    │   команд         │    │   подписок      │
│ - Отправка      │    │ - Управление     │    │ - Новостные     │
│   ответов       │    │   подписками     │    │   сообщения     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   База данных   │
                       │                 │
                       │ - Подписки      │
                       │ - Пользователи  │
                       └─────────────────┘
```

### Основные компоненты

- **Delivery Layer** (`internal/delivery/telegram`) - обработка входящих сообщений Telegram
- **Use Case Layer** (`internal/usecase`) - бизнес-логика приложения
- **Domain Layer** (`internal/domain`) - доменные модели и интерфейсы
- **Infrastructure Layer** (`internal/infrastructure`) - реализация внешних зависимостей

## Запуск в dev режиме

### Предварительные требования

- Go 1.19+
- Apache Kafka
- Telegram Bot Token

### Шаги для запуска

1. **Клонируйте репозиторий**
   ```bash
   git clone https://github.com/Conte777/NewsFlow.git
   cd NewsFlow/services/bot-service
   ```

2. **Установите зависимости**
   ```bash
   go mod download
   ```

3. **Настройте переменные окружения**
   ```bash
   cp .env.example .env
   # Отредактируйте .env файл
   ```

4. **Запустите сервис**
   ```bash
   go run cmd/bot-service/main.go
   ```

### Локальная разработка с Docker

```bash
# Запуск зависимостей (Kafka, Zookeeper)
docker-compose up -d kafka zookeeper

# Запуск сервиса
docker-compose up bot-service
```

## Запуск в Docker

### Production сборка

```bash
# Сборка образа
docker build -t newsflow/bot-service:latest .

# Запуск контейнера
docker run -d \
  --name bot-service \
  --env-file .env \
  -p 8080:8080 \
  newsflow/bot-service:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  bot-service:
    image: newsflow/bot-service:latest
    environment:
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - KAFKA_BROKERS=kafka:9092
      - LOG_LEVEL=info
    depends_on:
      - kafka
    restart: unless-stopped
```

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|-----------|
| `TELEGRAM_BOT_TOKEN` | ✅ | - | Токен Telegram бота |
| `KAFKA_BROKERS` | ✅ | `localhost:9092` | Список Kafka брокеров |
| `KAFKA_SUBSCRIPTION_TOPIC` | ❌ | `subscription-events` | Топик для событий подписок |
| `KAFKA_NEWS_TOPIC` | ❌ | `news-events` | Топик для новостных событий |
| `LOG_LEVEL` | ❌ | `info` | Уровень логирования |
| `HTTP_PORT` | ❌ | `8080` | Порт для health checks |

## Команды Telegram бота

### Основные команды

- `/start` - Начать работу с ботом
- `/help` - Получить справку по командам
- `/subscribe @channel1 @channel2` - Подписаться на каналы
- `/unsubscribe @channel1 @channel2` - Отписаться от каналов
- `/list` - Показать текущие подписки

### Примеры использования

```
/start
→ Добро пожаловать! Ваш ID: 12345

/subscribe @news @technology
→ ✅ Успешно подписались на каналы: @news, @technology

/list
→ 📋 Ваши подписки:
   • @news
   • @technology

/unsubscribe @news
→ ✅ Успешно отписались от каналов: @news
```

## Структура проекта

```
services/bot-service/
├── cmd/
│   └── bot-service/
│       └── main.go                 # Точка входа
├── internal/
│   ├── delivery/
│   │   └── telegram/
│   │       └── handler.go          # Обработчик Telegram сообщений
│   ├── usecase/
│   │   └── bot_usecase.go          # Бизнес-логика
│   ├── domain/
│   │   └── interfaces.go           # Интерфейсы и модели
│   └── infrastructure/
│       ├── kafka/
│       └── telegram/
├── config/
│   └── config.go                   # Конфигурация
├── docs/                           # Документация
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Contributing Guidelines

### Code Style

- Используйте `go fmt` для форматирования кода
- Следуйте принципам Clean Architecture
- Пишите тесты для новой функциональности
- Используйте structured logging с zerolog

### Git Workflow

1. Создайте feature branch от `develop`
2. Регулярно делайте коммиты с понятными сообщениями
3. Открывайте Pull Request для ревью
4. Убедитесь, что все тесты проходят

### Тестирование

```bash
# Запуск всех тестов
go test ./...

# Запуск тестов с покрытием
go test -cover ./...

# Запуск интеграционных тестов
go test -tags=integration ./...
```

## Troubleshooting

### Частые проблемы

**Бот не отвечает на сообщения**
- Проверьте `TELEGRAM_BOT_TOKEN`
- Убедитесь, что бот не заблокирован пользователем

**Ошибки подключения к Kafka**
- Проверьте доступность Kafka брокеров
- Убедитесь в правильности топиков

**Высокая загрузка памяти**
- Проверьте логи на утечки памяти
- Увеличьте лимиты контейнера при необходимости

### Логирование

Уровни логирования:
- `debug` - детальная отладочная информация
- `info` - основная информация о работе
- `warn` - предупреждения
- `error` - ошибки, требующие внимания

### Health Checks

```bash
# Проверка здоровья сервиса
curl http://localhost:8080/health

# Метрики Prometheus
curl http://localhost:8080/metrics
```

## Лицензия

MIT License - смотрите файл [LICENSE](../LICENSE) для деталей.
```

## 2. Документация по Kafka событиям

**`docs/KAFKA_EVENTS.md`**

```markdown
# Kafka Events Documentation

Документация по событиям Apache Kafka в Bot Service.

## Overview

Bot Service использует Kafka для асинхронной обработки событий подписок и доставки новостей.

## Топики

### subscription-events
**Назначение**: События создания и удаления подписок

**Формат сообщения**: JSON

**Пример конфигурации**:
```json
{
  "num_partitions": 3,
  "replication_factor": 2,
  "retention_ms": 604800000
}
```

### news-events
**Назначение**: Доставка новостей пользователям

**Формат сообщения**: JSON

**Пример конфигурации**:
```json
{
  "num_partitions": 6,
  "replication_factor": 2,
  "retention_ms": 86400000
}
```

## Схемы событий

### Subscription Event

Событие подписки/отписки пользователя.

```json
{
  "user_id": 123456789,
  "channels": ["@news", "@technology"],
  "event_type": "subscribe",
  "action": "subscribe",
  "timestamp": 1633046400000
}
```

**Поля**:
- `user_id` (int64) - ID пользователя Telegram
- `channels` ([]string) - Список каналов
- `event_type` (string) - Тип события: `subscribe` или `unsubscribe`
- `action` (string) - Действие: `subscribe` или `unsubscribe`
- `timestamp` (int64) - Временная метка события

### News Message Event

Событие доставки новости пользователю.

```json
{
  "id": "news-12345-abcde",
  "user_id": 123456789,
  "channel_id": "@news",
  "content": "Заголовок новости\n\nТекст новости...",
  "timestamp": 1633046400000
}
```

**Поля**:
- `id` (string) - Уникальный идентификатор новости
- `user_id` (int64) - ID пользователя Telegram
- `channel_id` (string) - ID канала-источника
- `content` (string) - Содержание новости
- `timestamp` (int64) - Временная метка публикации

## Producer Configuration

### Bot Service Producer

```go
type KafkaConfig struct {
    Brokers          []string `json:"brokers"`
    SubscriptionTopic string   `json:"subscription_topic"`
    NewsTopic        string   `json:"news_topic"`
    ClientID         string   `json:"client_id"`
    CompressionType  string   `json:"compression_type"` // "none", "gzip", "snappy", "lz4"
}
```

### Рекомендуемые настройки

```yaml
kafka:
  brokers:
    - "kafka1:9092"
    - "kafka2:9092"
    - "kafka3:9092"
  subscription_topic: "subscription-events"
  news_topic: "news-events"
  compression: "snappy"
  batch_size: 100000
  linger_ms: 10
```

## Consumer Groups

### News Delivery Consumer

**Group ID**: `bot-service-news-delivery`

**Назначение**: Получение новостей для доставки пользователям

**Распределение**: По partition key = `user_id`

### Пример обработки

```go
func (h *NewsHandler) Handle(message *NewsMessage) error {
    // Валидация сообщения
    if err := validateNewsMessage(message); err != nil {
        return fmt.Errorf("invalid news message: %w", err)
    }
    
    // Отправка через Telegram Bot API
    return h.botUseCase.SendNews(context.Background(), message)
}
```

## Обработка ошибок

### Retry Policy

- **Максимум попыток**: 3
- **Backoff стратегия**: Exponential
- **Initial delay**: 1 секунда
- **Max delay**: 30 секунд

### Dead Letter Queue

Необрабатываемые сообщения отправляются в DLQ:

- `subscription-events-dlq`
- `news-events-dlq`

### Мониторинг

Ключевые метрики для мониторинга:

- `kafka_producer_errors_total`
- `kafka_consumer_errors_total`
- `kafka_message_processing_duration_seconds`
- `kafka_dlq_messages_total`

## Примеры кода

### Producing Events

```go
func (p *KafkaProducer) SendSubscriptionEvent(ctx context.Context, event *SubscriptionEvent) error {
    message := &kafka.Message{
        Topic: p.subscriptionTopic,
        Key:   []byte(fmt.Sprintf("%d", event.UserID)),
        Value: event.ToJSON(),
    }
    
    return p.producer.Produce(message)
}
```

### Consuming Events

```go
func (c *KafkaConsumer) ConsumeNewsDelivery(ctx context.Context) error {
    return c.consumer.Consume(ctx, func(message *kafka.Message) error {
        var news NewsMessage
        if err := json.Unmarshal(message.Value, &news); err != nil {
            return fmt.Errorf("parse news message: %w", err)
        }
        
        return c.handler(&news)
    })
}
```

## Миграции схем

При изменении схемы событий:

1. Добавляйте новые поля (не удаляйте старые)
2. Используйте значения по умолчанию для новых полей
3. Обновите документацию
4. Уведомите потребителей событий

## Best Practices

1. **Идемпотентность**: Обработчики должны быть идемпотентными
2. **Валидация**: Всегда валидируйте входящие сообщения
3. **Мониторинг**: Настройте алерты на ошибки и задержки
4. **Тестирование**: Пишите тесты для обработчиков событий
```

## 3. API документация

**`docs/API.md`**

```markdown
# API Documentation

Документация API Bot Service.

## Overview

Bot Service предоставляет REST API для мониторинга и управления, а также обрабатывает сообщения через Telegram Bot API.

## REST Endpoints

### Health Check

**GET /health**

Проверка состояния сервиса и его зависимостей.

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2023-10-01T12:00:00Z",
  "dependencies": {
    "kafka": "connected",
    "telegram": "connected",
    "database": "connected"
  }
}
```

**Status Codes**:
- `200 OK` - Сервис здоров
- `503 Service Unavailable` - Проблемы с зависимостями

### Metrics

**GET /metrics**

Метрики Prometheus для мониторинга.

**Response**: Prometheus text format

### Ready Check

**GET /ready**

Проверка готовности сервиса к работе.

**Response**:
```json
{
  "status": "ready",
  "timestamp": "2023-10-01T12:00:00Z"
}
```

## Domain Interfaces

### BotUseCase Interface

```go
type BotUseCase interface {
    // HandleStart обрабатывает команду /start
    HandleStart(ctx context.Context, userID int64, username string) (string, error)
    
    // HandleHelp обрабатывает команду /help  
    HandleHelp(ctx context.Context) (string, error)
    
    // HandleSubscribe обрабатывает подписку на канал
    HandleSubscribe(ctx context.Context, userID int64, channelID string) (string, error)
    
    // HandleUnsubscribe обрабатывает отписку от канала
    HandleUnsubscribe(ctx context.Context, userID int64, channelID string) (string, error)
    
    // HandleListSubscriptions возвращает список подписок
    HandleListSubscriptions(ctx context.Context, userID int64) ([]Subscription, error)
    
    // SendNews отправляет новость пользователю
    SendNews(ctx context.Context, news *NewsMessage) error
    
    // SetTelegramBot устанавливает Telegram бота
    SetTelegramBot(bot TelegramBot)
    
    // HealthCheck проверяет здоровье зависимостей
    HealthCheck(ctx context.Context) error
}
```

### KafkaProducer Interface

```go
type KafkaProducer interface {
    // SendSubscriptionCreated отправляет событие создания подписки
    SendSubscriptionCreated(ctx context.Context, subscription *Subscription) error
    
    // SendSubscriptionDeleted отправляет событие удаления подписки
    SendSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error
    
    // SendSubscriptionEvent отправляет событие подписки (универсальное)
    SendSubscriptionEvent(ctx context.Context, event *SubscriptionEvent) error
    
    // SendNewsEvent отправляет новостное событие
    SendNewsEvent(ctx context.Context, news *NewsMessage) error
    
    // Close закрывает продюсер
    Close() error
}
```

### TelegramBot Interface

```go
type TelegramBot interface {
    // SendMessage отправляет текстовое сообщение
    SendMessage(ctx context.Context, userID int64, text string) error
    
    // SendMessageWithMedia отправляет сообщение с медиа
    SendMessageWithMedia(ctx context.Context, userID int64, text string, mediaURLs []string) error
    
    // Start запускает бота
    Start(ctx context.Context) error
    
    // Stop останавливает бота
    Stop() error
}
```

## Data Models

### Subscription

```go
type Subscription struct {
    UserID      int64  `json:"user_id"`
    ChannelID   string `json:"channel_id"`
    ChannelName string `json:"channel_name"`
    CreatedAt   int64  `json:"created_at"`
}
```

### NewsMessage

```go
type NewsMessage struct {
    ID        string `json:"id"`
    UserID    int64  `json:"user_id"`
    ChannelID string `json:"channel_id"`
    Content   string `json:"content"`
    Timestamp int64  `json:"timestamp"`
}
```

### SubscriptionEvent

```go
type SubscriptionEvent struct {
    UserID    int64    `json:"user_id"`
    Channels  []string `json:"channels"`
    EventType string   `json:"event_type"`
    Action    string   `json:"action"`
    Timestamp int64    `json:"timestamp"`
}
```

## Error Handling

### Error Types

- **ValidationError** - Ошибки валидации входных данных
- **NetworkError** - Сетевые ошибки (Kafka, Telegram API)
- **TimeoutError** - Таймауты операций
- **InternalError** - Внутренние ошибки сервиса

### Error Responses

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid channel format",
    "details": {
      "field": "channel_id",
      "reason": "must start with @"
    },
    "timestamp": "2023-10-01T12:00:00Z"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Ошибка валидации входных данных |
| `TELEGRAM_API_ERROR` | 502 | Ошибка Telegram API |
| `KAFKA_ERROR` | 503 | Ошибка подключения к Kafka |
| `TIMEOUT_ERROR` | 504 | Таймаут операции |
| `INTERNAL_ERROR` | 500 | Внутренняя ошибка сервиса |

## Configuration

### Environment Variables

```go
type Config struct {
    Telegram struct {
        BotToken string `env:"TELEGRAM_BOT_TOKEN,required"`
        Timeout  int    `env:"TELEGRAM_TIMEOUT" envDefault:"30"`
    }
    
    Kafka struct {
        Brokers          []string `env:"KAFKA_BROKERS" envSeparator:","`
        SubscriptionTopic string   `env:"KAFKA_SUBSCRIPTION_TOPIC" envDefault:"subscription-events"`
        NewsTopic        string   `env:"KAFKA_NEWS_TOPIC" envDefault:"news-events"`
    }
    
    Server struct {
        Port    int    `env:"HTTP_PORT" envDefault:"8080"`
        Timeout int    `env:"HTTP_TIMEOUT" envDefault:"30"`
    }
    
    Log struct {
        Level string `env:"LOG_LEVEL" envDefault:"info"`
    }
}
```

## Message Flow

### Subscription Flow

1. **Пользователь** отправляет `/subscribe @channel`
2. **Telegram Handler** парсит команду
3. **Bot UseCase** валидирует канал и пользователя
4. **Kafka Producer** отправляет событие подписки
5. **Response Handler** возвращает результат пользователю

### News Delivery Flow

1. **Kafka Consumer** получает новостное сообщение
2. **News Handler** валидирует и обрабатывает сообщение
3. **Bot UseCase** форматирует сообщение для Telegram
4. **Telegram Bot** отправляет сообщение пользователю

## Rate Limiting

### Telegram API Limits

- **Messages**: 30 сообщений в секунду
- **Media**: 20 сообщений в секунду
- **Broadcast**: 30 сообщений в секунду

### Implementation

```go
type RateLimiter struct {
    messages *rate.Limiter
    media    *rate.Limiter
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        messages: rate.NewLimiter(30, 30), // 30 messages per second
        media:    rate.NewLimiter(20, 20), // 20 media messages per second
    }
}
```

## Testing

### Unit Tests

```go
func TestBotUseCase_HandleSubscribe(t *testing.T) {
    // Setup
    mockProducer := new(MockKafkaProducer)
    mockBot := new(MockTelegramBot)
    useCase := NewBotUseCase(mockProducer, mockBot, logger)
    
    // Test
    result, err := useCase.HandleSubscribe(context.Background(), 123, "@news")
    
    // Assert
    assert.NoError(t, err)
    assert.Contains(t, result, "Успешно подписались")
}
```

### Integration Tests

```go
func TestSubscriptionFlow(t *testing.T) {
    // Setup test environment
    // Send test message
    // Verify Kafka event
    // Check response
}
```

## Deployment

### Health Checks

```yaml
# Kubernetes liveness probe
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

# Kubernetes readiness probe  
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Resource Limits

```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "200m"
```

## Monitoring

### Key Metrics

- `bot_messages_processed_total`
- `bot_subscriptions_total`
- `kafka_messages_produced_total`
- `telegram_api_errors_total`
- `request_duration_seconds`

### Alerting Rules

```yaml
groups:
- name: bot-service
  rules:
  - alert: BotServiceDown
    expr: up{job="bot-service"} == 0
    for: 5m
  - alert: HighErrorRate
    expr: rate(telegram_api_errors_total[5m]) > 0.1
    for: 2m
```

## Changelog

### v1.0.0 (2023-10-01)
- Initial release
- Basic subscription management
- Kafka integration
- Health checks
```

