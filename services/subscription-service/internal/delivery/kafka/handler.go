package kafka

import (
	"context"
	"errors"
	"strconv"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/events"

	"github.com/rs/zerolog"
)

type SubscriptionEventHandler struct {
	usecase domain.SubscriptionUseCase
	logger  zerolog.Logger
}

func NewSubscriptionEventHandler(
	usecase domain.SubscriptionUseCase,
	logger zerolog.Logger,
) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *SubscriptionEventHandler) HandleSubscriptionCreated(
	ctx context.Context,
	event *events.SubscriptionEvent,
) error {
	if event.UserID == "" || event.ChannelID == "" || event.ChannelName == "" {
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Str("channel_name", event.ChannelName).
			Msg("subscription created event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID, cannot convert to int64")
		return err
	}

	// --- Создаём подписку через usecase ---
	err = h.usecase.CreateSubscription(ctx, userID, event.ChannelID, event.ChannelName)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionAlreadyExists) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription already exists, skipping creation")
			return nil // идемпотентно: успех
		}
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("failed to create subscription")
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription successfully created")

	return nil
}

func (h *SubscriptionEventHandler) HandleSubscriptionDeleted(
	ctx context.Context,
	event *events.SubscriptionEvent,
) error {
	if event.UserID == "" || event.ChannelID == "" {
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("subscription deleted event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID, cannot convert to int64")
		return err
	}

	// --- Удаляем подписку через usecase ---
	err = h.usecase.DeleteSubscription(ctx, userID, event.ChannelID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription does not exist, skipping deletion")
			return nil // идемпотентно: успех
		}
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("failed to delete subscription")
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription successfully deleted")

	return nil
}
