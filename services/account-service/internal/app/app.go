package app

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/account"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news/handlers"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/cache"
	"go.uber.org/fx"
)

// CreateApp creates the fx application options
func CreateApp() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Out,
			context.Background,
		),
		infrastructure.Module,
		// Domain modules
		account.Module,
		channel.Module,
		cache.Module,    // Must be after channel.Module (depends on ChannelRepository)
		handlers.Module, // Must be after cache.Module (depends on MessageIDCache)
		news.Module,
		qrauth.Module,
	)
}
