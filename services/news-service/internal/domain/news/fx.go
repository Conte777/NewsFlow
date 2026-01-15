package news

import (
	"go.uber.org/fx"

	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news/delivery/kafka"
	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news/repository/http_clients/subscription"
	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news/repository/postgres"
	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news/usecase/buissines"
)

// Module provides news domain dependencies
var Module = fx.Module(
	"news",
	fx.Provide(
		postgres.NewNewsRepository,
		postgres.NewDeliveredNewsRepository,
		subscription.NewClient,
		buissines.NewUseCase,
		kafka.NewHandlers,
	),
)
