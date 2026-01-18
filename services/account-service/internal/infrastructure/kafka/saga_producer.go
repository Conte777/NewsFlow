package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

// SagaProducer sends Saga result events to subscription-service
type SagaProducer struct {
	producer sarama.SyncProducer
	config   *config.KafkaConfig
	logger   zerolog.Logger
}

// NewSagaProducer creates a new Kafka producer for Saga events
func NewSagaProducer(cfg *config.KafkaConfig, logger zerolog.Logger) (*SagaProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Retry.Max = 5
	saramaConfig.Producer.Retry.Backoff = 500 * time.Millisecond
	saramaConfig.Producer.Timeout = 10 * time.Second
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Version = sarama.V2_6_0_0
	saramaConfig.ClientID = "account-service-saga-producer"

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create Saga Kafka producer")
		return nil, err
	}

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Msg("Saga Kafka producer initialized")

	return &SagaProducer{
		producer: producer,
		config:   cfg,
		logger:   logger,
	}, nil
}

// SendSubscriptionActivated sends subscription.activated event
func (p *SagaProducer) SendSubscriptionActivated(ctx context.Context, userID int64, channelID string) error {
	event := NewSubscriptionActivatedEvent(userID, channelID)
	return p.sendEvent(ctx, p.config.TopicSubscriptionActivated, event)
}

// SendSubscriptionFailed sends subscription.failed event
func (p *SagaProducer) SendSubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error {
	event := NewSubscriptionFailedEvent(userID, channelID, reason)
	return p.sendEvent(ctx, p.config.TopicSubscriptionFailed, event)
}

// SendUnsubscriptionCompleted sends unsubscription.completed event
func (p *SagaProducer) SendUnsubscriptionCompleted(ctx context.Context, userID int64, channelID string) error {
	event := NewUnsubscriptionCompletedEvent(userID, channelID)
	return p.sendEvent(ctx, p.config.TopicUnsubscriptionCompleted, event)
}

// SendUnsubscriptionFailed sends unsubscription.failed event
func (p *SagaProducer) SendUnsubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error {
	event := NewUnsubscriptionFailedEvent(userID, channelID, reason)
	return p.sendEvent(ctx, p.config.TopicUnsubscriptionFailed, event)
}

func (p *SagaProducer) sendEvent(ctx context.Context, topic string, event *ResultEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		p.logger.Error().Err(err).
			Str("topic", topic).
			Msg("failed to marshal Saga event")
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(event.UserID),
		Value: sarama.ByteEncoder(bytes),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error().Err(err).
			Str("topic", topic).
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("failed to send Saga event")
		return err
	}

	p.logger.Info().
		Str("topic", topic).
		Str("type", event.Type).
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("Saga event sent")

	return nil
}

// Close closes the Kafka producer
func (p *SagaProducer) Close() error {
	if p.producer == nil {
		return nil
	}

	if err := p.producer.Close(); err != nil {
		p.logger.Error().Err(err).Msg("failed to close Saga producer")
		return err
	}

	p.logger.Info().Msg("Saga producer closed")
	return nil
}
