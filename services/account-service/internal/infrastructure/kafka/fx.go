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
	fx.Provide(
		NewKafkaProducerFx,
		NewSagaProducerFx,
	),
	fx.Invoke(registerKafkaConsumer),
)

// NewKafkaProducerFx creates a Kafka producer for news events
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

// NewSagaProducerFx creates a Kafka producer for Saga events
func NewSagaProducerFx(
	lc fx.Lifecycle,
	kafkaCfg *config.KafkaConfig,
	logger zerolog.Logger,
) (deps.SagaProducer, error) {
	producer, err := NewSagaProducer(kafkaCfg, logger.With().Str("component", "saga-producer").Logger())
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
	sagaHandler deps.SagaEventHandler,
	logger zerolog.Logger,
) error {
	// Saga topics only
	topics := []string{
		kafkaCfg.TopicSubscriptionPending,
		kafkaCfg.TopicUnsubscriptionPending,
	}

	consumer, err := NewKafkaConsumer(
		kafkaCfg.Brokers,
		kafkaCfg.GroupID,
		topics,
		sagaHandler,
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
