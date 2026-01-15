package app

import (
	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/database"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/grpc"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/kafka"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/logger"
	"go.uber.org/fx"
)

func CreateApp() fx.Option {
	return fx.Options(
		fx.Provide(config.Out),

		logger.Module,
		database.Module,
		kafka.Module,

		domain.Module,

		grpc.Module,
	)
}
