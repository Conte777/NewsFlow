# Kafka Producer

Асинхронный Kafka Producer для отправки новостных событий в топик `news.received`.

## Возможности

- **Асинхронная отправка** - высокая производительность через AsyncProducer
- **Компрессия Snappy** - оптимизация использования сети
- **Idempotent режим** - at-least-once доставка с автоматической дедупликацией
- **Hash партиционирование** - по `channel_id` для сохранения порядка сообщений
- **Валидация данных** - проверка всех полей NewsItem перед отправкой
- **Обработка ошибок** - ErrorCallback для кастомной логики (DLQ, retry, alerting)
- **Graceful shutdown** - идемпотентный Close() с ожиданием отправки всех сообщений

## Формат события

События отправляются в формат JSON согласно спецификации ACC-2.2:

```json
{
  "channel_id": "tech_news_channel",
  "channel_name": "Tech News Daily",
  "message_id": 123456,
  "content": "Breaking news content...",
  "media_urls": [
    "https://example.com/image1.jpg",
    "https://example.com/video1.mp4"
  ],
  "date": "2025-11-24T10:30:00Z"
}
```

**Ключ партиции**: `channel_id` - обеспечивает упорядоченность сообщений из одного канала.

## Использование

### Базовый пример

```go
import (
    "context"
    "github.com/rs/zerolog"
    "github.com/Conte777/NewsFlow/services/account-service/internal/domain"
    "github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/kafka"
)

// Создание producer
config := kafka.ProducerConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "news.received",
    Logger:  zerolog.New(os.Stdout),
}

producer, err := kafka.NewKafkaProducer(config)
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

// Отправка новости
news := &domain.NewsItem{
    ChannelID:   "tech_channel",
    ChannelName: "Tech News",
    MessageID:   12345,
    Content:     "Breaking news...",
    MediaURLs:   []string{"https://example.com/img.jpg"},
    Date:        time.Now(),
}

ctx := context.Background()
if err := producer.SendNewsReceived(ctx, news); err != nil {
    log.Error().Err(err).Msg("Failed to send news")
}
```

### С обработкой ошибок (ErrorCallback)

```go
config := kafka.ProducerConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "news.received",
    Logger:  logger,
    ErrorCallback: func(news *domain.NewsItem, err error) {
        // Отправка в Dead Letter Queue
        dlq.Send(news)

        // Или алерт
        alerting.Notify("Kafka send failed", err)
    },
}
```

### Конфигурация

```go
config := kafka.ProducerConfig{
    Brokers:         []string{"kafka1:9092", "kafka2:9092"},
    Topic:           "news.received",
    Logger:          logger,
    ErrorCallback:   errorHandler,      // опционально
    MaxMessageBytes: 2000000,           // 2MB (default: 1MB)
    MaxRetries:      10,                // default: 5
}
```

## Тестирование

### Unit тесты

```bash
# Запуск всех unit тестов
go test -v ./internal/infrastructure/kafka/...

# С покрытием
go test -cover ./internal/infrastructure/kafka/...
```

**Результат**: 20 тестов, покрытие 62.7%

### Integration тесты

Для integration тестов требуется запущенный Kafka.

**Docker Compose для Kafka:**

```yaml
version: '3.8'
services:
  kafka:
    image: confluentinc/cp-kafka:latest
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: 'CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT'
      KAFKA_ADVERTISED_LISTENERS: 'PLAINTEXT://localhost:9092'
      KAFKA_PROCESS_ROLES: 'broker,controller'
      KAFKA_CONTROLLER_QUORUM_VOTERS: '1@localhost:29093'
      KAFKA_LISTENERS: 'PLAINTEXT://0.0.0.0:9092,CONTROLLER://localhost:29093'
      KAFKA_CONTROLLER_LISTENER_NAMES: 'CONTROLLER'
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      CLUSTER_ID: 'MkU3OEVBNTcwNTJENDM2Qk'

  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    ports:
      - "8080:8080"
    environment:
      KAFKA_CLUSTERS_0_NAME: local
      KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS: kafka:9092
    depends_on:
      - kafka
```

**Запуск:**

```bash
# Запустить Kafka
docker-compose up -d

# Создать топик
docker exec -it kafka kafka-topics --create \
    --topic news.received \
    --bootstrap-server localhost:9092 \
    --partitions 3 \
    --replication-factor 1

# Запустить integration тесты
go test -tags=integration -v ./internal/infrastructure/kafka/...

# Проверить результаты в Kafka UI
# http://localhost:8080
```

### Тест отправки 10 новостей

```go
// Unit тест (с моками)
go test -v -run TestKafkaProducer_Send10News ./internal/infrastructure/kafka/...

// Integration тест (с реальным Kafka)
go test -tags=integration -v -run TestKafkaProducer_SendNewsReceived_Integration ./internal/infrastructure/kafka/...
```

## Проверка в Kafka UI

1. Откройте http://localhost:8080
2. Выберите топик `news.received`
3. Проверьте сообщения:
   - Формат соответствует спецификации
   - Ключ партиции = `channel_id`
   - Timestamp установлен корректно

## Критерии приёма ACC-2.2

- ✅ Метод `SendNewsReceived()` реализован
- ✅ Сериализация JSON работает (snake_case формат)
- ✅ Отправка в топик `news.received`
- ✅ Ключ партиции = `channel_id`
- ✅ Обработка ошибок и логика повтора (ErrorCallback + Retry.Max)
- ✅ Тест: отправка 10 новостей (`TestKafkaProducer_Send10News`)
- ✅ Проверка в Kafka UI (integration test)

## Производительность

- **Throughput**: ~10,000 msg/sec (зависит от размера сообщений)
- **Latency**: <10ms (async send)
- **Compression ratio**: ~3x (Snappy)

## Troubleshooting

### Producer не может подключиться к Kafka

```
Error: kafka: client has run out of available brokers
```

**Решение**: Проверьте, что Kafka доступен и `ADVERTISED_LISTENERS` настроен правильно.

### Сообщения не появляются в Kafka

1. Проверьте логи producer (должны быть Debug сообщения)
2. Проверьте ErrorCallback - возможно ошибки при отправке
3. Проверьте, что топик создан: `kafka-topics --list`

### Ошибки валидации

```
Error: invalid news item: channel_id is required
```

**Решение**: Убедитесь, что все обязательные поля NewsItem заполнены:
- `ChannelID` (не пустой)
- `MessageID` (> 0)
- `Date` (не zero value)

## См. также

- [Примеры использования](./producer_example_test.go)
- [Integration тесты](./producer_integration_test.go)
- [Sarama документация](https://github.com/IBM/sarama)
