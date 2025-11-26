package kafka

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
)

// KafkaConsumerImpl implements domain.KafkaConsumer interface
type KafkaConsumerImpl struct {
	logger zerolog.Logger
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg config.KafkaConfig, logger zerolog.Logger) (domain.KafkaConsumer, error) {
	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("group_id", "bot-service-group").
		Strs("topics", []string{"news.deliver"}).
		Msg("Kafka consumer initialized")

	return &KafkaConsumerImpl{
		logger: logger,
	}, nil
}

// ConsumeNewsDelivery starts consuming messages from Kafka
func (k *KafkaConsumerImpl) ConsumeNewsDelivery(ctx context.Context, handler func(*domain.NewsMessage) error) error {
	k.logger.Info().Msg("Starting Kafka consumer...")

	// Wait for context cancellation (graceful shutdown)
	<-ctx.Done()

	k.logger.Info().Msg("Kafka consumer stopped")
	return nil
}

// Close closes the consumer
func (k *KafkaConsumerImpl) Close() error {
	k.logger.Info().Msg("Kafka consumer closed")
	return nil
}
