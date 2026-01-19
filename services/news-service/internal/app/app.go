package app

import (
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/news-service/config"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/news-service/internal/infrastructure/database"
	"github.com/Conte777/NewsFlow/services/news-service/internal/infrastructure/kafka"
	"github.com/Conte777/NewsFlow/services/news-service/internal/infrastructure/logger"
)

// CreateApp creates the fx application with all dependencies
func CreateApp() fx.Option {
	return fx.Options(
		fx.Provide(config.Out),
		fx.Provide(logger.NewLogger),
		fx.Provide(database.NewPostgresDB),
		kafka.Module,
		domain.Module,
	)
}
