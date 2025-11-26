package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
)

// KafkaConsumerImpl реализует интерфейс domain.KafkaConsumer
type KafkaConsumerImpl struct {
	consumer sarama.ConsumerGroup
	logger   zerolog.Logger
	groupID  string
	topics   []string
}

// consumerGroupHandler реализует sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	logger  zerolog.Logger
	handler func(*domain.NewsMessage) error
}

// NewConsumer создает новый экземпляр Kafka consumer
func NewConsumer(cfg config.KafkaConfig, logger zerolog.Logger) (domain.KafkaConsumer, error) {
	// Если brokers не указаны, используем localhost по умолчанию
	if len(cfg.Brokers) == 0 {
		cfg.Brokers = []string{"localhost:9093"}
	}

	// Конфигурация Sarama consumer
	config := sarama.NewConfig()
	config.Version = sarama.V2_0_0_0 // Используем версию Kafka 2.0.0

	// Настройки consumer group
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
	}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	// Создаем consumer group
	consumer, err := sarama.NewConsumerGroup(cfg.Brokers, "bot-service-group", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("group_id", "bot-service-group").
		Strs("topics", []string{"news.deliver"}).
		Msg("Kafka consumer initialized successfully")

	return &KafkaConsumerImpl{
		consumer: consumer,
		logger:   logger,
		groupID:  "bot-service-group",
		topics:   []string{"news.deliver"},
	}, nil
}

// ConsumeNewsDelivery начинает потребление сообщений из Kafka
func (k *KafkaConsumerImpl) ConsumeNewsDelivery(ctx context.Context, handler func(*domain.NewsMessage) error) error {
	k.logger.Info().Msg("Starting Kafka consumer...")

	// Создаем handler для consumer group
	consumerHandler := &consumerGroupHandler{
		logger:  k.logger,
		handler: handler,
	}

	// Запускаем потребление в бесконечном цикле
	// (consumer group автоматически обрабатывает rebalance)
	for {
		select {
		case <-ctx.Done():
			k.logger.Info().Msg("Context cancelled - stopping consumer")
			return nil
		default:
			// Consume возвращает ошибку при rebalance, нужно перезапускать
			err := k.consumer.Consume(ctx, k.topics, consumerHandler)
			if err != nil {
				k.logger.Error().Err(err).Msg("Error in consumer loop")
				return fmt.Errorf("consumer error: %w", err)
			}
		}
	}
}

// Close закрывает consumer
func (k *KafkaConsumerImpl) Close() error {
	if k.consumer == nil {
		return nil
	}

	if err := k.consumer.Close(); err != nil {
		k.logger.Error().Err(err).Msg("Failed to close Kafka consumer")
		return err
	}

	k.logger.Info().Msg("Kafka consumer closed successfully")
	return nil
}

// ===== sarama.ConsumerGroupHandler implementation =====

// Setup вызывается когда consumer получает partition
func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.logger.Info().
		Int32("claims", int32(len(session.Claims()))).
		Msg("Consumer group setup completed")
	return nil
}

// Cleanup вызывается когда consumer теряет partition
func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.logger.Info().Msg("Consumer group cleanup completed")
	return nil
}

// ConsumeClaim обрабатывает сообщения из partition
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.logger.Info().
		Str("topic", claim.Topic()).
		Int32("partition", claim.Partition()).
		Msg("Starting to consume messages")

	for message := range claim.Messages() {
		select {
		case <-session.Context().Done():
			h.logger.Info().Msg("Session context cancelled - stopping consumption")
			return nil
		default:
			// Обрабатываем сообщение
			if err := h.processMessage(message); err != nil {
				h.logger.Error().
					Err(err).
					Str("topic", message.Topic).
					Int64("offset", message.Offset).
					Msg("Failed to process message")
				// Продолжаем обработку следующих сообщений несмотря на ошибку
				continue
			}

			// Помечаем сообщение как обработанное
			session.MarkMessage(message, "")
			session.MarkOffset(message.Topic, message.Partition, message.Offset+1, "")

			h.logger.Debug().
				Str("topic", message.Topic).
				Int32("partition", message.Partition).
				Int64("offset", message.Offset).
				Msg("Message processed successfully")
		}
	}

	return nil
}

// processMessage обрабатывает одно Kafka сообщение
func (h *consumerGroupHandler) processMessage(message *sarama.ConsumerMessage) error {
	h.logger.Debug().
		Str("topic", message.Topic).
		Int64("offset", message.Offset).
		Int32("partition", message.Partition).
		Msg("Processing Kafka message")

	// Десериализуем JSON в domain.NewsMessage
	var newsMessage domain.NewsMessage
	if err := json.Unmarshal(message.Value, &newsMessage); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Валидируем обязательные поля
	if newsMessage.UserID == 0 {
		return fmt.Errorf("invalid user_id in message")
	}
	if newsMessage.Content == "" {
		return fmt.Errorf("empty content in message")
	}

	h.logger.Info().
		Int64("user_id", newsMessage.UserID).
		Str("channel_id", newsMessage.ChannelID).
		Str("news_id", newsMessage.ID).
		Msg("Successfully parsed news message")

	// Вызываем переданный handler
	if err := h.handler(&newsMessage); err != nil {
		return fmt.Errorf("handler failed: %w", err)
	}

	return nil
}
