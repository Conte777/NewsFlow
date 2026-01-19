package database

import (
	"context"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
)

var Module = fx.Module("database",
	fx.Provide(NewPostgresDBWithLifecycle),
)

func NewPostgresDBWithLifecycle(
	lc fx.Lifecycle,
	cfg *config.DatabaseConfig,
	logger zerolog.Logger,
) (*gorm.DB, error) {
	db, err := NewPostgresDB(cfg)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				logger.Error().Err(err).Msg("Failed to get underlying sql.DB")
				return err
			}
			logger.Info().Msg("Closing database connection")
			return sqlDB.Close()
		},
	})

	logger.Info().
		Str("host", cfg.Host).
		Str("port", cfg.Port).
		Str("database", cfg.Name).
		Msg("Database connected")

	return db, nil
}
