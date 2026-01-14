package app

import (
	"context"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/account"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/channel"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/news"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure"
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
