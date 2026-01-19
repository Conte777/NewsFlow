package kafka

import (
	"context"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/news-service/config"
	kafkaHandlers "github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/delivery/kafka"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/deps"
)

var Module = fx.Module("kafka",
	fx.Provide(NewProducerFx),
	fx.Provide(NewConsumer),
	fx.Invoke(registerConsumerLifecycle),
	fx.Invoke(registerDeletedConsumerLifecycle),
	fx.Invoke(registerEditedConsumerLifecycle),
)

func NewProducerFx(
	lc fx.Lifecycle,
	cfg *config.KafkaConfig,
	logger zerolog.Logger,
) (deps.KafkaProducer, error) {
	producer, err := NewProducer(cfg, logger)
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

func registerConsumerLifecycle(
	lc fx.Lifecycle,
	cfg *config.KafkaConfig,
	handlers *kafkaHandlers.Handlers,
	logger zerolog.Logger,
) {
	consumer := NewConsumer(cfg, handlers, logger.With().Str("component", "kafka-consumer").Logger())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			consumer.Start()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return consumer.Stop()
		},
	})
}

func registerDeletedConsumerLifecycle(
	lc fx.Lifecycle,
	cfg *config.KafkaConfig,
	handlers *kafkaHandlers.Handlers,
	logger zerolog.Logger,
) {
	consumer := NewNewsDeletedConsumer(cfg, handlers, logger.With().Str("component", "kafka-deleted-consumer").Logger())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			consumer.Start()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return consumer.Stop()
		},
	})
}

func registerEditedConsumerLifecycle(
	lc fx.Lifecycle,
	cfg *config.KafkaConfig,
	handlers *kafkaHandlers.Handlers,
	logger zerolog.Logger,
) {
	consumer := NewNewsEditedConsumer(cfg, handlers, logger.With().Str("component", "kafka-edited-consumer").Logger())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			consumer.Start()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return consumer.Stop()
		},
	})
}
