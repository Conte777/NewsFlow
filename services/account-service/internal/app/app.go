package app

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/account"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure"
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
		news.Module,
	)
}
