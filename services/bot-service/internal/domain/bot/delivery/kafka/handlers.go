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

// HandleNewsDelivery handles news delivery events from Kafka (batch format)
func (h *Handlers) HandleNewsDelivery(ctx context.Context, data []byte) error {
	var event dto.NewsDeliveryEvent
	if err := json.Unmarshal(data, &event); err != nil {
		h.logger.Error().Err(err).Str("data", string(data)).Msg("Failed to unmarshal news delivery event")
		return err
	}

	h.logger.Info().
		Uint("news_id", event.NewsID).
		Int("users_count", len(event.UserIDs)).
		Str("channel_id", event.ChannelID).
		Msg("Processing batch news delivery event")

	var lastErr error
	successCount := 0

	for _, userID := range event.UserIDs {
		news := &entities.NewsMessage{
			ID:          event.NewsID,
			UserID:      userID,
			ChannelID:   event.ChannelID,
			ChannelName: event.ChannelName,
			Content:     event.Content,
			MediaURLs:   event.MediaURLs,
			Timestamp:   event.Timestamp,
		}

		if err := h.uc.SendNews(ctx, news); err != nil {
			h.logger.Error().Err(err).
				Uint("news_id", event.NewsID).
				Int64("user_id", userID).
				Msg("Failed to send news to user")
			lastErr = err
			continue
		}

		successCount++
		h.logger.Debug().
			Uint("news_id", event.NewsID).
			Int64("user_id", userID).
			Msg("News delivered to user")
	}

	h.logger.Info().
		Uint("news_id", event.NewsID).
		Int("success_count", successCount).
		Int("total_count", len(event.UserIDs)).
		Msg("Batch news delivery completed")

	return lastErr
}
