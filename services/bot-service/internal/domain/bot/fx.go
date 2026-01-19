// Package bot contains the bot domain module
package bot

import (
	"context"

	tgbot "github.com/go-telegram/bot"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
	kafkaDelivery "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/delivery/kafka"
	telegramDelivery "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/delivery/telegram"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/deps"
	kafkaRepo "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/repository/kafka"
	subscriptionClient "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/repository/grpc_clients/subscription"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/usecase/buissines"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/workers"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/infrastructure/telegram"
)

// Module provides bot domain components for fx dependency injection
var Module = fx.Module("bot",
	// Repository
	fx.Provide(kafkaRepo.NewProducer),
	fx.Provide(provideSubscriptionRepository),

	// UseCase
	fx.Provide(buissines.NewUseCase),

	// Delivery - Telegram (needs raw bot from infrastructure)
	fx.Provide(provideTelegramHandlers),
	fx.Provide(telegramDelivery.NewRouter),

	// Delivery - Kafka
	fx.Provide(kafkaDelivery.NewHandlers),

	// Workers
	workers.Module,

	// Wire cyclic dependency and register routes
	fx.Invoke(wireAndRegister),
)

// provideTelegramHandlers creates Telegram handlers with raw bot
func provideTelegramHandlers(uc *buissines.UseCase, bot *telegram.Bot, logger zerolog.Logger) *telegramDelivery.Handlers {
	return telegramDelivery.NewHandlers(uc, bot.Raw(), logger)
}

// wireAndRegister resolves cyclic dependency and registers routes
func wireAndRegister(
	lc fx.Lifecycle,
	uc *buissines.UseCase,
	handlers *telegramDelivery.Handlers,
	router *telegramDelivery.Router,
	bot *telegram.Bot,
	kafkaCfg *config.KafkaConfig,
	logger zerolog.Logger,
) {
	// Handlers implements deps.TelegramSender interface
	// This resolves the cyclic dependency: UseCase -> TelegramSender <- Handlers -> UseCase
	uc.SetSender(handlers)

	// Register Telegram command routes
	router.RegisterRoutes(bot.Raw())

	// Create and register rejection consumer (Saga workflow)
	// Handlers implements TelegramSender, so we can use it to send rejection notifications
	rejectionConsumer := workers.NewRejectionConsumer(
		kafkaCfg,
		handlers,
		logger.With().Str("component", "rejection-consumer").Logger(),
	)

	// Create and register confirmation consumer (Saga workflow)
	// Handlers implements TelegramSender, so we can use it to send confirmation notifications
	confirmationConsumer := workers.NewConfirmationConsumer(
		kafkaCfg,
		handlers,
		logger.With().Str("component", "confirmation-consumer").Logger(),
	)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			rejectionConsumer.Start(ctx)
			confirmationConsumer.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := rejectionConsumer.Stop(); err != nil {
				return err
			}
			return confirmationConsumer.Stop()
		},
	})
}

// provideSubscriptionRepository creates subscription repository (gRPC client)
func provideSubscriptionRepository(cfg *config.GRPCConfig, logger zerolog.Logger) (deps.SubscriptionRepository, error) {
	return subscriptionClient.NewClient(cfg.SubscriptionServiceAddr, logger)
}

// RawBotProvider extracts raw tgbot.Bot from infrastructure Bot
type RawBotProvider struct {
	fx.Out
	Bot *tgbot.Bot
}
