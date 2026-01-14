package kafka

import (
	"context"
	"fmt"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/channel/usecase/business"
	"github.com/rs/zerolog"
)

// SubscriptionHandler handles subscription events from Kafka
type SubscriptionHandler struct {
	channelUseCase *business.UseCase
	logger         zerolog.Logger
}

// NewSubscriptionHandler creates a new subscription event handler
func NewSubscriptionHandler(channelUseCase *business.UseCase, logger zerolog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{
		channelUseCase: channelUseCase,
		logger:         logger,
	}
}

// HandleSubscriptionCreated handles subscription created events
func (h *SubscriptionHandler) HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Str("channel_name", channelName).
		Msg("Handling subscription created event")

	if err := h.channelUseCase.Subscribe(ctx, channelID, channelName); err != nil {
		h.logger.Error().
			Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Str("channel_name", channelName).
			Msg("Failed to subscribe to channel")

		return fmt.Errorf("failed to subscribe to channel %s: %w", channelID, err)
	}

	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Str("channel_name", channelName).
		Msg("Successfully subscribed to channel")

	return nil
}

// HandleSubscriptionDeleted handles subscription deleted events
func (h *SubscriptionHandler) HandleSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Handling subscription deleted event")

	if err := h.channelUseCase.Unsubscribe(ctx, channelID); err != nil {
		h.logger.Error().
			Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to unsubscribe from channel")

		return fmt.Errorf("failed to unsubscribe from channel %s: %w", channelID, err)
	}

	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Successfully unsubscribed from channel")

	return nil
}
