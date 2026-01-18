// Package kafka contains Kafka repository implementations
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/deps"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/entities"
)

// Producer implements deps.SubscriptionEventProducer
type Producer struct {
	producer sarama.SyncProducer
	config   *config.KafkaConfig
	logger   zerolog.Logger
}

// NewProducer creates a new Kafka producer that implements deps.SubscriptionEventProducer
func NewProducer(cfg *config.KafkaConfig, logger zerolog.Logger) (deps.SubscriptionEventProducer, error) {
	brokers := cfg.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9093"}
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 3
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	logger.Info().Strs("brokers", brokers).Msg("Kafka producer initialized successfully")

	return &Producer{
		producer: producer,
		config:   cfg,
		logger:   logger,
	}, nil
}

// SendSubscriptionCreated sends subscription.requested event to Kafka (Saga flow)
func (p *Producer) SendSubscriptionCreated(ctx context.Context, subscription *entities.Subscription) error {
	event := map[string]interface{}{
		"type":         "subscription_requested",
		"user_id":      strconv.FormatInt(subscription.UserID, 10),
		"channel_id":   subscription.ChannelID,
		"channel_name": subscription.ChannelName,
		"timestamp":    time.Now().Unix(),
	}
	return p.sendEvent(ctx, p.config.TopicSubscriptionRequested, event)
}

// SendSubscriptionDeleted sends unsubscription.requested event to Kafka (Saga flow)
func (p *Producer) SendSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error {
	event := map[string]interface{}{
		"type":       "unsubscription_requested",
		"user_id":    strconv.FormatInt(userID, 10),
		"channel_id": channelID,
		"timestamp":  time.Now().Unix(),
	}
	return p.sendEvent(ctx, p.config.TopicUnsubscriptionRequested, event)
}

func (p *Producer) sendEvent(ctx context.Context, topic string, event interface{}) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(jsonData),
	}

	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		p.logger.Error().Err(err).Str("topic", topic).Msg("Failed to send Kafka message")
		return err
	}

	p.logger.Info().
		Str("topic", topic).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("Kafka message sent successfully")

	return nil
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	if p.producer == nil {
		return nil
	}
	if err := p.producer.Close(); err != nil {
		p.logger.Error().Err(err).Msg("Failed to close Kafka producer")
		return err
	}
	p.logger.Info().Msg("Kafka producer closed successfully")
	return nil
}
