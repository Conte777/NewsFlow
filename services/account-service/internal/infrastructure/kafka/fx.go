package kafka

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Module provides Kafka producer for fx DI
var Module = fx.Module("kafka",
	fx.Provide(NewKafkaProducerFx),
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
