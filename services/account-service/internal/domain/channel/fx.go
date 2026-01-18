package channel

import (
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/delivery/kafka"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/repository/postgres"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/usecase/business"
	"go.uber.org/fx"
)

// Module provides channel domain components for fx DI
var Module = fx.Module("channel",
	fx.Provide(
		postgres.NewRepository,
		business.NewUseCase,
		kafka.NewSagaHandler,
		// Provide SagaEventHandler interface for Kafka consumer (Saga workflow)
		func(h *kafka.SagaHandler) deps.SagaEventHandler {
			return h
		},
	),
)
