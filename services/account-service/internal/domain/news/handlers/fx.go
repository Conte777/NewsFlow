package handlers

import (
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	channeldeps "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/metrics"
)

// Module provides news handlers for fx DI
var Module = fx.Module("news-handlers",
	fx.Provide(NewNewsUpdateHandlerFx),
)

// NewNewsUpdateHandlerFx creates NewsUpdateHandler for fx DI
func NewNewsUpdateHandlerFx(
	channelRepo channeldeps.ChannelRepository,
	msgIDCache channeldeps.MessageIDCache,
	producer domain.KafkaProducer,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *NewsUpdateHandler {
	return NewNewsUpdateHandler(channelRepo, msgIDCache, producer, logger, m)
}
