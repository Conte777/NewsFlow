package main

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/Conte777/NewsFlow/services/news-service/config"
	"github.com/Conte777/NewsFlow/services/news-service/internal/app"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/delivery/kafka"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func main() {
	fx.New(
		app.CreateApp(),
		fx.Invoke(run),
	).Run()
}

func run(
	lc fx.Lifecycle,
	cfg *config.Config,
	db *gorm.DB,
	logger zerolog.Logger,
	handlers *kafka.Handlers,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info().
				Str("service", cfg.Service.Name).
				Str("port", cfg.Service.Port).
				Msg("Starting news service")

			logger.Info().Msg("Database connected successfully")

			// TODO: Start Kafka consumer
			// go consumer.ConsumeNewsReceived(ctx, handlers.HandleNewsReceived)

			logger.Info().Msg("News service initialized successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info().Msg("Shutting down news service...")

			// Close database connection
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}

			// TODO: Close Kafka connections

			logger.Info().Msg("News service stopped")
			return nil
		},
	})
}
