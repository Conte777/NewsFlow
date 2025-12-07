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

	// --- Валидация обязательных полей ---
	if event.UserID == "" {
		h.logger.Error().Msg("subscription created event missing UserID")
		return errors.New("missing UserID")
	}
	if event.ChannelID == "" {
		h.logger.Error().Msg("subscription created event missing ChannelID")
		return errors.New("missing ChannelID")
	}
	if event.ChannelName == "" {
		h.logger.Error().Msg("subscription created event missing ChannelName")
		return errors.New("missing ChannelName")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID, cannot convert to int64")
		return err
	}

	// Подготовка модели для бизнес-логики
	sub := domain.Subscription{
		UserID:      userID,
		ChannelID:   event.ChannelID,
		ChannelName: event.ChannelName,
	}

	// --- Основная логика ---
	if err := h.usecase.CreateSubscription(ctx, sub.UserID, sub.ChannelID, sub.ChannelName); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("failed to create subscription")

		return err
	}

	// --- Успех ---
	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription successfully created")

	return nil
}
