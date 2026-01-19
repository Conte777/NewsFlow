package main

import (
	"context"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/Conte777/NewsFlow/services/news-service/config"
	"github.com/Conte777/NewsFlow/services/news-service/internal/app"
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
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info().
				Str("service", cfg.Service.Name).
				Str("port", cfg.Service.Port).
				Msg("Starting news service")

			logger.Info().Msg("Database connected successfully")
			logger.Info().Msg("News service initialized successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info().Msg("Shutting down news service...")

			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}

			logger.Info().Msg("News service stopped")
			return nil
		},
	})
}
