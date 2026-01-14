package channel

import (
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/channel/delivery/kafka"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/channel/deps"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/channel/repository/memory"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/channel/usecase/business"
	"go.uber.org/fx"
)

// Module provides channel domain components for fx DI
var Module = fx.Module("channel",
	fx.Provide(
		memory.NewRepository,
		business.NewUseCase,
		kafka.NewSubscriptionHandler,
		// Provide SubscriptionEventHandler interface for Kafka consumer
		func(h *kafka.SubscriptionHandler) deps.SubscriptionEventHandler {
			return h
		},
	),
)
