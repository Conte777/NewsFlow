package news

import (
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/news/usecase/business"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/news/workers"
	"go.uber.org/fx"
)

// Module provides news domain components for fx DI
var Module = fx.Module("news",
	fx.Provide(business.NewUseCase),
	workers.Module,
)
