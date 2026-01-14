package database

import (
	"context"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides database components for fx dependency injection
var Module = fx.Module("database",
	fx.Provide(NewPostgresDBFx),
)

// NewPostgresDBFx creates a PostgreSQL database connection with fx lifecycle management
func NewPostgresDBFx(
	lc fx.Lifecycle,
	cfg *config.DatabaseConfig,
	logger zerolog.Logger,
) (*gorm.DB, error) {
	db, err := NewPostgresDB(cfg)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := RunMigrations(db, cfg); err != nil {
		logger.Warn().Err(err).Msg("Failed to run migrations")
		// Don't fail startup if migrations fail - they might already be applied
	} else {
		logger.Info().Msg("Database migrations completed successfully")
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info().Msg("Closing database connection")
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		},
	})

	logger.Info().
		Str("host", cfg.Host).
		Str("port", cfg.Port).
		Str("database", cfg.DBName).
		Msg("Database connected successfully")

	return db, nil
}
