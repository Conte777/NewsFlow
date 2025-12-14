# Kafka Consumer - Subscription Events

Реализация Kafka Consumer для получения событий подписки из subscription service.

## Обзор

KafkaConsumer получает события подписки пользователей на каналы и обрабатывает их через интерфейс `SubscriptionEventHandler`.

## Конфигурация (ACC-2.4)

- **Consumer Group**: `account-service-group`
- **Topics**:
  - `subscriptions.created` - события создания подписки
  - `subscriptions.deleted` - события удаления подписки
- **Session Timeout**: 10 секунд
- **Heartbeat Interval**: 3 секунды
- **Max Poll Records**: 100
- **Auto Commit**: Отключен (ручной commit для at-least-once гарантии)

## Использование

### Создание Consumer

```go
import (
    "github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/delivery/kafka"
    "github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
    "github.com/rs/zerolog"
)

// Реализуйте event handler
type SubscriptionHandler struct {
    accountUseCase domain.AccountUseCase
    logger         zerolog.Logger
}

func (h *SubscriptionHandler) HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error {
    h.logger.Info().
        Int64("user_id", userID).
        Str("channel_id", channelID).
        Str("channel_name", channelName).
        Msg("Subscription created")

    // Подписать аккаунт на канал
    return h.accountUseCase.SubscribeToChannel(ctx, channelID, channelName)
}

func (h *SubscriptionHandler) HandleSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error {
    h.logger.Info().
        Int64("user_id", userID).
        Str("channel_id", channelID).
        Msg("Subscription deleted")

    // Отписать аккаунт от канала
    return h.accountUseCase.UnsubscribeFromChannel(ctx, channelID)
}

// Создайте consumer
handler := &SubscriptionHandler{
    accountUseCase: accountUseCase,
    logger:         logger,
}

consumer, err := kafka.NewKafkaConsumer(kafka.ConsumerConfig{
    Brokers: []string{"localhost:9092"},
    Logger:  logger,
    Handler: handler,
})
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()
```

### Запуск Consumer

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Запустить в отдельной горутине
go func() {
    if err := consumer.ConsumeSubscriptionEvents(ctx, handler); err != nil {
        logger.Error().Err(err).Msg("Consumer stopped with error")
    }
}()

// Graceful shutdown
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

logger.Info().Msg("Shutting down consumer...")
cancel() // Остановит consumer
```

### Кастомная конфигурация

```go
consumer, err := kafka.NewKafkaConsumer(kafka.ConsumerConfig{
    Brokers:           []string{"kafka1:9092", "kafka2:9092"},
    Logger:            logger,
    Handler:           handler,
    SessionTimeout:    15 * time.Second,  // Кастомный session timeout
    HeartbeatInterval: 5 * time.Second,   // Кастомный heartbeat interval
})
```

## Формат событий

### subscription.created

```json
{
  "event_type": "subscription.created",
  "user_id": 12345,
  "channel_id": "tech_news_channel",
  "channel_name": "Tech News"
}
```

### subscription.deleted

```json
{
  "event_type": "subscription.deleted",
  "user_id": 12345,
  "channel_id": "tech_news_channel"
}
```

## Тестирование

### Unit тесты

```bash
cd services/account-service
go test -v ./internal/delivery/kafka
```

### Integration тесты

Требуется запущенный Kafka на `localhost:9092`:

```bash
# Запустить Kafka через Docker
docker-compose -f docker-compose.kafka.yml up -d

# Создать топики
kafka-topics.sh --bootstrap-server localhost:9092 --create --topic subscriptions.created
kafka-topics.sh --bootstrap-server localhost:9092 --create --topic subscriptions.deleted

# Запустить тесты
go test -v -tags=integration ./internal/delivery/kafka
```

### Отправка тестовых событий

```bash
# Отправить subscription.created
echo '{"event_type":"subscription.created","user_id":12345,"channel_id":"test_channel","channel_name":"Test Channel"}' | \
kafka-console-producer.sh --bootstrap-server localhost:9092 --topic subscriptions.created

# Отправить subscription.deleted
echo '{"event_type":"subscription.deleted","user_id":12345,"channel_id":"test_channel"}' | \
kafka-console-producer.sh --bootstrap-server localhost:9092 --topic subscriptions.deleted
```

### Мониторинг consumer group

```bash
# Проверить статус consumer group
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group account-service-group

