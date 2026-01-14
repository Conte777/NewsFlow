package app

import (
	"go.uber.org/fx"

	"github.com/yourusername/telegram-news-feed/news-service/config"
	"github.com/yourusername/telegram-news-feed/news-service/internal/domain"
	"github.com/yourusername/telegram-news-feed/news-service/internal/infrastructure/database"
	"github.com/yourusername/telegram-news-feed/news-service/internal/infrastructure/logger"
)

// CreateApp creates the fx application with all dependencies
func CreateApp() fx.Option {
	return fx.Options(
		fx.Provide(config.Out),
		fx.Provide(logger.NewLogger),
		fx.Provide(database.NewPostgresDB),
		domain.Module,
	)
}
