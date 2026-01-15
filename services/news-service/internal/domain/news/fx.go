package news

import (
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/delivery/kafka"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/repository/grpc_clients/subscription"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/repository/postgres"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/usecase/buissines"
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