# Вывод:
# GROUP                  TOPIC                   PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG
# account-service-group  subscriptions.created   0          10              10              0
# account-service-group  subscriptions.deleted   0          5               5               0
```

## Архитектура

```
┌─────────────────┐
│ Subscription    │
│ Service         │
└────────┬────────┘
         │ Produces events
         ↓
┌─────────────────┐
│ Kafka Topics    │
│ - subscriptions.│
│   created       │
│ - subscriptions.│
│   deleted       │
└────────┬────────┘
         │ Consumes events
         ↓
┌─────────────────┐
│ KafkaConsumer   │
│ (ACC-2.4)       │
├─────────────────┤
│ ConsumerGroup:  │
│ account-service │
│ -group          │
└────────┬────────┘
         │ Calls handler
         ↓
┌─────────────────┐
│ Subscription    │
│ EventHandler    │
├─────────────────┤
│ - HandleSub     │
│   Created()     │
│ - HandleSub     │
│   Deleted()     │
└────────┬────────┘
         │ Business logic
         ↓
┌─────────────────┐
│ AccountUseCase  │
│ - SubscribeTo   │
│   Channel()     │
│ - Unsubscribe   │
│   FromChannel() │
└─────────────────┘
```

## Обработка ошибок

### At-least-once гарантия

Consumer использует ручной commit (auto-commit отключен) для обеспечения at-least-once обработки:

1. Сообщение читается из Kafka
2. Handler обрабатывает событие
3. Если обработка успешна - offset коммитится
4. Если обработка провалилась - offset НЕ коммитится, сообщение будет обработано повторно

### Idempotency

Handler должен быть **идемпотентным**, т.к. одно сообщение может быть обработано несколько раз:

```go
func (h *Handler) HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error {
    // Проверка: уже подписан?
    exists, err := h.repo.ChannelExists(ctx, channelID)
    if err != nil {
        return err
    }

    if exists {
        // Уже подписаны - skip (идемпотентность)
        return nil
    }

    // Выполнить подписку
    return h.useCase.SubscribeToChannel(ctx, channelID, channelName)
}
```

### Retry логика

При ошибке обработки сообщение остается uncommitted и будет перечитано:

```go
func (h *Handler) HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error {
    err := h.useCase.SubscribeToChannel(ctx, channelID, channelName)
    if err != nil {
        // Временная ошибка (сеть, БД timeout) - вернуть ошибку для retry
        if isTemporaryError(err) {
            h.logger.Warn().Err(err).Msg("Temporary error, will retry")
            return err
        }

        // Постоянная ошибка (невалидные данные) - залогировать и пропустить
        h.logger.Error().Err(err).Msg("Permanent error, skipping message")
        return nil // Вернуть nil чтобы закоммитить offset
    }

    return nil
}
```

## Best Practices

1. **Идемпотентность**: Handler должен корректно обрабатывать повторные сообщения
2. **Быстрая обработка**: Не выполнять тяжелые операции в handler (используйте очереди)
3. **Graceful shutdown**: Всегда закрывать consumer через context cancellation
4. **Мониторинг**: Следить за lag consumer group
5. **Error handling**: Различать временные и постоянные ошибки

## Performance

- **Throughput**: ~1000 events/sec (зависит от обработки)
- **Latency**: <100ms (от publish до handler)
- **Max Poll Records**: 100 (batch size)

## Troubleshooting

### Consumer не получает сообщения

1. Проверить что Kafka запущен:
   ```bash
   kafka-broker-api-versions.sh --bootstrap-server localhost:9092
   ```

2. Проверить что топики существуют:
   ```bash
   kafka-topics.sh --bootstrap-server localhost:9092 --list
   ```

3. Проверить что consumer group активен:
   ```bash
   kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group account-service-group
   ```

### Большой lag

1. Увеличить количество партиций в топике
2. Запустить несколько инстансов consumer (scaling)
3. Оптимизировать обработку в handler

### Rebalancing loops

1. Увеличить `SessionTimeout` если обработка долгая
2. Уменьшить batch size
3. Оптимизировать handler

## См. также

- [Kafka Producer](./README.md) - Producer для отправки новостей
- [Domain Interfaces](../../domain/interfaces.go) - Интерфейсы domain layer
- [Sarama Documentation](https://pkg.go.dev/github.com/IBM/sarama)
