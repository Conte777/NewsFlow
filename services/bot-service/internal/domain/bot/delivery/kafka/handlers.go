// Package kafka contains Kafka delivery handlers
package kafka

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/dto"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/entities"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/usecase/buissines"
)

// Handlers contains Kafka message handlers
type Handlers struct {
	uc     *buissines.UseCase
	logger zerolog.Logger
}

// NewHandlers creates new Kafka handlers
func NewHandlers(uc *buissines.UseCase, logger zerolog.Logger) *Handlers {
	return &Handlers{
		uc:     uc,
		logger: logger,
	}
}

// HandleNewsDelivery handles news delivery events from Kafka
func (h *Handlers) HandleNewsDelivery(ctx context.Context, data []byte) error {
	var event dto.NewsDeliveryEvent
	if err := json.Unmarshal(data, &event); err != nil {
		h.logger.Error().Err(err).Str("data", string(data)).Msg("Failed to unmarshal news delivery event")
		return err
	}

	h.logger.Info().
		Str("news_id", event.ID).
		Int64("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("Processing news delivery event")

	news := &entities.NewsMessage{
		ID:          event.ID,
		UserID:      event.UserID,
		ChannelID:   event.ChannelID,
		ChannelName: event.ChannelName,
		Content:     event.Content,
		MediaURLs:   event.MediaURLs,
	}

	if err := h.uc.SendNews(ctx, news); err != nil {
		h.logger.Error().Err(err).Str("news_id", event.ID).Msg("Failed to send news to user")
		return err
	}

	h.logger.Info().Str("news_id", event.ID).Int64("user_id", event.UserID).Msg("News delivered successfully")
	return nil
}
