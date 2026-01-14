package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
)

type KafkaProducerImpl struct {
	producer sarama.SyncProducer
	logger   zerolog.Logger
}

func NewProducer(cfg config.KafkaConfig, logger zerolog.Logger) (domain.KafkaProducer, error) {
	if len(cfg.Brokers) == 0 {
		cfg.Brokers = []string{"localhost:9093"}
	}

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true
	config.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	logger.Info().Strs("brokers", cfg.Brokers).Msg("Kafka producer initialized successfully")

	return &KafkaProducerImpl{
		producer: producer,
		logger:   logger,
	}, nil
}

// ИСПРАВЛЕНО: убрал * перед domain.Subscription
func (k *KafkaProducerImpl) SendSubscriptionCreated(ctx context.Context, subscription *domain.Subscription) error {
	event := map[string]interface{}{
		"user_id":      subscription.UserID,      // ← subscription теперь указатель, но это нормально
		"channel_id":   subscription.ChannelID,   // ← Go автоматически разыменовывает
		"channel_name": subscription.ChannelName, // ← структуры при доступе к полям
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}
	return k.sendEvent(ctx, "subscriptions.created", event)
}

func (k *KafkaProducerImpl) SendSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error {
	event := map[string]interface{}{
		"user_id":    userID,
		"channel_id": channelID,
		"deleted_at": time.Now().UTC().Format(time.RFC3339),
	}
	return k.sendEvent(ctx, "subscriptions.deleted", event)
}

func (k *KafkaProducerImpl) sendEvent(ctx context.Context, topic string, event interface{}) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(jsonData),
	}

	partition, offset, err := k.producer.SendMessage(message)
	if err != nil {
		k.logger.Error().Err(err).Str("topic", topic).Msg("Failed to send Kafka message")
		return err
	}

	k.logger.Info().Str("topic", topic).Int32("partition", partition).Int64("offset", offset).Msg("Kafka message sent successfully")
	return nil
}

func (k *KafkaProducerImpl) Close() error {
	if k.producer == nil {
		return nil
	}
	if err := k.producer.Close(); err != nil {
		k.logger.Error().Err(err).Msg("Failed to close Kafka producer")
		return err
	}
	k.logger.Info().Msg("Kafka producer closed successfully")
	return nil
}
