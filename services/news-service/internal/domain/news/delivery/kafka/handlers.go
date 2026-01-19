package kafka

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/dto"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/usecase/buissines"
)

// Handlers handles Kafka messages for news domain
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

// HandleNewsReceived handles news received event from account service
func (h *Handlers) HandleNewsReceived(ctx context.Context, message []byte) error {
	var event dto.NewsReceivedEvent
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error().Err(err).
			Str("raw_message", string(message)).
			Msg("Failed to unmarshal news received event")
		return err
	}

	h.logger.Info().
		Str("channel_id", event.ChannelID).
		Str("channel_name", event.ChannelName).
		Int("message_id", event.MessageID).
		Msg("Processing news received event")

	req := &dto.ProcessNewsRequest{
		ChannelID:   event.ChannelID,
		ChannelName: event.ChannelName,
		MessageID:   event.MessageID,
		Content:     event.Content,
		MediaURLs:   event.MediaURLs,
	}

	_, err := h.uc.ProcessNewsReceived(ctx, req)
	if err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", event.ChannelID).
			Int("message_id", event.MessageID).
			Msg("Failed to process news received event")
		return err
	}

	h.logger.Info().
		Str("channel_id", event.ChannelID).
		Int("message_id", event.MessageID).
		Msg("News received event processed successfully")

	return nil
}

// HandleDeliveryConfirmation handles delivery confirmation from bot service
func (h *Handlers) HandleDeliveryConfirmation(ctx context.Context, message []byte) error {
	var event struct {
		NewsID uint  `json:"newsId"`
		UserID int64 `json:"userId"`
	}

	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error().Err(err).
			Str("raw_message", string(message)).
			Msg("Failed to unmarshal delivery confirmation event")
		return err
	}

	h.logger.Info().
		Uint("news_id", event.NewsID).
		Int64("user_id", event.UserID).
		Msg("Processing delivery confirmation")

	if err := h.uc.MarkAsDelivered(ctx, event.NewsID, event.UserID); err != nil {
		h.logger.Error().Err(err).
			Uint("news_id", event.NewsID).
			Int64("user_id", event.UserID).
			Msg("Failed to mark news as delivered")
		return err
	}

	return nil
}

// HandleNewsDeleted handles news deleted event from account service
func (h *Handlers) HandleNewsDeleted(ctx context.Context, message []byte) error {
	var event dto.NewsDeletedEvent
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error().Err(err).
			Str("raw_message", string(message)).
			Msg("Failed to unmarshal news deleted event")
		return err
	}

	h.logger.Info().
		Str("channel_id", event.ChannelID).
		Ints("message_ids", event.MessageIDs).
		Msg("Processing news deleted event")

	if err := h.uc.ProcessNewsDeleted(ctx, event.ChannelID, event.MessageIDs); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", event.ChannelID).
			Ints("message_ids", event.MessageIDs).
			Msg("Failed to process news deleted event")
		return err
	}

	h.logger.Info().
		Str("channel_id", event.ChannelID).
		Ints("message_ids", event.MessageIDs).
		Msg("News deleted event processed successfully")

	return nil
}

// HandleNewsEdited handles news edited event from account service
func (h *Handlers) HandleNewsEdited(ctx context.Context, message []byte) error {
	var event dto.NewsEditedEvent
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error().Err(err).
			Str("raw_message", string(message)).
			Msg("Failed to unmarshal news edited event")
		return err
	}

	h.logger.Info().
		Str("channel_id", event.ChannelID).
		Int("message_id", event.MessageID).
		Msg("Processing news edited event")

	if err := h.uc.ProcessNewsEdited(ctx, &event); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", event.ChannelID).
			Int("message_id", event.MessageID).
			Msg("Failed to process news edited event")
		return err
	}

	h.logger.Info().
		Str("channel_id", event.ChannelID).
		Int("message_id", event.MessageID).
		Msg("News edited event processed successfully")

	return nil
}
