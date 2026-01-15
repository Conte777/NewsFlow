package subscription

import (
	"context"

	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/consts"
	subkafka "github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/delivery/kafka"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/deps"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/repository/postgres"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/usecase/buissines"
	kafkaInfra "github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/kafka"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var Module = fx.Module(
	"subscription",
	fx.Provide(
		NewRepository,
		NewKafkaProducerAdapter,
		NewUseCase,
		subkafka.NewEventHandler,
	),
	fx.Invoke(registerKafkaConsumer),
)

func NewRepository(db *gorm.DB) deps.SubscriptionRepository {
	return postgres.NewRepository(db)
}

func NewKafkaProducerAdapter(adapter *kafkaInfra.ProducerAdapter) deps.KafkaProducer {
	return adapter
}

func NewUseCase(
	repo deps.SubscriptionRepository,
	kafkaProducer deps.KafkaProducer,
	logger zerolog.Logger,
) deps.SubscriptionUseCase {
	return buissines.NewUseCase(repo, kafkaProducer, logger)
}

func registerKafkaConsumer(
	lc fx.Lifecycle,
	cfg *config.KafkaConfig,
	handler *subkafka.EventHandler,
	log zerolog.Logger,
) error {
	consumer, err := kafkaInfra.NewKafkaConsumer(
		cfg.Brokers,
		cfg.GroupID,
		consts.ConsumerTopics,
		handler,
		log,
	)
	if err != nil {
		return err
	}

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			consumer.Start(consumerCtx)
			log.Info().Msg("kafka consumer started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("stopping kafka consumer...")
			cancelConsumer()
			return consumer.Close()
		},
	})

	return nil
}
