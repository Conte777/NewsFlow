package database

import (
	"context"

	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var Module = fx.Module(
	"database",
	fx.Provide(NewDB),
)

func NewDB(lc fx.Lifecycle, cfg *config.DatabaseConfig, log zerolog.Logger) (*gorm.DB, error) {
	db, err := NewPostgresDB(*cfg)
	if err != nil {
		return nil, err
	}

	if err := RunMigrations(db, *cfg); err != nil {
		return nil, err
	}

	log.Info().Msg("database connected and migrations completed")

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("closing database connection...")
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		},
	})

	return db, nil
}
