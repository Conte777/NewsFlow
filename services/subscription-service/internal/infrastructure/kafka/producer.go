package kafka

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	logger   zerolog.Logger
}

func NewKafkaProducer(brokers []string, logger zerolog.Logger) (*KafkaProducer, error) {

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 5
	config.Producer.Retry.Backoff = 100
	config.Producer.RequiredAcks = sarama.WaitForAll

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create Kafka SyncProducer")
		return nil, err
	}

	logger.Info().Msg("Kafka SyncProducer successfully initialized")

	return &KafkaProducer{
		producer: producer,
		logger:   logger,
	}, nil
}

func (p *KafkaProducer) Close() error {
	if p.producer == nil {
		p.logger.Info().Msg("Kafka producer already closed or not initialized")
		return nil
	}

	err := p.producer.Close()
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to close Kafka producer")
		return err
	}

	p.logger.Info().Msg("Kafka producer successfully closed")
	return nil
}

func (p *KafkaProducer) NotifyAccountService(ctx context.Context, event *SubscriptionEvent) error {
	topic := getTopicByEventType(event.Type)

	bytes, err := json.Marshal(event)
	if err != nil {
		p.logger.Error().
			Err(err).
			Str("event_type", event.Type).
			Msg("failed to marshal subscription event")
		return err
	}

	key := sarama.StringEncoder(event.UserID)

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   key,
		Value: sarama.ByteEncoder(bytes),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error().
			Err(err).
			Str("topic", topic).
			Str("user_id", event.UserID).
			Msg("failed to send subscription event to kafka")
		return err
	}

	p.logger.Info().
		Str("topic", topic).
		Int32("partition", partition).
		Int64("offset", offset).
		Str("user_id", event.UserID).
		Msg("subscription event sent to kafka")

	return nil
}

func getTopicByEventType(eventType string) string {
	switch eventType {
	case "subscription_created":
		return "subscription.created"
	case "subscription_updated":
		return "subscription.updated"
	case "subscription_cancelled":
		return "subscription.cancelled"
	default:
		return "subscription.unknown"
	}
}

type SubscriptionEvent struct {
	Type        string `json:"type"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Active      bool   `json:"active"`
}
