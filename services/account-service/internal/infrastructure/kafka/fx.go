package kafka

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Module provides Kafka producer and consumer for fx DI
var Module = fx.Module("kafka",
	fx.Provide(NewKafkaProducerFx),
	fx.Invoke(registerKafkaConsumer),
)

// NewKafkaProducerFx creates a Kafka producer for fx DI
func NewKafkaProducerFx(
	lc fx.Lifecycle,
	kafkaCfg *config.KafkaConfig,
	logger zerolog.Logger,
) (domain.KafkaProducer, error) {
	producer, err := NewKafkaProducer(ProducerConfig{
		Brokers: kafkaCfg.Brokers,
		Topic:   kafkaCfg.TopicNewsReceived,
		Logger:  logger,
	})
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return producer.Close()
		},
	})

	return producer, nil
}

// registerKafkaConsumer creates and registers Kafka consumer with lifecycle hooks
func registerKafkaConsumer(
	lc fx.Lifecycle,
	kafkaCfg *config.KafkaConfig,
	handler deps.SubscriptionEventHandler,
	logger zerolog.Logger,
) error {
	topics := []string{
		kafkaCfg.TopicSubscriptionsCreated,
		kafkaCfg.TopicSubscriptionsDeleted,
	}

	consumer, err := NewKafkaConsumer(
		kafkaCfg.Brokers,
		kafkaCfg.GroupID,
		topics,
		handler,
		logger.With().Str("component", "kafka-consumer").Logger(),
	)
	if err != nil {
		return err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			consumer.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return consumer.Close()
		},
	})

	return nil
}
