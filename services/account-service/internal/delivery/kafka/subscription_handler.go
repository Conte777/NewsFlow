package kafka

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// SubscriptionHandler handles subscription events from Kafka
//
// This handler receives subscription events (created/deleted) from the
// subscription service via Kafka and delegates the actual subscription
// operations to the AccountUseCase.
type SubscriptionHandler struct {
	accountUseCase domain.AccountUseCase
	logger         zerolog.Logger
}

// NewSubscriptionHandler creates a new subscription event handler
//
// Parameters:
//   - accountUseCase: Use case for account operations (subscribe/unsubscribe)
//   - logger: Logger for monitoring and debugging
//
// Returns:
//   - Pointer to SubscriptionHandler that implements domain.SubscriptionEventHandler
func NewSubscriptionHandler(accountUseCase domain.AccountUseCase, logger zerolog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{
		accountUseCase: accountUseCase,
		logger:         logger,
	}
}

// HandleSubscriptionCreated handles subscription created events
//
// When a user subscribes to a channel in the subscription service, this handler
// receives the event and subscribes the account service to that channel.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - userID: ID of the user who subscribed (for logging)
//   - channelID: Telegram channel ID to subscribe to
//   - channelName: Human-readable channel name
//
// Returns:
//   - nil on success
//   - error if subscription fails (network issues, invalid channel, etc.)
//
// Implementation notes:
//   - This handler is idempotent - subscribing to an already subscribed channel is safe
//   - Errors are logged and returned for retry by the Kafka consumer
//   - The userID is currently only used for logging (ACC-2.7)
func (h *SubscriptionHandler) HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Str("channel_name", channelName).
		Msg("Handling subscription created event")

	// Delegate to use case for actual subscription logic
	if err := h.accountUseCase.SubscribeToChannel(ctx, channelID, channelName); err != nil {
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
//
// When a user unsubscribes from a channel in the subscription service, this handler
// receives the event and unsubscribes the account service from that channel.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - userID: ID of the user who unsubscribed (for logging)
//   - channelID: Telegram channel ID to unsubscribe from
//
// Returns:
//   - nil on success
//   - error if unsubscription fails (network issues, channel not found, etc.)
//
// Implementation notes:
//   - This handler is idempotent - unsubscribing from a non-subscribed channel is safe
//   - Errors are logged and returned for retry by the Kafka consumer
//   - The userID is currently only used for logging (ACC-2.7)
func (h *SubscriptionHandler) HandleSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Handling subscription deleted event")

	// Delegate to use case for actual unsubscription logic
	if err := h.accountUseCase.UnsubscribeFromChannel(ctx, channelID); err != nil {
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
