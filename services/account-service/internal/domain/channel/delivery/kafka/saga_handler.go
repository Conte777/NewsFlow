package kafka

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/usecase/business"
	"github.com/rs/zerolog"
)

// SagaHandler handles Saga workflow events from subscription-service
type SagaHandler struct {
	channelUseCase *business.UseCase
	sagaProducer   deps.SagaProducer
	logger         zerolog.Logger
}

// NewSagaHandler creates a new Saga event handler
func NewSagaHandler(
	channelUseCase *business.UseCase,
	sagaProducer deps.SagaProducer,
	logger zerolog.Logger,
) *SagaHandler {
	return &SagaHandler{
		channelUseCase: channelUseCase,
		sagaProducer:   sagaProducer,
		logger:         logger,
	}
}

// HandleSubscriptionPending handles subscription.pending events
// Attempts to subscribe to a Telegram channel and sends result back
func (h *SagaHandler) HandleSubscriptionPending(ctx context.Context, userID int64, channelID, channelName string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Str("channel_name", channelName).
		Msg("Handling subscription.pending event")

	err := h.channelUseCase.Subscribe(ctx, channelID, channelName)
	if err != nil {
		h.logger.Error().
			Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to subscribe to channel, sending subscription.failed")

		if sendErr := h.sagaProducer.SendSubscriptionFailed(ctx, userID, channelID, err.Error()); sendErr != nil {
			h.logger.Error().
				Err(sendErr).
				Int64("user_id", userID).
				Str("channel_id", channelID).
				Msg("Failed to send subscription.failed event")
			return sendErr
		}

		return nil // Don't return original error - we handled it by sending failure event
	}

	// Send success notification
	if err := h.sagaProducer.SendSubscriptionActivated(ctx, userID, channelID); err != nil {
		h.logger.Error().
			Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to send subscription.activated event")
		return err
	}

	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Successfully subscribed to channel, sent subscription.activated")

	return nil
}

// HandleUnsubscriptionPending handles unsubscription.pending events
// Attempts to unsubscribe from a Telegram channel and sends result back
func (h *SagaHandler) HandleUnsubscriptionPending(ctx context.Context, userID int64, channelID string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Handling unsubscription.pending event")

	err := h.channelUseCase.Unsubscribe(ctx, channelID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to unsubscribe from channel, sending unsubscription.failed")

		if sendErr := h.sagaProducer.SendUnsubscriptionFailed(ctx, userID, channelID, err.Error()); sendErr != nil {
			h.logger.Error().
				Err(sendErr).
				Int64("user_id", userID).
				Str("channel_id", channelID).
				Msg("Failed to send unsubscription.failed event")
			return sendErr
		}

		return nil // Don't return original error - we handled it by sending failure event
	}

	// Send success notification
	if err := h.sagaProducer.SendUnsubscriptionCompleted(ctx, userID, channelID); err != nil {
		h.logger.Error().
			Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to send unsubscription.completed event")
		return err
	}

	h.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Successfully unsubscribed from channel, sent unsubscription.completed")

	return nil
}
