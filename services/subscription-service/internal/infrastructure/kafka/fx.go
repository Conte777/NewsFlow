package kafka

import (
	"context"

	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"kafka",
	fx.Provide(
		NewProducer,
		NewAdapter,
	),
)

func NewProducer(lc fx.Lifecycle, cfg *config.KafkaConfig, log zerolog.Logger) (*KafkaProducer, error) {
	producer, err := NewKafkaProducer(cfg.Brokers, log)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("closing kafka producer...")
			return producer.Close()
		},
	})

	return producer, nil
}

func NewAdapter(producer *KafkaProducer) *ProducerAdapter {
	return NewProducerAdapter(producer)
}
