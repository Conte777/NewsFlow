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
		Int("media_count", len(event.MediaURLs)).
		Msg("Processing batch news delivery event")

	// Download files once for all users
	var downloadedFiles []*entities.DownloadedFile
	if len(event.MediaURLs) > 0 {
		var err error
		downloadedFiles, err = h.uc.DownloadFiles(ctx, event.MediaURLs)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to download media files")
			// Continue without media - will try URL fallback in UseCase
		} else {
			h.logger.Info().Int("downloaded_count", len(downloadedFiles)).Msg("Media files downloaded for batch delivery")
		}
	}

	var lastErr error
	successCount := 0

	for _, userID := range event.UserIDs {
		news := &entities.NewsMessage{
			ID:              event.NewsID,
			UserID:          userID,
			ChannelID:       event.ChannelID,
			ChannelName:     event.ChannelName,
			MessageID:       event.MessageID,
			Content:         event.Content,
			MediaURLs:       event.MediaURLs,
			DownloadedFiles: downloadedFiles, // Pre-downloaded files for all users
			Timestamp:       event.Timestamp,
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

	// Files will be garbage collected after this function returns
	return lastErr
}

// HandleNewsDelete handles news delete events from Kafka
func (h *Handlers) HandleNewsDelete(ctx context.Context, data []byte) error {
	var event dto.NewsDeleteEvent
	if err := json.Unmarshal(data, &event); err != nil {
		h.logger.Error().Err(err).Str("data", string(data)).Msg("Failed to unmarshal news delete event")
		return err
	}

	h.logger.Info().
		Uint("news_id", event.NewsID).
		Int("users_count", len(event.UserIDs)).
		Msg("Processing news delete event")

	if err := h.uc.DeleteNews(ctx, event.NewsID, event.UserIDs); err != nil {
		h.logger.Error().Err(err).
			Uint("news_id", event.NewsID).
			Msg("Failed to delete news from user chats")
		return err
	}

	h.logger.Info().
		Uint("news_id", event.NewsID).
		Msg("News delete event processed successfully")

	return nil
}

// HandleNewsEdit handles news edit events from Kafka
func (h *Handlers) HandleNewsEdit(ctx context.Context, data []byte) error {
	var event dto.NewsEditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		h.logger.Error().Err(err).Str("data", string(data)).Msg("Failed to unmarshal news edit event")
		return err
	}

	h.logger.Info().
		Uint("news_id", event.NewsID).
		Int("users_count", len(event.UserIDs)).
		Str("channel_name", event.ChannelName).
		Msg("Processing news edit event")

	if err := h.uc.EditNews(ctx, &event); err != nil {
		h.logger.Error().Err(err).
			Uint("news_id", event.NewsID).
			Msg("Failed to edit news in user chats")
		return err
	}

	h.logger.Info().
		Uint("news_id", event.NewsID).
		Msg("News edit event processed successfully")

	return nil
}
